package weixin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/session"
	"rm_ai_agent/internal/store"
)

const (
	wsEndpoint = "wss://openws.work.weixin.qq.com"
)

type Handler struct {
	cfg      config.WeixinConfig
	store    store.Store
	sessions *session.Service
	client   *Client
	logger   *logx.Logger
}

func NewHandler(cfg config.WeixinConfig, store store.Store, sessions *session.Service, client *Client, logger *logx.Logger) *Handler {
	return &Handler{cfg: cfg, store: store, sessions: sessions, client: client, logger: logger}
}

func (h *Handler) StartLongConnection(ctx context.Context) error {
	if !h.cfg.Enabled {
		return nil
	}

	h.logger.Info("starting weixin long connection", "bot_id", mask(h.cfg.BotID))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := h.runLongConnection(ctx); err != nil {
				h.logger.Error("weixin long connection error, reconnecting in 5s", "error", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (h *Handler) runLongConnection(ctx context.Context) error {
	h.logger.Info("weixin connecting to websocket", "url", wsEndpoint)

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, _, err := dialer.Dial(wsEndpoint, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	defer conn.Close()

	h.logger.Info("weixin websocket connected, sending subscribe request")

	// 发送订阅请求
	if err := h.sendSubscribe(conn); err != nil {
		return fmt.Errorf("send subscribe: %w", err)
	}

	// 等待订阅结果
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read subscribe response: %w", err)
	}
	h.logger.Info("weixin received subscribe response", "data", string(msgBytes))

	var subscribeResp struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(msgBytes, &subscribeResp); err != nil {
		return fmt.Errorf("parse subscribe response: %w", err)
	}
	if subscribeResp.ErrCode != 0 {
		return fmt.Errorf("subscribe failed: %s", subscribeResp.ErrMsg)
	}

	h.logger.Info("weixin subscription successful")

	errCh := make(chan error, 1)
	writerCh := make(chan []byte, 10)

	// 启动 writer goroutine
	go func() {
		for {
			select {
			case msg := <-writerCh:
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					h.logger.Error("weixin write message error", "error", err)
					errCh <- err
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 启动 reader 和 ping goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pingMsg := map[string]any{
					"cmd": "ping",
					"headers": map[string]string{
						"req_id": uuid.New().String(),
					},
				}
				data, _ := json.Marshal(pingMsg)
				select {
				case writerCh <- data:
					h.logger.Debug("weixin sent ping")
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				h.logger.Error("weixin read message error", "error", err)
				errCh <- err
				return
			}
			h.logger.Info("weixin received message", "data", string(msgBytes))
			h.handleMessage(ctx, msgBytes, writerCh)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (h *Handler) sendSubscribe(conn *websocket.Conn) error {
	reqID := uuid.New().String()
	msg := map[string]any{
		"cmd": "aibot_subscribe",
		"headers": map[string]string{
			"req_id": reqID,
		},
		"body": map[string]string{
			"bot_id": h.cfg.BotID,
			"secret": h.cfg.BotSecret,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal subscribe: %w", err)
	}

	h.logger.Info("weixin sending subscribe", "req_id", reqID, "bot_id", mask(h.cfg.BotID))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write subscribe: %w", err)
	}
	return nil
}

type writerChan chan []byte

func (h *Handler) handleMessage(ctx context.Context, msgBytes []byte, writer writerChan) {
	var msg struct {
		Cmd     string `json:"cmd"`
		Headers struct {
			ReqID string `json:"req_id"`
		} `json:"headers"`
		Body json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		h.logger.Warn("weixin failed to parse message", "error", err)
		return
	}

	switch msg.Cmd {
	case "aibot_msg_callback":
		h.logger.Info("weixin message cmd", "cmd", msg.Cmd)
		h.handleMsgCallback(ctx, msg.Body, msg.Headers.ReqID, writer)
	case "aibot_event_callback":
		h.logger.Info("weixin message cmd", "cmd", msg.Cmd)
		h.handleEventCallback(ctx, msg.Body, msg.Headers.ReqID, writer)
	case "pong":
		h.logger.Debug("weixin received pong")
	case "":
		// 忽略空 cmd（通常是服务器对发送消息的确认响应 {"errcode":0,"errmsg":"ok"}）
	default:
		h.logger.Debug("weixin unknown cmd", "cmd", msg.Cmd)
	}
}

func (h *Handler) handleMsgCallback(ctx context.Context, body json.RawMessage, reqID string, writer writerChan) {
	var msg struct {
		MsgID   string `json:"msgid"`
		BotID   string `json:"aibotid"`
		ChatID  string `json:"chatid"`
		ChatType string `json:"chattype"`
		From struct {
			UserID string `json:"userid"`
		} `json:"from"`
		MsgType string `json:"msgtype"`
		Text struct {
			Content string `json:"content"`
		} `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		h.logger.Warn("weixin failed to parse msg callback body", "error", err)
		return
	}

	h.logger.Info("weixin msg callback", "msg_id", msg.MsgID, "msg_type", msg.MsgType, "content", msg.Text.Content)

	text := strings.TrimSpace(msg.Text.Content)
	if text == "" {
		return
	}

	inbound := protocol.InboundMessage{
		Channel:    protocol.ChannelWeixin,
		TenantKey:  h.cfg.BotID,
		ChatID:     msg.ChatID,
		ChatType:   msg.ChatType,
		ThreadID:   "",
		MessageID:  msg.MsgID,
		SenderID:   msg.From.UserID,
		Text:       text,
		RawJSON:    string(body),
		ReceivedAt: time.Now().UTC(),
	}

	created, err := h.store.RecordInboundMessage(ctx, inbound)
	if err != nil {
		h.logger.Error("record inbound failed", "error", err)
		return
	}
	if !created {
		h.logger.Info("ignore duplicate message", "message_id", inbound.MessageID)
		return
	}

	go h.processInbound(ctx, inbound, reqID, writer)
}

func (h *Handler) handleEventCallback(ctx context.Context, body json.RawMessage, reqID string, writer writerChan) {
	var msg struct {
		MsgID   string `json:"msgid"`
		BotID   string `json:"aibotid"`
		ChatID  string `json:"chatid"`
		ChatType string `json:"chattype"`
		From struct {
			UserID string `json:"userid"`
		} `json:"from"`
		MsgType string `json:"msgtype"`
		Event struct {
			EventType string `json:"eventtype"`
		} `json:"event"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		h.logger.Warn("weixin failed to parse event callback body", "error", err)
		return
	}

	h.logger.Info("weixin event callback", "event_type", msg.Event.EventType)

	if msg.Event.EventType == "enter_chat" {
		// 回复欢迎语
		welcomeMsg := map[string]any{
			"cmd": "aibot_respond_welcome_msg",
			"headers": map[string]string{
				"req_id": reqID,
			},
			"body": map[string]any{
				"msgtype": "text",
				"text": map[string]string{
					"content": "您好！我是智能助手，有什么可以帮您的吗？",
				},
			},
		}
		data, _ := json.Marshal(welcomeMsg)
		select {
		case writer <- data:
			h.logger.Info("weixin sent welcome message")
		default:
		}
	}
}

func (h *Handler) processInbound(ctx context.Context, inbound protocol.InboundMessage, reqID string, writer writerChan) {
	h.logger.Info("start processing inbound message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID)

	sink := &progressSink{client: h.client, reqID: reqID, writer: writer}
	response, err := h.sessions.HandleInbound(ctx, inbound, sink)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.logger.Info("inbound message interrupted", "message_id", inbound.MessageID)
			return
		}
		h.logger.Error("process inbound failed", "message_id", inbound.MessageID, "error", err)
		response = "处理消息失败，请稍后重试。"
	}

	// 发送回复（使用 aibot_respond_msg + stream 类型 + 原始 req_id）
	// 注意：使用 sink.sessionID 确保和 SendProgress 的 id 一致，这样 finish=true 会替换进度消息
	sessionID := sink.sessionID
	if sessionID == "" {
		sessionID = inbound.MessageID
	}
	replyMsg := map[string]any{
		"cmd": "aibot_respond_msg",
		"headers": map[string]string{
			"req_id": reqID,
		},
		"body": map[string]any{
			"msgtype": "stream",
			"stream": map[string]any{
				"id":     sessionID,
				"finish": true,
				"content": response,
			},
		},
	}
	data, _ := json.Marshal(replyMsg)
	select {
	case writer <- data:
		h.logger.Info("weixin sent reply", "message_id", inbound.MessageID)
	default:
		h.logger.Warn("weixin writer channel full, skip reply")
	}

	h.logger.Info("finished processing inbound message", "message_id", inbound.MessageID)
}

type progressSink struct {
	client     *Client
	reqID     string
	writer    writerChan
	sessionID string // 用于最终消息替换的 id
}

func (s *progressSink) SendProgress(ctx context.Context, session protocol.SessionRef, text string) error {
	// 记录 sessionID，供最终消息替换用
	s.sessionID = session.PublicID

	// 发送进度消息（后续会被 finish=true 的最终消息替换）
	replyMsg := map[string]any{
		"cmd": "aibot_respond_msg",
		"headers": map[string]string{
			"req_id": s.reqID,
		},
		"body": map[string]any{
			"msgtype": "stream",
			"stream": map[string]any{
				"id":     session.PublicID,
				"finish": false,
				"content": fmt.Sprintf("[会话 %s][进度] %s", session.PublicID, strings.TrimSpace(text)),
			},
		},
	}
	data, _ := json.Marshal(replyMsg)
	select {
	case s.writer <- data:
	default:
	}
	return nil
}

func (s *progressSink) SendChatText(ctx context.Context, chatID string, text string) error {
	replyMsg := map[string]any{
		"cmd": "aibot_send_msg",
		"headers": map[string]string{
			"req_id": uuid.New().String(),
		},
		"body": map[string]any{
			"chatid":  chatID,
			"chat_type": 1,
			"msgtype": "text",
			"text": map[string]string{
				"content": text,
			},
		},
	}
	data, _ := json.Marshal(replyMsg)
	select {
	case s.writer <- data:
	default:
	}
	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("weixin handler panic recovered", "panic", r)
			writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		}
	}()

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("weixin webhook called", "method", r.Method, "path", r.URL.Path)

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		h.logger.Error("weixin webhook failed to read body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var msg struct {
		MsgType string `json:"MsgType"`
		Content string `json:"Content"`
		MsgId   string `json:"MsgId"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		h.logger.Warn("weixin webhook failed to parse json", "error", err)
		writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		return
	}

	h.logger.Info("weixin webhook message", "msg_type", msg.MsgType, "content", msg.Content)

	writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
}

func shortText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 100 {
		return text
	}
	return text[:100] + "..."
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func mask(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:4] + "..." + value[len(value)-4:]
}
