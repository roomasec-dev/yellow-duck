package weixin

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
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
	wsEndpoint    = "wss://openws.work.weixin.qq.com"
	apiBaseURL    = "https://qyapi.weixin.qq.com"
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

	// 获取 URL 参数
	msgSignature := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		h.logger.Error("weixin webhook failed to read body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.logger.Debug("weixin webhook raw body", "body", string(body))

	// 首先尝试解析 JSON 格式
	var rawBody struct {
		Encrypt string `json:"encrypt"`
		MsgType string `json:"msgtype"`
		Content string `json:"content"`
		MsgId   string `json:"msgid"`
		FromUserName string `json:"fromusername"`
	}

	if err := json.Unmarshal(body, &rawBody); err != nil {
		h.logger.Warn("weixin webhook failed to parse json", "error", err)
		writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		return
	}

	var msgContent []byte
	reqID := uuid.New().String()

	// 如果有 encrypt 字段，说明是加密消息
	if rawBody.Encrypt != "" {
		// 验证签名
		if err := h.verifyWeixinSignature(msgSignature, timestamp, nonce, rawBody.Encrypt); err != nil {
			h.logger.Warn("weixin signature verification failed", "error", err)
			writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
			return
		}

		// 解密消息
		decrypted, err := h.decryptWeixinMessage(rawBody.Encrypt)
		if err != nil {
			h.logger.Warn("weixin failed to decrypt message", "error", err)
			writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
			return
		}

		h.logger.Debug("weixin decrypted message", "content", string(decrypted))
		msgContent = decrypted
	} else {
		// 未加密的消息，直接使用原始 body
		msgContent = body
	}

	// 解析 XML 格式的消息
	var msg webhookMessage
	if err := xml.Unmarshal(msgContent, &msg); err != nil {
		h.logger.Warn("weixin failed to parse message xml", "error", err, "content", string(msgContent))
		writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		return
	}

	// 处理 URL 验证事件
	if msg.Event == "verify_url" || msg.MsgType == "" {
		h.logger.Info("weixin url verification received")
		writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		return
	}

	// 处理消息回调
	if msg.MsgType == "text" || msg.MsgType == "image" || msg.MsgType == "voice" {
		h.handleWebhook(r.Context(), &msg, reqID)
	}

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

// webhookMessage 是企业微信回调的加密消息结构
type webhookMessage struct {
	XMLName xml.Name `xml:"xml"`
	MsgType     string `xml:"MsgType"`
	Content     string `xml:"Content"`
	MsgID       string `xml:"MsgId"`
	FromUserName string `xml:"FromUserName"`
	ToUserName   string `xml:"ToUserName"`
	CreateTime   string `xml:"CreateTime"`
	AgentID      string `xml:"AgentID"`
	Event        string `xml:"Event"`
}

// verifyWeixinSignature 验证企业微信签名
func (h *Handler) verifyWeixinSignature(msgSignature, timestamp, nonce, encryptStr string) error {
	if h.cfg.Token == "" || msgSignature == "" {
		return nil
	}

	// 签名计算：SHA1(token + timestamp + nonce + encryptStr)
	arr := []string{h.cfg.Token, timestamp, nonce, encryptStr}
	sort.Strings(arr)
	str := strings.Join(arr, "")
	h2 := sha1.Sum([]byte(str))
	expectedSign := fmt.Sprintf("%x", h2)

	if expectedSign != msgSignature {
		return fmt.Errorf("signature mismatch: expected %s, got %s", expectedSign, msgSignature)
	}
	return nil
}

// decryptWeixinMessage 解密企业微信消息
func (h *Handler) decryptWeixinMessage(encryptStr string) ([]byte, error) {
	if h.cfg.EncryptKey == "" {
		// 未配置加密密钥，直接返回
		return []byte(encryptStr), nil
	}

	// 企业微信：AES key = base64(encryptKey + "=")
	keyWithPadding := h.cfg.EncryptKey + "="
	aesKey, err := base64.StdEncoding.DecodeString(keyWithPadding)
	if err != nil {
		return nil, fmt.Errorf("encrypt_key base64 decode failed: %w", err)
	}
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("encrypt_key must be 32 bytes after decode, got %d", len(aesKey))
	}

	// 对密文进行 base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encryptStr)
	if err != nil {
		return nil, fmt.Errorf("ciphertext base64 decode failed: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher failed: %w", err)
	}

	// IV: 取 AES key 的前 16 字节
	iv := make([]byte, 16)
	copy(iv, aesKey)

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	// 去除 PKCS7 padding
	plaintext, err = unpadPKCS7(plaintext)
	if err != nil {
		return nil, fmt.Errorf("unpad failed: %w", err)
	}

	// 企业微信加密消息格式：
	// 0-15: 16字节随机串
	// 16-19: 4字节网络序的明文长度
	// 20 到 end: 明文（XML格式）
	if len(plaintext) < 20 {
		return nil, fmt.Errorf("decrypted plaintext too short: %d", len(plaintext))
	}

	// 读取明文长度（网络字节序，大端）
	length := int(binary.BigEndian.Uint32(plaintext[16:20]))
	if len(plaintext) < 20+length {
		return nil, fmt.Errorf("invalid plaintext length: claimed %d, have %d", length, len(plaintext)-20)
	}

	return plaintext[20 : 20+length], nil
}

// unpadPKCS7 removes PKCS7 padding
func unpadPKCS7(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	padLen := int(data[len(data)-1])
	if padLen > len(data) || padLen == 0 {
		return nil, fmt.Errorf("invalid padding length: %d", padLen)
	}
	return data[:len(data)-padLen], nil
}

// handleWebhook 处理 webhook 回调
func (h *Handler) handleWebhook(ctx context.Context, msg *webhookMessage, reqID string) {
	h.logger.Info("weixin webhook message", "msg_type", msg.MsgType, "msg_id", msg.MsgID, "content", msg.Content)

	text := strings.TrimSpace(msg.Content)
	if text == "" {
		return
	}

	inbound := protocol.InboundMessage{
		Channel:    protocol.ChannelWeixin,
		TenantKey:  h.cfg.BotID,
		ChatID:     msg.FromUserName, // 企业微信中 fromusername 是发送者用户ID
		ChatType:   "single",
		ThreadID:   "",
		MessageID:  msg.MsgID,
		SenderID:   msg.FromUserName,
		Text:       text,
		RawJSON:    "",
		ReceivedAt: time.Now().UTC(),
	}

	created, err := h.store.RecordInboundMessage(ctx, inbound)
	if err != nil {
		h.logger.Error("record inbound failed", "error", err)
		return
	}
	if !created {
		h.logger.Info("ignore duplicate webhook message", "message_id", inbound.MessageID)
		return
	}

	go h.processWebhookInbound(ctx, inbound, reqID, msg.FromUserName)
}

// processWebhookInbound 处理 webhook 收到的消息
func (h *Handler) processWebhookInbound(ctx context.Context, inbound protocol.InboundMessage, reqID string, fromUserName string) {
	h.logger.Info("start processing webhook inbound message", "message_id", inbound.MessageID, "from_user", fromUserName)

	sink := &webhookProgressSink{client: h.client, fromUserName: fromUserName, reqID: reqID}
	response, err := h.sessions.HandleInbound(ctx, inbound, sink)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.logger.Info("inbound message interrupted", "message_id", inbound.MessageID)
			return
		}
		h.logger.Error("process inbound failed", "message_id", inbound.MessageID, "error", err)
		response = "处理消息失败，请稍后重试。"
	}

	// 通过 webhook API 发送回复
	if err := h.client.sendWebhookReply(ctx, fromUserName, response); err != nil {
		h.logger.Error("send webhook reply failed", "message_id", inbound.MessageID, "error", err)
		return
	}
	h.logger.Info("finished processing webhook inbound message", "message_id", inbound.MessageID)
}

type webhookProgressSink struct {
	client      *Client
	fromUserName string
	reqID      string
}

func (s *webhookProgressSink) SendProgress(ctx context.Context, session protocol.SessionRef, text string) error {
	content := fmt.Sprintf("[会话 %s][进度] %s", session.PublicID, strings.TrimSpace(text))
	return s.client.sendWebhookReply(ctx, s.fromUserName, content)
}

func (s *webhookProgressSink) SendChatText(ctx context.Context, chatID string, text string) error {
	// webhook 模式下使用 fromUserName 作为聊天对象
	return s.client.sendWebhookReply(ctx, s.fromUserName, text)
}
