package notify

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultOneBotReverseAddr = ":8090"
	oneBotPingInterval       = 30 * time.Second
	oneBotReadTimeout        = 90 * time.Second
)

var oneBotEchoSeq atomic.Uint64

// OneBotWSConfig 描述 OneBot v11 WebSocket 连接配置。
type OneBotWSConfig struct {
	// ForwardURL 是正向 WebSocket 地址，后端主动连接 OneBot；可留空。
	ForwardURL string
	// ReverseEnabled 控制是否启动反向 WebSocket 服务端。
	ReverseEnabled bool
	ReverseAddr    string
	ReversePath    string
	// AccessToken 用于正向和反向 WebSocket 鉴权。
	AccessToken string
}

// oneBotWSManager 统一管理正向客户端和反向服务端。发送时优先使用反向 WS，
// 其次正向 WS；两者都未连接时由 HTTPSender 回退到 HTTP API。
type oneBotWSManager struct {
	forward *oneBotForwardClient
	reverse *oneBotReverseServer
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func newOneBotWSManager(cfg OneBotWSConfig) (*oneBotWSManager, error) {
	manager := &oneBotWSManager{}
	if forwardURL := strings.TrimSpace(cfg.ForwardURL); forwardURL != "" {
		manager.forward = newOneBotForwardClient(forwardURL, strings.TrimSpace(cfg.AccessToken))
	}
	var reverseErr error
	if cfg.ReverseEnabled {
		addr := strings.TrimSpace(cfg.ReverseAddr)
		if addr == "" {
			addr = defaultOneBotReverseAddr
		}
		path := strings.TrimSpace(cfg.ReversePath)
		if path == "" {
			reverseErr = errors.New("反向 WebSocket 路径不能为空")
		} else {
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			server, err := newOneBotReverseServer(addr, path, strings.TrimSpace(cfg.AccessToken))
			if err != nil {
				// 反向监听失败时仍保留已配置的正向 WebSocket。
				reverseErr = err
			} else {
				manager.reverse = server
			}
		}
	}
	return manager, reverseErr
}

func (m *oneBotWSManager) start() {
	if m == nil || (m.forward == nil && m.reverse == nil) {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	if m.reverse != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			_ = m.reverse.serve()
		}()
	}
	if m.forward != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.forward.run(ctx)
		}()
	}
}

func (m *oneBotWSManager) close() {
	if m == nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	if m.reverse != nil {
		m.reverse.close()
	}
	if m.forward != nil {
		m.forward.close()
	}
	m.wg.Wait()
}

func (m *oneBotWSManager) reverseAddr() string {
	if m == nil || m.reverse == nil {
		return ""
	}
	return m.reverse.addr
}

// sendPrivateMessage 返回 handled=false 表示当前没有可用的 WS 连接。
func (m *oneBotWSManager) sendPrivateMessage(ctx context.Context, qq int64, message string) (bool, error) {
	if m == nil {
		return false, nil
	}
	if conn := m.reverse.active(); conn != nil {
		return true, conn.sendPrivateMessage(ctx, qq, message)
	}
	if conn := m.forward.current(); conn != nil {
		return true, conn.sendPrivateMessage(ctx, qq, message)
	}
	return false, nil
}

type oneBotForwardClient struct {
	url   string
	token string
	mu    sync.RWMutex
	conn  *oneBotWSConn
}

func newOneBotForwardClient(url string, token string) *oneBotForwardClient {
	return &oneBotForwardClient{url: url, token: token}
}

func (f *oneBotForwardClient) run(ctx context.Context) {
	const maxBackoff = 30 * time.Second
	backoff := time.Second
	for ctx.Err() == nil {
		headers := http.Header{}
		if f.token != "" {
			headers.Set("Authorization", "Bearer "+f.token)
		}
		dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
		conn, resp, err := dialer.DialContext(ctx, f.url, headers)
		if err != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			if !sleepWithContext(ctx, backoff) {
				return
			}
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}
		backoff = time.Second
		wsConn := newOneBotWSConn(conn)
		f.setConn(wsConn)
		wsConn.readLoop()
		f.setConn(nil)
		wsConn.close()
		if !sleepWithContext(ctx, time.Second) {
			return
		}
	}
}

func (f *oneBotForwardClient) current() *oneBotWSConn {
	if f == nil {
		return nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.conn
}

func (f *oneBotForwardClient) setConn(conn *oneBotWSConn) {
	f.mu.Lock()
	f.conn = conn
	f.mu.Unlock()
}

func (f *oneBotForwardClient) close() {
	if conn := f.current(); conn != nil {
		conn.close()
	}
}

type oneBotReverseServer struct {
	addr     string
	path     string
	token    string
	listener net.Listener
	server   *http.Server
	mu       sync.RWMutex
	conns    []*oneBotWSConn
	closed   bool
}

func newOneBotReverseServer(addr string, path string, token string) (*oneBotReverseServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("监听 OneBot 反向 WebSocket 失败：%w", err)
	}
	server := &oneBotReverseServer{
		addr:     listener.Addr().String(),
		path:     path,
		token:    token,
		listener: listener,
	}
	mux := http.NewServeMux()
	mux.HandleFunc(path, server.handle)
	server.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return server, nil
}

func (s *oneBotReverseServer) serve() error {
	return s.server.Serve(s.listener)
}

func (s *oneBotReverseServer) handle(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	upgrader := websocket.Upgrader{
		// OneBot 客户端通常不携带 Origin；带 Origin 的浏览器请求要求已配置 token。
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Origin") == "" || strings.TrimSpace(s.token) != ""
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	wsConn := newOneBotWSConn(conn)
	if !s.add(wsConn) {
		wsConn.close()
		return
	}
	defer func() {
		s.remove(wsConn)
		wsConn.close()
	}()
	wsConn.readLoop()
}

func (s *oneBotReverseServer) authorized(r *http.Request) bool {
	if strings.TrimSpace(s.token) == "" {
		return true
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		if secureEqual(strings.TrimSpace(auth[len("bearer "):]), s.token) {
			return true
		}
	}
	return secureEqual(strings.TrimSpace(r.URL.Query().Get("access_token")), s.token)
}

func secureEqual(a string, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *oneBotReverseServer) add(conn *oneBotWSConn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.conns = append(s.conns, conn)
	return true
}

func (s *oneBotReverseServer) remove(conn *oneBotWSConn) {
	s.mu.Lock()
	for i, item := range s.conns {
		if item == conn {
			s.conns = append(s.conns[:i], s.conns[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
}

func (s *oneBotReverseServer) active() *oneBotWSConn {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.conns) == 0 {
		return nil
	}
	return s.conns[len(s.conns)-1]
}

func (s *oneBotReverseServer) close() {
	if s == nil {
		return
	}
	if s.server != nil {
		_ = s.server.Close()
	}
	s.mu.Lock()
	s.closed = true
	conns := append([]*oneBotWSConn(nil), s.conns...)
	s.conns = nil
	s.mu.Unlock()
	for _, conn := range conns {
		conn.close()
	}
}

// oneBotWSConn 在一条 WebSocket 连接上复用 echo 关联请求与响应。
type oneBotWSConn struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex
	mu        sync.Mutex
	pending   map[string]chan oneBotResponse
	closed    bool
	closeOnce sync.Once
}

func newOneBotWSConn(conn *websocket.Conn) *oneBotWSConn {
	return &oneBotWSConn{
		conn:    conn,
		pending: make(map[string]chan oneBotResponse),
	}
}

func (c *oneBotWSConn) readLoop() {
	c.conn.SetReadLimit(1 << 20)
	_ = c.conn.SetReadDeadline(time.Now().Add(oneBotReadTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(oneBotReadTimeout))
	})
	stopPing := make(chan struct{})
	defer close(stopPing)
	go c.pingLoop(stopPing)

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(oneBotReadTimeout))
		var response oneBotResponse
		if err := json.Unmarshal(data, &response); err != nil {
			continue
		}
		// 没有 echo 的帧是 OneBot 事件或心跳，忽略即可。
		if response.Echo == "" {
			continue
		}
		c.deliver(response)
	}
}

func (c *oneBotWSConn) pingLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(oneBotPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c.writeMu.Lock()
			err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
			c.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (c *oneBotWSConn) deliver(response oneBotResponse) {
	c.mu.Lock()
	ch, ok := c.pending[response.Echo]
	if ok {
		delete(c.pending, response.Echo)
	}
	c.mu.Unlock()
	if ok {
		ch <- response
	}
}

func (c *oneBotWSConn) call(ctx context.Context, action string, params any) (json.RawMessage, error) {
	echo := fmt.Sprintf("biligo-%d", oneBotEchoSeq.Add(1))
	ch := make(chan oneBotResponse, 1)
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, errors.New("OneBot WebSocket 连接已关闭")
	}
	c.pending[echo] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, echo)
		c.mu.Unlock()
	}()

	data, err := json.Marshal(map[string]any{
		"action": action,
		"params": params,
		"echo":   echo,
	})
	if err != nil {
		return nil, err
	}
	c.writeMu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()
	if err != nil {
		return nil, err
	}

	select {
	case response := <-ch:
		if err := checkOneBotResponse(response, nil); err != nil {
			return nil, err
		}
		return response.Data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *oneBotWSConn) sendPrivateMessage(ctx context.Context, qq int64, message string) error {
	_, err := c.call(ctx, "send_private_msg", map[string]any{
		"user_id": qq,
		"message": message,
	})
	return err
}

func (c *oneBotWSConn) close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		for echo, ch := range c.pending {
			delete(c.pending, echo)
			select {
			case ch <- oneBotResponse{Status: "failed", Retcode: -1, Wording: "OneBot WebSocket 连接已关闭"}:
			default:
			}
		}
		c.mu.Unlock()
		c.writeMu.Lock()
		_ = c.conn.Close()
		c.writeMu.Unlock()
	})
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
