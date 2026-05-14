package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
)

type Client struct {
	cfg    config.SlackConfig
	logger *logx.Logger
	http   *http.Client
}

func NewClient(cfg config.SlackConfig, logger *logx.Logger) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) SendChatText(ctx context.Context, chatID string, text string) error {
	if !c.cfg.Enabled {
		c.logger.Info("skip slack proactive send because channel is disabled", "chat_id", chatID)
		return nil
	}
	return c.postMessage(ctx, chatID, "", text)
}

func (c *Client) ReplyInThread(ctx context.Context, channelID string, threadTS string, text string) error {
	if !c.cfg.Enabled {
		c.logger.Info("skip slack reply because channel is disabled", "channel_id", channelID)
		return nil
	}
	return c.postMessage(ctx, channelID, threadTS, text)
}

func (c *Client) postMessage(ctx context.Context, channelID string, threadTS string, text string) error {
	payload := map[string]any{
		"channel": channelID,
		"text":    text,
	}
	if strings.TrimSpace(threadTS) != "" {
		payload["thread_ts"] = threadTS
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/api/chat.postMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.BotToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send slack request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("read slack response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("slack http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("decode slack response: %w", err)
	}
	if !result.OK {
		if strings.TrimSpace(result.Error) == "" {
			result.Error = "unknown_error"
		}
		return fmt.Errorf("slack api error: %s", result.Error)
	}
	return nil
}

func (c *Client) OpenSocketModeConnection(ctx context.Context) (string, error) {
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/api/apps.connections.open"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(""))
	if err != nil {
		return "", fmt.Errorf("create slack socket mode request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.AppToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("send slack socket mode request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("read slack socket mode response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("slack socket mode http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		OK    bool   `json:"ok"`
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode slack socket mode response: %w", err)
	}
	if !result.OK {
		if strings.TrimSpace(result.Error) == "" {
			result.Error = "unknown_error"
		}
		return "", fmt.Errorf("slack socket mode api error: %s", result.Error)
	}
	if strings.TrimSpace(result.URL) == "" {
		return "", fmt.Errorf("slack socket mode url is empty")
	}

	return result.URL, nil
}
