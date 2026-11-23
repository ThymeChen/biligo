package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fdcs99/biligo/internal/model"
	"github.com/gorilla/websocket"
)

func TestHTTPSenderPushPlus(t *testing.T) {
	var payload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHTTPSender(server.Client())
	sender.PushPlusURL = server.URL
	err := sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderPushPlus,
		Config:   map[string]string{"token": "push-token"},
	}, "测试标题", "测试内容")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if payload["token"] != "push-token" || payload["title"] != "测试标题" || payload["content"] != "测试内容" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestHTTPSenderBarkToken(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHTTPSender(server.Client())
	sender.BarkBaseURL = server.URL
	err := sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderBark,
		Config:   map[string]string{"token": "bark-token"},
	}, "抢票成功", "请支付")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(gotPath, "/bark-token/") || !strings.Contains(gotPath, "%E6%8A%A2%E7%A5%A8%E6%88%90%E5%8A%9F") {
		t.Fatalf("unexpected bark path: %s", gotPath)
	}
}

func TestHTTPSenderBarkFullURL(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHTTPSender(server.Client())
	err := sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderBark,
		Config:   map[string]string{"token": server.URL + "/custom-key"},
	}, "Biligo", "订单创建成功")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(gotPath, "/custom-key/Biligo/") || !strings.Contains(gotPath, "%E8%AE%A2%E5%8D%95%E5%88%9B%E5%BB%BA%E6%88%90%E5%8A%9F") {
		t.Fatalf("unexpected bark full url path: %s", gotPath)
	}
}

func TestHTTPSenderQQ(t *testing.T) {
	var gotPath string
	var gotAuth string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"message_id":1}}`))
	}))
	defer server.Close()

	sender := NewHTTPSender(server.Client())
	sender.OneBotAPIURL = server.URL
	sender.OneBotAccessToken = "onebot-secret"
	err := sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderQQ,
		Config:   map[string]string{"qq": "123456789"},
	}, "抢票成功", "请支付")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotPath != oneBotSendPrivateMsgPath {
		t.Fatalf("path = %s, want %s", gotPath, oneBotSendPrivateMsgPath)
	}
	if gotAuth != "Bearer onebot-secret" {
		t.Fatalf("Authorization = %q, want Bearer token", gotAuth)
	}
	if userID, _ := payload["user_id"].(float64); int64(userID) != 123456789 {
		t.Fatalf("user_id = %v, want 123456789", payload["user_id"])
	}
	message, _ := payload["message"].(string)
	if !strings.Contains(message, "抢票成功") || !strings.Contains(message, "请支付") {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestHTTPSenderQQRejectsBadResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"failed","retcode":100,"wording":"账号未登录"}`))
	}))
	defer server.Close()

	sender := NewHTTPSender(server.Client())
	sender.OneBotAPIURL = server.URL
	err := sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderQQ,
		Config:   map[string]string{"qq": "123456789"},
	}, "测试", "内容")
	if err == nil || !strings.Contains(err.Error(), "账号未登录") {
		t.Fatalf("Send err = %v, want OneBot error detail", err)
	}
}

func TestHTTPSenderQQRequiresConfig(t *testing.T) {
	sender := NewHTTPSender(nil)
	err := sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderQQ,
		Config:   map[string]string{"qq": "123456789"},
	}, "测试", "内容")
	if err == nil {
		t.Fatal("Send err = nil, want missing OneBot URL error")
	}
	err = sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderQQ,
		Config:   map[string]string{"qq": "not-a-qq"},
	}, "测试", "内容")
	if err == nil {
		t.Fatal("Send err = nil, want invalid QQ error")
	}
}

type oneBotWSAction struct {
	Action string `json:"action"`
	Params struct {
		UserID  int64  `json:"user_id"`
		Message string `json:"message"`
	} `json:"params"`
	Echo string `json:"echo"`
}

func waitForCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func TestHTTPSenderQQForwardWebSocket(t *testing.T) {
	actionCh := make(chan oneBotWSAction, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ws-secret" {
			t.Errorf("Authorization = %q, want Bearer ws-secret", got)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var action oneBotWSAction
		if err := conn.ReadJSON(&action); err != nil {
			return
		}
		actionCh <- action
		_ = conn.WriteJSON(map[string]any{
			"status":  "ok",
			"retcode": 0,
			"echo":    action.Echo,
			"data":    map[string]any{"message_id": 1},
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	sender := NewHTTPSender(server.Client())
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	if err := sender.EnableOneBotWS(OneBotWSConfig{ForwardURL: wsURL, AccessToken: "ws-secret"}); err != nil {
		t.Fatalf("EnableOneBotWS: %v", err)
	}
	defer sender.Close()
	waitForCondition(t, func() bool { return sender.oneBotWS.forward.current() != nil })

	err := sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderQQ,
		Config:   map[string]string{"qq": "123456789"},
	}, "抢票成功", "请支付")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case action := <-actionCh:
		if action.Action != "send_private_msg" || action.Params.UserID != 123456789 {
			t.Fatalf("unexpected action: %#v", action)
		}
		if !strings.Contains(action.Params.Message, "抢票成功") || !strings.Contains(action.Params.Message, "请支付") {
			t.Fatalf("unexpected message: %q", action.Params.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for WebSocket action")
	}
}

func TestHTTPSenderQQReverseWebSocket(t *testing.T) {
	sender := NewHTTPSender(nil)
	if err := sender.EnableOneBotWS(OneBotWSConfig{
		ReverseEnabled: true,
		ReverseAddr:    "127.0.0.1:0",
		ReversePath:    "/onebot/ws",
		AccessToken:    "reverse-secret",
	}); err != nil {
		t.Fatalf("EnableOneBotWS: %v", err)
	}
	defer sender.Close()
	addr := sender.OneBotReverseWSAddr()
	if addr == "" {
		t.Fatal("reverse WebSocket address is empty")
	}

	headers := http.Header{"Authorization": []string{"Bearer reverse-secret"}}
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr+"/onebot/ws", headers)
	if err != nil {
		t.Fatalf("Dial reverse WebSocket: %v", err)
	}
	defer conn.Close()
	waitForCondition(t, func() bool { return sender.oneBotWS.reverse.active() != nil })

	actionCh := make(chan oneBotWSAction, 1)
	go func() {
		var action oneBotWSAction
		if err := conn.ReadJSON(&action); err != nil {
			return
		}
		actionCh <- action
		_ = conn.WriteJSON(map[string]any{
			"status":  "ok",
			"retcode": 0,
			"echo":    action.Echo,
			"data":    map[string]any{"message_id": 2},
		})
	}()

	err = sender.Send(context.Background(), model.Notification{
		Provider: model.NotificationProviderQQ,
		Config:   map[string]string{"qq": "987654321"},
	}, "Biligo 通知测试", "反向 WS 测试")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case action := <-actionCh:
		if action.Action != "send_private_msg" || action.Params.UserID != 987654321 {
			t.Fatalf("unexpected action: %#v", action)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for reverse WebSocket action")
	}
}

func TestHTTPSenderQQReverseWebSocketRejectsBadToken(t *testing.T) {
	sender := NewHTTPSender(nil)
	if err := sender.EnableOneBotWS(OneBotWSConfig{
		ReverseEnabled: true,
		ReverseAddr:    "127.0.0.1:0",
		ReversePath:    "/onebot/ws",
		AccessToken:    "reverse-secret",
	}); err != nil {
		t.Fatalf("EnableOneBotWS: %v", err)
	}
	defer sender.Close()

	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+sender.OneBotReverseWSAddr()+"/onebot/ws", http.Header{
		"Authorization": []string{"Bearer wrong-token"},
	})
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("Dial with wrong token should fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("response = %#v, want 401", resp)
	}
}
