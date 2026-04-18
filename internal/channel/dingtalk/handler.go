package dingtalk

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/session"
	"rm_ai_agent/internal/store"
)

const (
	streamAPIEndpoint = "https://api.dingtalk.com/v1.0/gateway/connections/open"
)

type Handler struct {
	cfg             config.DingtalkConfig
	store           store.Store
	sessions        *session.Service
	client          *Client
	logger          *logx.Logger
	connectedAt     time.Time
}

func NewHandler(cfg config.DingtalkConfig, store store.Store, sessions *session.Service, client *Client, logger *logx.Logger) *Handler {
	return &Handler{cfg: cfg, store: store, sessions: sessions, client: client, logger: logger}
}

func (h *Handler) StartLongConnection(ctx context.Context) error {
	if !h.cfg.Enabled {
		return nil
	}

	h.logger.Info("starting dingtalk long connection", "client_id", mask(h.cfg.ClientID))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := h.runLongConnection(ctx); err != nil {
				h.logger.Error("dingtalk long connection error, reconnecting in 5s", "error", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (h *Handler) runLongConnection(ctx context.Context) error {
	connInfo, err := h.openConnection()
	if err != nil {
		return fmt.Errorf("open connection: %w", err)
	}

	wsURL := fmt.Sprintf("%s?ticket=%s", connInfo.Endpoint, connInfo.Ticket)
	h.logger.Info("dingtalk connecting to websocket", "url", connInfo.Endpoint)

	// Use a dialer with explicit timeout
	dialer := websocket.Dialer{
		NetDial: (&net.Dialer{Timeout: 30 * time.Second}).Dial,
		HandshakeTimeout: 30 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	defer conn.Close()

	h.logger.Info("dingtalk websocket connected successfully")
	h.connectedAt = time.Now()

	errCh := make(chan error, 1)
	go func() {
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				h.logger.Error("dingtalk read message error", "error", err)
				errCh <- err
				return
			}
			h.logger.Info("dingtalk received raw message", "data", string(msgBytes))
			h.handleStreamMessage(ctx, msgBytes)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

type connectionInfo struct {
	Endpoint string `json:"endpoint"`
	Ticket  string `json:"ticket"`
}

func (h *Handler) openConnection() (*connectionInfo, error) {
	localIP, _ := getLocalIP()

	body := map[string]any{
		"clientId":     h.cfg.ClientID,
		"clientSecret": h.cfg.ClientSecret,
		"subscriptions": []map[string]string{
			{"type": "CALLBACK", "topic": "/v1.0/im/bot/messages/get"},
		},
		"localIp": localIP,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, streamAPIEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("open connection failed: %d %s", resp.StatusCode, string(respBody))
	}

	var result connectionInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &result, nil
}

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

type streamMessage struct {
	Type    string          `json:"type"`
	Headers messageHeaders  `json:"headers"`
	Data    json.RawMessage `json:"data"`
}

type messageHeaders struct {
	AppId         string `json:"appId"`
	ConnectionId string `json:"connectionId"`
	ContentType  string `json:"contentType"`
	MessageId    string `json:"messageId"`
	Time         string `json:"time"`
	Topic        string `json:"topic"`
}

type callbackData struct {
	SenderPlatform  string `json:"senderPlatform"`
	ConversationID  string `json:"conversationId"`
	ChatbotCorpId   string `json:"chatbotCorpId"`
	ChatbotUserId   string `json:"chatbotUserId"`
	OpenThreadId    string `json:"openThreadId"`
	MsgId           string `json:"msgId"`
	SenderNick      string `json:"senderNick"`
	IsAdmin         bool   `json:"isAdmin"`
	SenderStaffId   string `json:"senderStaffId"`
	CreateAt        int64  `json:"createAt"`
	SenderCorpId    string `json:"senderCorpId"`
	ConversationType string `json:"conversationType"`
	SenderId        string `json:"senderId"`
	ConversationTitle string `json:"conversationTitle"`
	IsInAtList      bool   `json:"isInAtList"`
	SessionWebhook string `json:"sessionWebhook"`
	Text           struct {
		Content string `json:"content"`
	} `json:"text"`
	RobotCode string `json:"robotCode"`
	MsgType   string `json:"msgtype"`
}

func (h *Handler) handleStreamMessage(ctx context.Context, msgBytes []byte) {
	h.logger.Info("handleStreamMessage called", "bytes_len", len(msgBytes))

	var msg streamMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		h.logger.Warn("failed to parse stream message", "error", err)
		return
	}

	h.logger.Info("outer message parsed", "type", msg.Type, "topic", msg.Headers.Topic, "data_len", len(msg.Data))

	if msg.Type == "ping" {
		h.logger.Debug("dingtalk received ping")
		return
	}

	if msg.Headers.Topic == "/v1.0/im/bot/messages/get" {
		h.processStreamBotMessage(ctx, msg.Data)
	}
}

func (h *Handler) processStreamBotMessage(ctx context.Context, dataBytes json.RawMessage) {
	h.logger.Info("processStreamBotMessage called", "data", string(dataBytes))

	// data字段是一个JSON字符串，需要再解析一次
	var dataStr string
	if err := json.Unmarshal(dataBytes, &dataStr); err != nil {
		h.logger.Warn("failed to extract data string", "error", err)
		return
	}

	var data callbackData
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		h.logger.Warn("failed to parse callback data", "error", err, "data", dataStr)
		return
	}

	// 跳过历史消息（消息创建时间早于连接建立时间）
	if data.CreateAt > 0 {
		msgTime := time.UnixMilli(data.CreateAt)
		if msgTime.Before(h.connectedAt.Add(-5 * time.Second)) {
			h.logger.Info("skipping historical message", "msg_time", msgTime, "connected_at", h.connectedAt)
			return
		}
	}

	h.logger.Info("callback data parsed", "msg_type", data.MsgType, "text", data.Text.Content, "session_webhook", data.SessionWebhook)

	text := strings.TrimSpace(data.Text.Content)
	if text == "" || data.SessionWebhook == "" {
		h.logger.Warn("dingtalk stream message empty or no session webhook", "msg_id", data.MsgId, "text", text, "session_webhook", data.SessionWebhook)
		return
	}

	h.logger.Info("creating inbound message")

	inbound := protocol.InboundMessage{
		Channel:    protocol.ChannelDingtalk,
		TenantKey:  data.RobotCode,
		ChatID:     data.ConversationID,
		ChatType:   data.ConversationType,
		ThreadID:   "",
		MessageID:  fmt.Sprintf("%s:%s", data.ConversationID, data.MsgId),
		SenderID:   data.SenderStaffId,
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
		h.logger.Info("ignore duplicate stream message", "message_id", inbound.MessageID)
		return
	}

	h.logger.Info("received dingtalk stream message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID, "chat_type", inbound.ChatType, "text_preview", shortText(inbound.Text))
	go h.processInbound(inbound, data.SessionWebhook)
}

func (h *Handler) processInbound(inbound protocol.InboundMessage, sessionWebhook string) {
	h.logger.Info("start processing inbound message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID, "chat_type", inbound.ChatType)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	sink := &progressSink{client: h.client, sessionWebhook: sessionWebhook}
	response, err := h.sessions.HandleInbound(ctx, inbound, sink)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.logger.Info("inbound message interrupted", "message_id", inbound.MessageID)
			return
		}
		h.logger.Error("process inbound failed", "message_id", inbound.MessageID, "error", err)
		response = "处理消息失败，请稍后重试。"
	}

	// 等待一小段时间确保进度消息先发送完成
	time.Sleep(500 * time.Millisecond)

	if err := h.client.replyText(ctx, sessionWebhook, response); err != nil {
		h.logger.Error("reply dingtalk message failed", "message_id", inbound.MessageID, "error", err)
		return
	}
	h.logger.Info("finished processing inbound message", "message_id", inbound.MessageID)
}

type progressSink struct {
	client         *Client
	sessionWebhook string
}

func (s *progressSink) SendImmediateReply(ctx context.Context, session protocol.SessionRef, text string) error {
	return s.client.replyText(ctx, s.sessionWebhook, text)
}

func (s *progressSink) SendProgress(ctx context.Context, session protocol.SessionRef, text string) error {
	return s.client.replyText(ctx, s.sessionWebhook, fmt.Sprintf("[会话 %s][进度] %s", session.PublicID, strings.TrimSpace(text)))
}

func (s *progressSink) SendChatText(ctx context.Context, chatID string, text string) error {
	return s.client.SendChatText(ctx, chatID, text)
}

func shortText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 100 {
		return text
	}
	return text[:100] + "..."
}

func mask(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:4] + "..." + value[len(value)-4:]
}

// ServeHTTP handles HTTP webhook (for backward compatibility when mode is webhook)
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("dingtalk handler panic recovered", "panic", r)
			writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		}
	}()

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("dingtalk webhook called", "method", r.Method, "path", r.URL.Path)

	// 先读取 body 获取 encrypt 字段用于签名验证
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		h.logger.Error("dingtalk webhook failed to read body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req webhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Warn("dingtalk webhook failed to parse json", "error", err, "body", string(body))
		writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		return
	}

	// 验证签名
	if err := h.verifyDingTalkSignature(r, req.Encrypt); err != nil {
		h.logger.Warn("dingtalk signature verification failed", "error", err)
		writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		return
	}

	h.logger.Info("dingtalk webhook received",
		"msgType", req.MsgType,
		"encrypt", req.Encrypt != "",
		"text", req.Text.Content,
		"sessionWebhook", req.SessionWebhook,
		"conversationId", req.ConversationID)

	// Decrypt message if encrypted
	text := req.Text.Content
	if req.Encrypt != "" {
		h.logger.Info("dingtalk message is encrypted, attempting to decrypt")
		decrypted, err := h.decryptMessage(req.Encrypt)
		if err != nil {
			h.logger.Warn("dingtalk failed to decrypt message", "error", err)
			writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
			return
		}
		h.logger.Info("dingtalk decrypted successfully", "decrypted", decrypted)
		// Parse decrypted content
		var decryptedMsg struct {
			EventType      string `json:"EventType"`
			Text          struct{ Content string `json:"content"` } `json:"text"`
			SessionWebhook string `json:"sessionWebhook"`
			ConversationID string `json:"conversationId"`
			ConversationType string `json:"conversationType"`
			SenderStaffID string `json:"senderStaffId"`
			RobotCode    string `json:"robotCode"`
		}
		if err := json.Unmarshal([]byte(decrypted), &decryptedMsg); err != nil {
			h.logger.Warn("dingtalk failed to parse decrypted message", "error", err)
			writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
			return
		}

		// 处理 check_url 验证消息
		if decryptedMsg.EventType == "check_url" {
			h.logger.Info("dingtalk check_url verification received, returning encrypted success")
			// 钉钉要求返回加密的 "success" 消息
			encrypt, err := h.encryptMessage("success")
			if err != nil {
				h.logger.Error("encrypt success message failed", "error", err)
				writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
				return
			}
			timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
			nonce := generateRandomString(16)
			sign := computeSignature(h.cfg.VerificationToken, timestamp, nonce, encrypt)
			writeJSON(w, http.StatusOK, map[string]string{
				"msg_signature": sign,
				"encrypt":       encrypt,
				"timeStamp":     timestamp,
				"nonce":         nonce,
			})
			return
		}

		text = decryptedMsg.Text.Content
		req.SessionWebhook = decryptedMsg.SessionWebhook
		req.ConversationID = decryptedMsg.ConversationID
		req.ConversationType = decryptedMsg.ConversationType
		req.SenderStaffID = decryptedMsg.SenderStaffID
		req.RobotCode = decryptedMsg.RobotCode
	}

	// 处理 check_url 验证消息（未加密的情况）
	if req.MsgType == "check_url" || (req.Encrypt != "" && text == "") {
		h.logger.Info("dingtalk check_url verification received (unencrypted)")
		encrypt, err := h.encryptMessage("success")
		if err != nil {
			h.logger.Error("encrypt success message failed", "error", err)
			writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
			return
		}
		timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
		nonce := generateRandomString(16)
		sign := computeSignature(h.cfg.VerificationToken, timestamp, nonce, encrypt)
		writeJSON(w, http.StatusOK, map[string]string{
			"msg_signature": sign,
			"encrypt":       encrypt,
			"timeStamp":     timestamp,
			"nonce":         nonce,
		})
		return
	}

	if strings.TrimSpace(text) == "" {
		h.logger.Info("dingtalk message text is empty, ignoring")
		writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		return
	}

	if req.SessionWebhook == "" {
		h.logger.Warn("dingtalk webhook missing session webhook, ignoring message")
		writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
		return
	}

	inbound := protocol.InboundMessage{
		Channel:    protocol.ChannelDingtalk,
		TenantKey:  req.RobotCode,
		ChatID:     req.ConversationID,
		ChatType:   req.ConversationType,
		ThreadID:   "",
		MessageID:  fmt.Sprintf("%s:%s", req.ConversationID, req.SessionWebhook),
		SenderID:   req.SenderStaffID,
		Text:       strings.TrimSpace(text),
		RawJSON:    string(body),
		ReceivedAt: time.Now().UTC(),
	}

	created, err := h.store.RecordInboundMessage(r.Context(), inbound)
	if err != nil {
		h.logger.Error("record inbound failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": ""})
	if !created {
		h.logger.Info("ignore duplicate webhook message", "message_id", inbound.MessageID)
		return
	}
	h.logger.Info("received webhook dingtalk message", "message_id", inbound.MessageID, "chat_id", inbound.ChatID, "chat_type", inbound.ChatType, "text_preview", shortText(inbound.Text))

	go h.processInbound(inbound, req.SessionWebhook)
}

// verifyDingTalkSignature verifies the DingTalk webhook signature
func (h *Handler) verifyDingTalkSignature(r *http.Request, encryptStr string) error {
	sign := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	if sign == "" || timestamp == "" || nonce == "" {
		return nil
	}

	if h.cfg.VerificationToken == "" {
		return nil
	}

	// 签名计算：SHA1(token + timestamp + nonce + encrypt)
	arr := []string{h.cfg.VerificationToken, timestamp, nonce, encryptStr}
	sort.Strings(arr)
	str := strings.Join(arr, "")
	sigHash := sha1.Sum([]byte(str))
	expectedSign := hex.EncodeToString(sigHash[:])

	if expectedSign != sign {
		return fmt.Errorf("signature mismatch: expected %s, got %s", expectedSign, sign)
	}

	return nil
}

// decryptMessage decrypts the encrypted message from DingTalk
func (h *Handler) decryptMessage(encryptStr string) (string, error) {
	if h.cfg.EncryptKey == "" {
		return encryptStr, nil
	}

	// 钉钉：encodingAesKey + "=" 然后 base64 解码得到 32 字节 AES key
	keyWithPadding := h.cfg.EncryptKey + "="
	aesKey, err := base64.StdEncoding.DecodeString(keyWithPadding)
	if err != nil {
		return "", fmt.Errorf("encrypt_key base64 decode failed: %w", err)
	}
	if len(aesKey) != 32 {
		return "", fmt.Errorf("encrypt_key must be 32 bytes after decode, got %d", len(aesKey))
	}

	// 对密文进行 base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encryptStr)
	if err != nil {
		return "", fmt.Errorf("ciphertext base64 decode failed: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("create cipher failed: %w", err)
	}

	// IV: 取 AES key 的前 16 字节
	iv := make([]byte, 16)
	copy(iv, aesKey)

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	// 去除 PKCS7 padding
	plaintext, err = unpadPKCS7(plaintext)
	if err != nil {
		return "", fmt.Errorf("unpad failed: %w", err)
	}

	// 解析钉钉加密消息格式：
	// 0-15: 16字节随机串
	// 16-19: 4字节网络序的明文长度
	// 20 到 20+length: 明文
	// 20+length 之后: corpId
	if len(plaintext) < 20 {
		return "", fmt.Errorf("decrypted plaintext too short: %d", len(plaintext))
	}

	// 读取明文长度（网络字节序，大端）
	length := int(binary.BigEndian.Uint32(plaintext[16:20]))
	if len(plaintext) < 20+length {
		return "", fmt.Errorf("invalid plaintext length: claimed %d, have %d", length, len(plaintext)-20)
	}

	result := string(plaintext[20 : 20+length])
	return result, nil
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

// encryptMessage encrypts a message for DingTalk response
func (h *Handler) encryptMessage(plaintext string) (string, error) {
	if h.cfg.EncryptKey == "" {
		return plaintext, nil
	}

	// 获取 AES key
	keyWithPadding := h.cfg.EncryptKey + "="
	aesKey, err := base64.StdEncoding.DecodeString(keyWithPadding)
	if err != nil {
		return "", fmt.Errorf("encrypt_key base64 decode failed: %w", err)
	}
	if len(aesKey) != 32 {
		return "", fmt.Errorf("encrypt_key must be 32 bytes after decode, got %d", len(aesKey))
	}

	// 生成 16 字节随机串
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random bytes failed: %w", err)
	}

	// 钉钉加密消息格式：random(16) + length(4) + plaintext + corpId
	plaintextBytes := []byte(plaintext)
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(plaintextBytes)))
	corpId := h.cfg.ClientID // 钉钉用 clientId 作为 corpId

	// 组装数据
	data := make([]byte, 0, 16+4+len(plaintextBytes)+len(corpId)+32)
	data = append(data, randomBytes...)
	data = append(data, lengthBytes...)
	data = append(data, plaintextBytes...)
	data = append(data, corpId...)

	// PKCS7 padding（block size = 32）
	padLen := 32 - len(data)%32
	if padLen == 0 {
		padLen = 32
	}
	padding := make([]byte, padLen)
	for i := range padding {
		padding[i] = byte(padLen)
	}
	data = append(data, padding...)

	// AES-CBC 加密
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("create cipher failed: %w", err)
	}
	iv := make([]byte, 16)
	copy(iv, aesKey)

	ciphertext := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, data)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(result)
}

// computeSignature computes DingTalk signature for response
func computeSignature(token, timestamp, nonce, encrypt string) string {
	arr := []string{token, timestamp, nonce, encrypt}
	sort.Strings(arr)
	str := strings.Join(arr, "")
	h := sha1.Sum([]byte(str))
	return hex.EncodeToString(h[:])
}

type webhookEnvelope struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	ConversationID    string `json:"conversationId"`
	ConversationType string `json:"conversationType"`
	SenderNick       string `json:"senderNick"`
	SenderStaffID    string `json:"senderStaffId"`
	SessionWebhook   string `json:"sessionWebhook"`
	SessionWebhookURI string `json:"sessionWebhookUri"`
	RobotCode       string `json:"robotCode"`
	Time            int64  `json:"time"`
}

// webhookRequest is the actual DingTalk webhook callback format
type webhookRequest struct {
	Timestamp      string `json:"timestamp"`
	MsgType       string `json:"msgtype"`
	EncrypteMode  string `json:"encrypteMode"`
	Encrypt       string `json:"encrypt"`
	WebhookURI    string `json:"webhookURI"`
	RobotCode     string `json:"robotCode"`
	ConversationID string `json:"conversationId"`
	ConversationType string `json:"conversationType"`
	SenderNick    string `json:"senderNick"`
	SenderStaffID string `json:"senderStaffId"`
	SessionWebhook string `json:"sessionWebhook"`
	Text          struct {
		Content string `json:"content"`
	} `json:"text"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *Client) replyText(ctx context.Context, sessionWebhook string, text string) error {
	if !c.cfg.Enabled {
		c.logger.Info("skip dingtalk reply because channel is disabled")
		return nil
	}
	c.logger.Info("replying dingtalk text via session webhook", "text_preview", preview(text))

	body := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}

	if err := c.postSessionWebhook(ctx, sessionWebhook, body); err != nil {
		return err
	}
	c.logger.Info("replied dingtalk text via session webhook")
	return nil
}

func (c *Client) postSessionWebhook(ctx context.Context, webhookURL string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal dingtalk payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create dingtalk request: %w", err)
	}
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
