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
	base := strings.TrimSpace(s.OneBotAPIURL)
	if base == "" {
		return errors.New("未配置 OneBot 服务地址，无法发送 QQ 通知")
	}
	payload := map[string]any{
		"user_id": qq,
		"message": title + "\n" + content,
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
	var result struct {
		Status  string `json:"status"`
		Retcode int    `json:"retcode"`
		Wording string `json:"wording"`
		Msg     string `json:"msg"`
	}
	payloadBody := bytes.TrimSpace(data)
	if len(payloadBody) == 0 {
		// 部分实现成功时不返回响应体。
		return nil
	}
	if err := json.Unmarshal(payloadBody, &result); err != nil {
		return fmt.Errorf("OneBot 响应无法解析：%s", truncateBody(payloadBody))
	}
	// retcode=0 表示成功；异步动作以 status=async、retcode=1 表示已受理。
	success := result.Retcode == 0 || (result.Status == "async" && result.Retcode == 1)
	if !success || (result.Status != "" && result.Status != "ok" && result.Status != "async") {
		detail := strings.TrimSpace(result.Wording)
		if detail == "" {
			detail = strings.TrimSpace(result.Msg)
		}
		if detail == "" {
			detail = truncateBody(payloadBody)
		}
		return fmt.Errorf("OneBot 返回错误（retcode=%d）：%s", result.Retcode, detail)
	}
	return nil
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
