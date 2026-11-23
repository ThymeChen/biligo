package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fdcs99/biligo/internal/model"
)

const (
	defaultPushPlusURL = "http://www.pushplus.plus/send"
	defaultBarkBaseURL = "https://api.day.app"

	// OneBot v11 HTTP API 发送私聊消息的接口路径。
	oneBotSendPrivateMsgPath = "/send_private_msg"
)

type Sender interface {
	Send(ctx context.Context, notification model.Notification, title string, content string) error
}

type HTTPSender struct {
	client      *http.Client
	PushPlusURL string
	BarkBaseURL string

	// OneBotAPIURL 是 OneBot v11 HTTP API 服务地址（不含接口路径）。
	OneBotAPIURL string
	// OneBotAccessToken 是 OneBot 服务端配置的 access_token，可留空。
	OneBotAccessToken string

	wsMu sync.Mutex
	// oneBotWS 管理 OneBot WebSocket 连接；未启用时为 nil。
	oneBotWS *oneBotWSManager
}

func NewHTTPSender(client *http.Client) *HTTPSender {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &HTTPSender{
		client:      client,
		PushPlusURL: defaultPushPlusURL,
		BarkBaseURL: defaultBarkBaseURL,
	}
}

// EnableOneBotWS 启用 OneBot WebSocket。反向 WS 监听失败会返回错误，
// 但已配置的正向 WS 仍会启动。
func (s *HTTPSender) EnableOneBotWS(cfg OneBotWSConfig) error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	if s.oneBotWS != nil {
		return nil
	}
	manager, err := newOneBotWSManager(cfg)
	if manager.forward != nil || manager.reverse != nil {
		manager.start()
		s.oneBotWS = manager
	}
	return err
}

func (s *HTTPSender) Close() {
	if s == nil {
		return
	}
	s.wsMu.Lock()
	manager := s.oneBotWS
	s.oneBotWS = nil
	s.wsMu.Unlock()
	if manager != nil {
		manager.close()
	}
}

// OneBotReverseWSAddr 返回反向 WS 实际监听地址，仅启用后非空。
func (s *HTTPSender) OneBotReverseWSAddr() string {
	manager := s.currentOneBotWS()
	if manager == nil {
		return ""
	}
	return manager.reverseAddr()
}

func (s *HTTPSender) currentOneBotWS() *oneBotWSManager {
	if s == nil {
		return nil
	}
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	return s.oneBotWS
}

func (s *HTTPSender) Send(ctx context.Context, notification model.Notification, title string, content string) error {
	switch model.NormalizeNotificationProvider(notification.Provider) {
	case model.NotificationProviderPushPlus:
		return s.sendPushPlus(ctx, notification, title, content)
	case model.NotificationProviderBark:
		return s.sendBark(ctx, notification, title, content)
	case model.NotificationProviderQQ:
		return s.sendQQ(ctx, notification, title, content)
	default:
		return fmt.Errorf("不支持的通知类型：%s", notification.Provider)
	}
}

func (s *HTTPSender) sendPushPlus(ctx context.Context, notification model.Notification, title string, content string) error {
	token := strings.TrimSpace(notification.Config["token"])
	if token == "" {
		return errors.New("PushPlus Token 不能为空")
	}
	endpoint := strings.TrimSpace(s.PushPlusURL)
	if endpoint == "" {
		endpoint = defaultPushPlusURL
	}
	payload := map[string]string{
		"token":   token,
		"title":   title,
		"content": content,
	}
	return s.postJSON(ctx, endpoint, payload)
}

func (s *HTTPSender) sendBark(ctx context.Context, notification model.Notification, title string, content string) error {
	token := strings.TrimSpace(notification.Config["token"])
	if token == "" {
		return errors.New("Bark Token 或完整推送地址不能为空")
	}
	base := strings.TrimSpace(s.BarkBaseURL)
	if base == "" {
		base = defaultBarkBaseURL
	}
	endpoint := buildBarkURL(base, token, title, content)
	payload := map[string]string{
		"icon":   "https://raw.githubusercontent.com/mikumifa/biliTickerBuy/refs/heads/main/assets/icon.ico",
		"group":  "Biligo",
		"url":    "https://mall.bilibili.com/neul/index.html?page=box_me&noTitleBar=1",
		"sound":  "telegraph",
		"level":  "critical",
		"volume": "10",
	}
	return s.postJSON(ctx, endpoint, payload)
}

func (s *HTTPSender) sendQQ(ctx context.Context, notification model.Notification, title string, content string) error {
	qq, ok := model.ParseQQNumber(notification.Config["qq"])
	if !ok {
		return errors.New("QQ 号不能为空或格式不正确")
	}
	message := title + "\n" + content
	// WebSocket 已连接时优先使用 WS，避免与 HTTP 双发。
	if ws := s.currentOneBotWS(); ws != nil {
		if handled, err := ws.sendPrivateMessage(ctx, qq, message); handled {
			return err
		}
	}
	base := strings.TrimSpace(s.OneBotAPIURL)
	if base == "" {
		if s.currentOneBotWS() != nil {
			return errors.New("OneBot WebSocket 未连接且未配置 HTTP API，无法发送 QQ 通知")
		}
		return errors.New("未配置 OneBot HTTP API 或 WebSocket，无法发送 QQ 通知")
	}
	payload := map[string]any{
		"user_id": qq,
		"message": message,
	}
	return s.postOneBot(ctx, strings.TrimRight(base, "/")+oneBotSendPrivateMsgPath, payload)
}

// postOneBot 调用 OneBot v11 HTTP API。除 HTTP 状态码外，还需检查响应中的 retcode，
// 因为 OneBot 在 HTTP 200 时仍可能返回业务错误。
func (s *HTTPSender) postOneBot(ctx context.Context, endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(s.OneBotAccessToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("OneBot 接口返回 %d：%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var result oneBotResponse
	payloadBody := bytes.TrimSpace(data)
	if len(payloadBody) == 0 {
		// 部分实现成功时不返回响应体。
		return nil
	}
	if err := json.Unmarshal(payloadBody, &result); err != nil {
		return fmt.Errorf("OneBot 响应无法解析：%s", truncateBody(payloadBody))
	}
	return checkOneBotResponse(result, payloadBody)
}

// oneBotResponse 是 OneBot v11 HTTP/WS 动作响应的公共结构。
type oneBotResponse struct {
	Status  string          `json:"status"`
	Retcode int             `json:"retcode"`
	Wording string          `json:"wording"`
	Msg     string          `json:"msg"`
	Data    json.RawMessage `json:"data"`
	Echo    string          `json:"echo"`
}

func checkOneBotResponse(response oneBotResponse, raw []byte) error {
	// retcode=0 表示成功；异步动作以 status=async、retcode=1 表示已受理。
	success := response.Retcode == 0 || (response.Status == "async" && response.Retcode == 1)
	if success && (response.Status == "" || response.Status == "ok" || response.Status == "async") {
		return nil
	}
	detail := strings.TrimSpace(response.Wording)
	if detail == "" {
		detail = strings.TrimSpace(response.Msg)
	}
	if detail == "" && len(raw) > 0 {
		detail = truncateBody(raw)
	}
	if detail == "" {
		detail = strings.TrimSpace(response.Status)
	}
	if detail == "" {
		detail = "未知错误"
	}
	return fmt.Errorf("OneBot 返回错误（retcode=%d）：%s", response.Retcode, detail)
}

func truncateBody(data []byte) string {
	const limit = 300
	text := strings.TrimSpace(string(data))
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit]) + "..."
	}
	return text
}

func buildBarkURL(base string, token string, title string, content string) string {
	escapedTitle := url.PathEscape(title)
	escapedContent := url.PathEscape(content)
	if parsed, err := url.Parse(token); err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		return strings.TrimRight(token, "/") + "/" + escapedTitle + "/" + escapedContent
	}
	return strings.TrimRight(base, "/") + "/" + strings.Trim(token, "/") + "/" + escapedTitle + "/" + escapedContent
}

func (s *HTTPSender) postJSON(ctx context.Context, endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("通知接口返回 %d：%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}
