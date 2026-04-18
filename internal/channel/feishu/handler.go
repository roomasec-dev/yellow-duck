package feishu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/core/httpserverext"
	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/session"
	"rm_ai_agent/internal/store"
)

type Handler struct {
	cfg      config.FeishuConfig
	store    store.Store
	sessions *session.Service
	client   *Client
	logger   *logx.Logger
	api      *lark.Client
	dispatch *dispatcher.EventDispatcher
}

func NewHandler(cfg config.FeishuConfig, store store.Store, sessions *session.Service, client *Client, logger *logx.Logger) (*Handler, error) {
	h := &Handler{cfg: cfg, store: store, sessions: sessions, client: client, logger: logger}
	if !cfg.Enabled {
		return h, nil
	}

	h.api = lark.NewClient(cfg.AppID, cfg.AppSecret, lark.WithLogLevel(larkcore.LogLevelInfo))
	h.dispatch = dispatcher.NewEventDispatcher(cfg.VerificationToken, cfg.EncryptKey).
		OnP2MessageReceiveV1(h.handleLongConnMessage)
	return h, nil
}

type eventEnvelope struct {
	Type      string          `json:"type"`
	Challenge string          `json:"challenge"`
	Token     string          `json:"token"`
	Schema    string          `json:"schema"`
	Header    eventHeader     `json:"header"`
	Event     json.RawMessage `json:"event"`
}

type eventHeader struct {
	EventType string `json:"event_type"`
	TenantKey string `json:"tenant_key"`
}

type messageReceiveEvent struct {
	Sender struct {
		SenderID struct {
			OpenID string `json:"open_id"`
		} `json:"sender_id"`
	} `json:"sender"`
	Message struct {
		MessageID   string `json:"message_id"`
		ChatID      string `json:"chat_id"`
		ChatType    string `json:"chat_type"`
		MessageType string `json:"message_type"`
		Content     string `json:"content"`
		ThreadID    string `json:"thread_id"`
		RootID      string `json:"root_id"`
	} `json:"message"`
}

type textContent struct {
	Text string `json:"text"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.dispatch != nil {
		httpserverext.NewEventHandlerFunc(h.dispatch, larkevent.WithLogLevel(larkcore.LogLevelInfo))(w, r)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var envelope eventEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if h.cfg.VerificationToken != "" && envelope.Token != "" && envelope.Token != h.cfg.VerificationToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if envelope.Type == "url_verification" {
		writeJSON(w, http.StatusOK, map[string]string{"challenge": envelope.Challenge})
		return
	}

	if envelope.Header.EventType != "im.message.receive_v1" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	var event messageReceiveEvent
	if err := json.Unmarshal(envelope.Event, &event); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	text := extractText(event.Message.Content)
	if strings.TrimSpace(text) == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	inbound := protocol.InboundMessage{
		Channel:    protocol.ChannelFeishu,
		TenantKey:  envelope.Header.TenantKey,
		ChatID:     event.Message.ChatID,
		ChatType:   event.Message.ChatType,
		ThreadID:   firstNonEmpty(event.Message.ThreadID, event.Message.RootID),
		MessageID:  event.Message.MessageID,
		SenderID:   event.Sender.SenderID.OpenID,
		Text:       text,
		RawJSON:    string(body),
		ReceivedAt: time.Now().UTC(),
	}

	created, err := h.store.RecordInboundMessage(r.Context(), inbound)
	if err != nil {
		h.logger.Error("record inbound failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	if !created {
		h.logger.Info("ignore duplicate webhook message", "message_id", inbound.MessageID)
		return
	}
	h.logger.Info("received webhook feishu message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID, "chat_type", inbound.ChatType, "text_preview", shortText(inbound.Text))

	go h.processInbound(inbound)
}

func (h *Handler) StartLongConnection(ctx context.Context) error {
	if !h.cfg.Enabled || h.dispatch == nil {
		return nil
	}
	cli := larkws.NewClient(h.cfg.AppID, h.cfg.AppSecret,
		larkws.WithEventHandler(h.dispatch),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)
	h.logger.Info("starting feishu long connection", "app_id", maskFeishuID(h.cfg.AppID), "mode", h.cfg.Mode)
	return cli.Start(ctx)
}

func (h *Handler) processInbound(inbound protocol.InboundMessage) {
	h.logger.Info("start processing inbound message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID, "chat_type", inbound.ChatType)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	sink := &progressSink{client: h.client, replyToMessageID: inbound.MessageID}
	response, err := h.sessions.HandleInbound(ctx, inbound, sink)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.logger.Info("inbound message interrupted", "message_id", inbound.MessageID)
			return
		}
		h.logger.Error("process inbound failed", "message_id", inbound.MessageID, "error", err)
		response = fmt.Sprintf("处理消息失败：%s", err.Error())
	}

	if err := h.client.ReplyText(ctx, inbound.MessageID, response); err != nil {
		h.logger.Error("reply feishu message failed", "message_id", inbound.MessageID, "error", err)
		return
	}
	h.logger.Info("finished processing inbound message", "message_id", inbound.MessageID)
}

func (h *Handler) handleLongConnMessage(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	msg := event.Event.Message
	msgType := derefString(msg.MessageType)
	if msgType != "text" {
		return nil
	}

	inbound := protocol.InboundMessage{
		Channel:    protocol.ChannelFeishu,
		TenantKey:  event.TenantKey(),
		ChatID:     derefString(msg.ChatId),
		ChatType:   derefString(msg.ChatType),
		ThreadID:   firstNonEmpty(derefString(msg.ThreadId), derefString(msg.RootId)),
		MessageID:  derefString(msg.MessageId),
		SenderID:   safeOpenID(event),
		Text:       extractText(derefString(msg.Content)),
		RawJSON:    marshalEvent(event),
		ReceivedAt: time.Now().UTC(),
	}
	if strings.TrimSpace(inbound.Text) == "" || strings.TrimSpace(inbound.MessageID) == "" {
		return nil
	}

	created, err := h.store.RecordInboundMessage(ctx, inbound)
	if err != nil {
		h.logger.Error("record inbound failed", "error", err)
		return err
	}
	if !created {
		h.logger.Info("ignore duplicate longconn message", "message_id", inbound.MessageID)
		return nil
	}
	h.logger.Info("received longconn feishu message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID, "chat_type", inbound.ChatType, "text_preview", shortText(inbound.Text))

	go h.processInbound(inbound)
	return nil
}

type progressSink struct {
	client           *Client
	replyToMessageID string
}

func (s *progressSink) SendImmediateReply(ctx context.Context, session protocol.SessionRef, text string) error {
	return s.client.ReplyText(ctx, s.replyToMessageID, text)
}

func (s *progressSink) SendProgress(ctx context.Context, session protocol.SessionRef, text string) error {
	return s.client.ReplyText(ctx, s.replyToMessageID, fmt.Sprintf("[会话 %s][进度] %s", session.PublicID, strings.TrimSpace(text)))
}

func extractText(raw string) string {
	var content textContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return ""
	}
	return strings.TrimSpace(content.Text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func shortText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 100 {
		return text
	}
	return text[:100] + "..."
}

func maskFeishuID(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func safeOpenID(event *larkim.P2MessageReceiveV1) string {
	if event == nil || event.Event == nil || event.Event.Sender == nil || event.Event.Sender.SenderId == nil || event.Event.Sender.SenderId.OpenId == nil {
		return ""
	}
	return *event.Event.Sender.SenderId.OpenId
}

func marshalEvent(event *larkim.P2MessageReceiveV1) string {
	body, err := json.Marshal(event)
	if err != nil {
		return ""
	}
	return string(body)
}
