package weixin

import (
	"context"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
)

type Client struct {
	cfg    config.WeixinConfig
	logger *logx.Logger
}

func NewClient(cfg config.WeixinConfig, logger *logx.Logger) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
	}
}

// ReplyText is not used in long connection mode - messages are sent via WebSocket
func (c *Client) ReplyText(ctx context.Context, messageID string, text string) error {
	c.logger.Debug("weixin ReplyText not used in longconn mode")
	return nil
}

func (c *Client) SendChatText(ctx context.Context, chatID string, text string) error {
	c.logger.Debug("weixin SendChatText not used in longconn mode")
	return nil
}
