package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fdcs99/biligo/internal/applog"
	"github.com/fdcs99/biligo/internal/model"
)

type Event struct {
	Name string
	Data []byte
}

type Hub struct {
	mu          sync.Mutex
	nextID      int
	subscribers map[int]subscriber
	taskOwners  map[string]int64
	ownerLookup func(ctx context.Context, kind string, taskID int64) (int64, error)
	logger      *applog.Logger
}

type subscriber struct {
	ownerUID int64
	ch       chan Event
}

func NewHub(logger ...*applog.Logger) *Hub {
	var selectedLogger *applog.Logger
	if len(logger) > 0 {
		selectedLogger = logger[0]
	}
	return &Hub{
		subscribers: map[int]subscriber{},
		taskOwners:  map[string]int64{},
		logger:      selectedLogger,
	}
}

func (h *Hub) SetTaskOwnerLookup(lookup func(ctx context.Context, kind string, taskID int64) (int64, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ownerLookup = lookup
}

func (h *Hub) Subscribe() (int, <-chan Event) {
	return h.SubscribeForOwner(0)
}

func (h *Hub) SubscribeForOwner(ownerUID int64) (int, <-chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID
	ch := make(chan Event, 32)
	h.subscribers[id] = subscriber{ownerUID: ownerUID, ch: ch}
	return id, ch
}

func (h *Hub) Unsubscribe(id int) {
	h.mu.Lock()
	subscriber, ok := h.subscribers[id]
	if ok {
		delete(h.subscribers, id)
		close(subscriber.ch)
	}
	h.mu.Unlock()
}

func (h *Hub) Publish(name string, payload any) {
	h.PublishForOwner(name, payload, 0)
}

func (h *Hub) PublishForOwner(name string, payload any, ownerUID int64) {
	h.logEvent(name, payload)

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	event := Event{Name: name, Data: data}

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, subscriber := range h.subscribers {
		if !h.shouldDeliverLocked(name, data, ownerUID, subscriber.ownerUID) {
			continue
		}
		select {
		case subscriber.ch <- event:
		default:
		}
	}
}

func (h *Hub) shouldDeliverLocked(name string, data []byte, publisherOwner int64, subscriberOwner int64) bool {
	if subscriberOwner == 0 {
		return true
	}
	if publisherOwner != 0 {
		return publisherOwner == subscriberOwner
	}
	owner, known := h.eventOwnerLocked(name, data)
	if !known {
		return true
	}
	return owner == subscriberOwner
}

func (h *Hub) eventOwnerLocked(name string, data []byte) (int64, bool) {
	switch name {
	case "task.updated":
		var task model.Task
		if err := json.Unmarshal(data, &task); err != nil || task.ID <= 0 {
			return 0, false
		}
		h.taskOwners[taskOwnerKey("ticket", task.ID)] = task.OwnerUID
		return task.OwnerUID, true
	case "bws.task.updated":
		var task model.BWSTask
		if err := json.Unmarshal(data, &task); err != nil || task.ID <= 0 {
			return 0, false
		}
		h.taskOwners[taskOwnerKey("bws", task.ID)] = task.OwnerUID
		return task.OwnerUID, true
	case "log.created":
		var log model.TaskLog
		if err := json.Unmarshal(data, &log); err != nil || log.TaskID <= 0 {
			return 0, false
		}
		return h.resolveTaskOwnerLocked("ticket", log.TaskID)
	case "bws.log.created":
		var log model.BWSTaskLog
		if err := json.Unmarshal(data, &log); err != nil || log.TaskID <= 0 {
			return 0, false
		}
		return h.resolveTaskOwnerLocked("bws", log.TaskID)
	case "task.deleted":
		return h.deletedEventOwnerLocked("ticket", data)
	case "bws.task.deleted":
		return h.deletedEventOwnerLocked("bws", data)
	default:
		return 0, false
	}
}

func (h *Hub) deletedEventOwnerLocked(kind string, data []byte) (int64, bool) {
	var payload struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || payload.ID <= 0 {
		return 0, false
	}
	return h.resolveTaskOwnerLocked(kind, payload.ID)
}

func (h *Hub) resolveTaskOwnerLocked(kind string, taskID int64) (int64, bool) {
	key := taskOwnerKey(kind, taskID)
	if owner, ok := h.taskOwners[key]; ok {
		return owner, true
	}
	if h.ownerLookup == nil {
		return 0, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	owner, err := h.ownerLookup(ctx, kind, taskID)
	if err != nil {
		return 0, false
	}
	h.taskOwners[key] = owner
	return owner, true
}

func taskOwnerKey(kind string, taskID int64) string {
	return fmt.Sprintf("%s:%d", kind, taskID)
}

func (h *Hub) logEvent(name string, payload any) {
	if h.logger == nil || (name != "log.created" && name != "bws.log.created") {
		return
	}
	switch value := payload.(type) {
	case model.TaskLog:
		h.logger.Log(value.Level, fmt.Sprintf("任务日志实时同步：任务 %d：%s", value.TaskID, value.Message))
	case *model.TaskLog:
		if value != nil {
			h.logger.Log(value.Level, fmt.Sprintf("任务日志实时同步：任务 %d：%s", value.TaskID, value.Message))
		}
	case model.BWSTaskLog:
		h.logger.Log(value.Level, fmt.Sprintf("BWS 任务日志实时同步：任务 %d：%s", value.TaskID, value.Message))
	case *model.BWSTaskLog:
		if value != nil {
			h.logger.Log(value.Level, fmt.Sprintf("BWS 任务日志实时同步：任务 %d：%s", value.TaskID, value.Message))
		}
	}
}
