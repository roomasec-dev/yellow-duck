package weixin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
)

type Client struct {
	cfg        config.WeixinConfig
	logger     *logx.Logger
	http       *http.Client
	accessToken string
	tokenMu     sync.RWMutex
	tokenExpire time.Time
	sender     messageSender
}

type messageSender interface {
	sendText(ctx context.Context, chatID string, text string) error
}

func NewClient(cfg config.WeixinConfig, logger *logx.Logger) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ReplyText is not used in long connection mode - messages are sent via WebSocket
func (c *Client) ReplyText(ctx context.Context, messageID string, text string) error {
	c.logger.Debug("weixin ReplyText not used in longconn mode")
	return nil
}

func (c *Client) SendChatText(ctx context.Context, chatID string, text string) error {
	if !c.cfg.Enabled {
		c.logger.Info("skip weixin proactive send because channel is disabled", "chat_id", chatID)
		return nil
	}

	// Use robot/send API for proactive sending
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/cgi-bin/robot/send?access_token=%s", c.cfg.BaseURL, token)
	body := map[string]any{
		"chatid":  chatID,
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.logger.Info("weixin sending proactive text", "chat_id", chatID, "text_preview", preview(text), "url", apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	json.Unmarshal(respBody, &result)
	if result.ErrCode != 0 {
		c.logger.Error("weixin robot send failed", "errcode", result.ErrCode, "errmsg", result.ErrMsg)
		return fmt.Errorf("weixin robot send error: %d %s", result.ErrCode, result.ErrMsg)
	}

	c.logger.Info("weixin proactive text sent successfully", "chat_id", chatID)
	return nil
}

func (c *Client) SetSender(sender messageSender) {
	c.sender = sender
}

// sendWebhookReply 通过 webhook 模式发送回复消息
func (c *Client) sendWebhookReply(ctx context.Context, toUser string, text string) error {
	if !c.cfg.Enabled {
		c.logger.Info("skip weixin reply because channel is disabled")
		return nil
	}

	c.logger.Info("sending weixin webhook reply", "to_user", toUser, "text_preview", preview(text))

	// 获取 access token
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	// 调用发送消息 API
	apiURL := fmt.Sprintf("%s/cgi-bin/message/send?access_token=%s", c.cfg.BaseURL, token)

	body := map[string]any{
		"touser": toUser,
		"msgtype": "text",
		"agentid": c.cfg.BotID,
		"text": map[string]string{
			"content": text,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("weixin api http %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("weixin api error: %d %s", result.ErrCode, result.ErrMsg)
	}

	c.logger.Info("weixin webhook reply sent successfully")
	return nil
}

// getAccessToken 获取 access token，带缓存
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpire) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	// 需要重新获取
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// 再次检查（可能其他协程已经刷新）
	if c.accessToken != "" && time.Now().Before(c.tokenExpire) {
		return c.accessToken, nil
	}

	apiURL := fmt.Sprintf("%s/cgi-bin/gettoken?corpid=%s&corpsecret=%s", c.cfg.BaseURL, c.cfg.CorpID, c.cfg.CorpSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("get token request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		c.logger.Error("weixin gettoken failed", "status_code", resp.StatusCode, "response", string(respBody), "error", err)
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("get token error: %d %s", result.ErrCode, result.ErrMsg)
	}

	c.accessToken = result.AccessToken
	// 提前 5 分钟过期
	c.tokenExpire = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	c.logger.Info("weixin access token refreshed", "expires_in", result.ExpiresIn)
	return c.accessToken, nil
}

func preview(text string) string {
	if len(text) <= 100 {
		return text
	}
	return text[:100] + "..."
}
