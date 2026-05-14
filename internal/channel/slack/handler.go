package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/session"
	"rm_ai_agent/internal/store"
)

type Handler struct {
	cfg      config.SlackConfig
	store    store.Store
	sessions *session.Service
	client   *Client
	logger   *logx.Logger
}

func NewHandler(cfg config.SlackConfig, dataStore store.Store, sessions *session.Service, client *Client, logger *logx.Logger) *Handler {
	return &Handler{cfg: cfg, store: dataStore, sessions: sessions, client: client, logger: logger}
}

type eventEnvelope struct {
	Type      string          `json:"type"`
	Challenge string          `json:"challenge"`
	TeamID    string          `json:"team_id"`
	EventID   string          `json:"event_id"`
	Event     json.RawMessage `json:"event"`
}

type messageEvent struct {
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	User        string `json:"user"`
	Text        string `json:"text"`
	Channel     string `json:"channel"`
	ChannelType string `json:"channel_type"`
	TS          string `json:"ts"`
	ThreadTS    string `json:"thread_ts"`
	ClientMsgID string `json:"client_msg_id"`
	BotID       string `json:"bot_id"`
}

const (
	slackEventMessage    = "message"
	slackEventAppMention = "app_mention"
)

type rtmMessage struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type socketEnvelope struct {
	EnvelopeID             string          `json:"envelope_id"`
	Type                   string          `json:"type"`
	AcceptsResponsePayload bool            `json:"accepts_response_payload"`
	Payload                json.RawMessage `json:"payload"`
}

func (h *Handler) StartLongConnection(ctx context.Context) error {
	if !h.cfg.Enabled {
		return nil
	}

	h.logger.Info("starting slack long connection")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := h.runLongConnection(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				h.logger.Error("slack long connection error, reconnecting in 5s", "error", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (h *Handler) runLongConnection(ctx context.Context) error {
	wsURL, err := h.client.OpenSocketModeConnection(ctx)
	if err != nil {
		return fmt.Errorf("open slack socket mode connection: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial slack websocket: %w", err)
	}
	defer conn.Close()

	h.logger.Info("slack websocket connected")
	errCh := make(chan error, 1)
	go func() {
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			h.handleLongConnMessage(ctx, conn, msgBytes)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (h *Handler) handleLongConnMessage(ctx context.Context, conn *websocket.Conn, body []byte) {
	var msg rtmMessage
	if err := json.Unmarshal(body, &msg); err == nil {
		switch msg.Type {
		case "hello":
			h.logger.Info("slack socket mode hello received")
			return
		case "disconnect":
			h.logger.Info("slack socket mode disconnect received", "reason", msg.Reason)
			return
		}
	}

	var envelope socketEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return
	}
	if strings.TrimSpace(envelope.EnvelopeID) == "" {
		return
	}
	if err := conn.WriteJSON(map[string]string{"envelope_id": envelope.EnvelopeID}); err != nil {
		h.logger.Warn("ack slack socket envelope failed", "envelope_id", envelope.EnvelopeID, "error", err)
		return
	}
	if envelope.Type != "events_api" {
		return
	}

	var payload eventEnvelope
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return
	}
	if payload.Type != "event_callback" {
		return
	}

	var event messageEvent
	if err := json.Unmarshal(payload.Event, &event); err != nil {
		return
	}
	if event.Type != slackEventMessage && event.Type != slackEventAppMention {
		h.logger.Debug("ignore slack longconn event", "event_type", event.Type)
		return
	}
	if strings.TrimSpace(event.Subtype) != "" || strings.TrimSpace(event.BotID) != "" {
		return
	}
	if strings.TrimSpace(event.User) == "" {
		return
	}
	text := strings.TrimSpace(event.Text)
	if text == "" {
		return
	}

	messageID := strings.TrimSpace(event.ClientMsgID)
	if messageID == "" {
		messageID = fmt.Sprintf("%s:%s", event.Channel, event.TS)
	}

	inbound := protocol.InboundMessage{
		Channel:    protocol.ChannelSlack,
		TenantKey:  strings.TrimSpace(payload.TeamID),
		ChatID:     strings.TrimSpace(event.Channel),
		ChatType:   chatTypeFromLongConn(event.ChannelType, event.Channel),
		ThreadID:   strings.TrimSpace(event.ThreadTS),
		MessageID:  messageID,
		SenderID:   strings.TrimSpace(event.User),
		Text:       text,
		RawJSON:    string(body),
		ReceivedAt: time.Now().UTC(),
	}

	created, err := h.store.RecordInboundMessage(ctx, inbound)
	if err != nil {
		h.logger.Error("record slack inbound failed", "error", err)
		return
	}
	if !created {
		h.logger.Info("ignore duplicate slack message", "message_id", inbound.MessageID)
		return
	}

	replyThreadTS := strings.TrimSpace(event.ThreadTS)
	if replyThreadTS == "" {
		replyThreadTS = strings.TrimSpace(event.TS)
	}
	h.logger.Info("received slack longconn message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID, "chat_type", inbound.ChatType, "text_preview", shortText(inbound.Text))
	go h.processInbound(inbound, replyThreadTS)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.verifySignature(r, body); err != nil {
		h.logger.Warn("slack signature verification failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var envelope eventEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if envelope.Type == "url_verification" {
		writeJSON(w, http.StatusOK, map[string]string{"challenge": envelope.Challenge})
		return
	}

	if envelope.Type != "event_callback" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	var event messageEvent
	if err := json.Unmarshal(envelope.Event, &event); err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if event.Type != slackEventMessage && event.Type != slackEventAppMention {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	if strings.TrimSpace(event.BotID) != "" || strings.TrimSpace(event.Subtype) != "" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	text := strings.TrimSpace(event.Text)
	if text == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	messageID := strings.TrimSpace(event.ClientMsgID)
	if messageID == "" {
		messageID = fmt.Sprintf("%s:%s", event.Channel, event.TS)
	}

	inbound := protocol.InboundMessage{
		Channel:    protocol.ChannelSlack,
		TenantKey:  envelope.TeamID,
		ChatID:     event.Channel,
		ChatType:   chatTypeFromSlack(event.ChannelType),
		ThreadID:   strings.TrimSpace(event.ThreadTS),
		MessageID:  messageID,
		SenderID:   event.User,
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
		h.logger.Info("ignore duplicate slack message", "message_id", inbound.MessageID)
		return
	}

	replyThreadTS := strings.TrimSpace(event.ThreadTS)
	if replyThreadTS == "" {
		replyThreadTS = strings.TrimSpace(event.TS)
	}
	h.logger.Info("received slack message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID, "chat_type", inbound.ChatType, "text_preview", shortText(inbound.Text))
	go h.processInbound(inbound, replyThreadTS)
}

func (h *Handler) processInbound(inbound protocol.InboundMessage, replyThreadTS string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	sink := &progressSink{client: h.client, channelID: inbound.ChatID, threadTS: replyThreadTS}
	response, err := h.sessions.HandleInbound(ctx, inbound, sink)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.logger.Info("slack inbound message interrupted", "message_id", inbound.MessageID)
			return
		}
		h.logger.Error("process slack inbound failed", "message_id", inbound.MessageID, "error", err)
		response = "处理消息失败，请稍后重试。"
	}

	if err := h.client.ReplyInThread(ctx, inbound.ChatID, replyThreadTS, response); err != nil {
		h.logger.Error("reply slack message failed", "message_id", inbound.MessageID, "error", err)
		return
	}
}

func (h *Handler) verifySignature(r *http.Request, body []byte) error {
	secret := strings.TrimSpace(h.cfg.SigningSecret)
	if secret == "" {
		return nil
	}

	timestamp := strings.TrimSpace(r.Header.Get("X-Slack-Request-Timestamp"))
	signature := strings.TrimSpace(r.Header.Get("X-Slack-Signature"))
	if timestamp == "" || signature == "" {
		return fmt.Errorf("missing slack signature headers")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid slack timestamp: %w", err)
	}
	if delta := time.Now().Unix() - ts; delta > 300 || delta < -300 {
		return fmt.Errorf("stale slack timestamp")
	}

	base := "v0:" + timestamp + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(base))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("slack signature mismatch")
	}
	return nil
}

type progressSink struct {
	client    *Client
	channelID string
	threadTS  string
}

func (s *progressSink) SendImmediateReply(ctx context.Context, session protocol.SessionRef, text string) error {
	return s.client.ReplyInThread(ctx, s.channelID, s.threadTS, text)
}

func (s *progressSink) SendProgress(ctx context.Context, session protocol.SessionRef, text string) error {
	msg := fmt.Sprintf("[会话 %s][进度] %s", session.PublicID, strings.TrimSpace(text))
	return s.client.ReplyInThread(ctx, s.channelID, s.threadTS, msg)
}

func (s *progressSink) SendChatText(ctx context.Context, chatID string, text string) error {
	return s.client.SendChatText(ctx, chatID, text)
}

func chatTypeFromSlack(channelType string) string {
	switch strings.ToLower(strings.TrimSpace(channelType)) {
	case "channel", "group", "mpim":
		return "group"
	default:
		return "p2p"
	}
}

func chatTypeFromLongConn(channelType string, channelID string) string {
	if strings.TrimSpace(channelType) != "" {
		return chatTypeFromSlack(channelType)
	}
	channelID = strings.TrimSpace(channelID)
	if strings.HasPrefix(channelID, "C") || strings.HasPrefix(channelID, "G") {
		return "group"
	}
	return "p2p"
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
