package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
)

type Client struct {
	cfg    config.FeishuConfig
	logger *logx.Logger
	http   *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func NewClient(cfg config.FeishuConfig, logger *logx.Logger) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) ReplyText(ctx context.Context, messageID string, text string) error {
	if !c.cfg.Enabled {
		c.logger.Info("skip feishu reply because channel is disabled", "message_id", messageID)
		return nil
	}
	c.logger.Info("replying feishu text", "message_id", messageID, "text_preview", preview(text))

	token, err := c.tenantAccessToken(ctx)
	if err != nil {
		return err
	}

	body := map[string]any{
		"content":  marshalTextContent(text),
		"msg_type": "text",
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/open-apis/im/v1/messages/" + messageID + "/reply"
	if err := c.post(ctx, url, token, body); err != nil {
		return err
	}
	c.logger.Info("replied feishu text", "message_id", messageID)
	return nil
}

func (c *Client) SendChatText(ctx context.Context, chatID string, text string) error {
	if !c.cfg.Enabled {
		c.logger.Info("skip feishu proactive send because channel is disabled", "chat_id", chatID)
		return nil
	}
	c.logger.Info("sending feishu proactive text", "chat_id", chatID, "text_preview", preview(text))
	token, err := c.tenantAccessToken(ctx)
	if err != nil {
		return err
	}
	body := map[string]any{
		"receive_id": chatID,
		"content":    marshalTextContent(text),
		"msg_type":   "text",
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/open-apis/im/v1/messages?receive_id_type=chat_id"
	if err := c.post(ctx, url, token, body); err != nil {
		return err
	}
	c.logger.Info("sent feishu proactive text", "chat_id", chatID)
	return nil
}

func (c *Client) tenantAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt.Add(-1*time.Minute)) {
		return c.accessToken, nil
	}

	payload := map[string]string{
		"app_id":     c.cfg.AppID,
		"app_secret": c.cfg.AppSecret,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal tenant token request: %w", err)
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/open-apis/auth/v3/tenant_access_token/internal"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create tenant token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("request tenant token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("feishu auth http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode tenant token response: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("get tenant token failed: %s", result.Msg)
	}

	c.accessToken = result.TenantAccessToken
	c.expiresAt = time.Now().Add(time.Duration(result.Expire) * time.Second)
	c.logger.Info("refreshed feishu tenant token", "expire_seconds", result.Expire)
	return c.accessToken, nil
}

func (c *Client) post(ctx context.Context, url string, token string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal feishu payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create feishu request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send feishu request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("feishu http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode feishu response: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("feishu api error: %s", result.Msg)
	}
	return nil
}

func marshalTextContent(text string) string {
	body, _ := json.Marshal(map[string]string{"text": text})
	return string(body)
}

func preview(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 80 {
		return text
	}
	return text[:80] + "..."
}
