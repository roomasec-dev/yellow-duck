package dingtalk

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
	cfg    config.DingtalkConfig
	logger *logx.Logger
	http   *http.Client

	mu          sync.Mutex
	tokenValue  string
	expiresAt   time.Time
}

func NewClient(cfg config.DingtalkConfig, logger *logx.Logger) *Client {
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
		c.logger.Info("skip dingtalk reply because channel is disabled", "message_id", messageID)
		return nil
	}
	c.logger.Info("replying dingtalk text", "message_id", messageID, "text_preview", preview(text))

	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}

	body := map[string]any{
		"content": text,
		"msgtype": "text",
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/robot/send"
	if err := c.post(ctx, url, token, body); err != nil {
		return err
	}
	c.logger.Info("replied dingtalk text", "message_id", messageID)
	return nil
}

func (c *Client) SendChatText(ctx context.Context, chatID string, text string) error {
	if !c.cfg.Enabled {
		c.logger.Info("skip dingtalk proactive send because channel is disabled", "chat_id", chatID)
		return nil
	}
	c.logger.Info("sending dingtalk proactive text", "chat_id", chatID, "text_preview", preview(text))

	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}

	body := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/robot/send"
	if err := c.post(ctx, url, token, body); err != nil {
		return err
	}
	c.logger.Info("sent dingtalk proactive text", "chat_id", chatID)
	return nil
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenValue != "" && time.Now().Before(c.expiresAt.Add(-1*time.Minute)) {
		return c.tokenValue, nil
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + fmt.Sprintf("/gettoken?appkey=%s&appsecret=%s", c.cfg.ClientID, c.cfg.ClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create dingtalk token request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("request dingtalk token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("dingtalk auth http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Token   string `json:"access_token"`
		Expire  int    `json:"expire_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode dingtalk token response: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("get dingtalk token failed: %s", result.ErrMsg)
	}

	c.tokenValue = result.Token
	c.expiresAt = time.Now().Add(time.Duration(result.Expire) * time.Second)
	c.logger.Info("refreshed dingtalk access token", "expire_seconds", result.Expire)
	return c.tokenValue, nil
}

func (c *Client) post(ctx context.Context, url string, token string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal dingtalk payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create dingtalk request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send dingtalk request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("dingtalk http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode dingtalk response: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("dingtalk api error: %s", result.ErrMsg)
	}
	return nil
}

func preview(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 80 {
		return text
	}
	return text[:80] + "..."
}
