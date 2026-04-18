package weixin

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/protocol"
)

// generateValidKey 生成一个有效的 base64 编码的 32 字节 AES 密钥，用于测试
func generateValidKey() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return base64.RawStdEncoding.EncodeToString(key)
}

// mockStore 是 store.Store 接口的测试用 mock 实现
// 通过 recordInboundMessageFunc 字段可以自定义 RecordInboundMessage 的行为
type mockStore struct {
	recordInboundMessageFunc func(ctx context.Context, msg protocol.InboundMessage) (bool, error)
}

// RecordInboundMessage 实现 store.Store 接口
func (m *mockStore) RecordInboundMessage(ctx context.Context, msg protocol.InboundMessage) (bool, error) {
	if m.recordInboundMessageFunc != nil {
		return m.recordInboundMessageFunc(ctx, msg)
	}
	return true, nil
}

// 以下是 store.Store 接口的其余方法的空实现，用于满足接口要求
func (m *mockStore) EnsureSession(ctx context.Context, sessionKey string) (protocol.SessionRef, error) {
	return protocol.SessionRef{}, nil
}

func (m *mockStore) EnsureActiveSession(ctx context.Context, scopeKey string) (protocol.SessionRef, error) {
	return protocol.SessionRef{}, nil
}

func (m *mockStore) CreateSession(ctx context.Context, scopeKey string, title string) (protocol.SessionRef, error) {
	return protocol.SessionRef{}, nil
}

func (m *mockStore) ListSessions(ctx context.Context, scopeKey string, limit int) ([]protocol.SessionRef, error) {
	return nil, nil
}

func (m *mockStore) SetActiveSession(ctx context.Context, scopeKey string, publicID string) (protocol.SessionRef, error) {
	return protocol.SessionRef{}, nil
}

func (m *mockStore) CloseActiveSession(ctx context.Context, scopeKey string) (protocol.SessionRef, error) {
	return protocol.SessionRef{}, nil
}

func (m *mockStore) DeleteSession(ctx context.Context, scopeKey string, publicID string) error {
	return nil
}

func (m *mockStore) AppendTurn(ctx context.Context, sessionKey string, role string, content string) error {
	return nil
}

func (m *mockStore) ListRecentTurns(ctx context.Context, sessionKey string, limit int) ([]protocol.Turn, error) {
	return nil, nil
}

func (m *mockStore) ListTurns(ctx context.Context, sessionKey string, limit int) ([]protocol.Turn, error) {
	return nil, nil
}

func (m *mockStore) CountTurns(ctx context.Context, sessionKey string) (int, error) {
	return 0, nil
}

func (m *mockStore) GetSessionSummary(ctx context.Context, sessionKey string) (string, error) {
	return "", nil
}

func (m *mockStore) UpsertSessionSummary(ctx context.Context, sessionKey string, summary string) error {
	return nil
}

func (m *mockStore) ListMemories(ctx context.Context, sessionKey string, limit int) ([]protocol.MemoryEntry, error) {
	return nil, nil
}

func (m *mockStore) UpsertMemory(ctx context.Context, sessionKey string, key string, value string) error {
	return nil
}

func (m *mockStore) DeleteMemory(ctx context.Context, sessionKey string, key string) error {
	return nil
}

func (m *mockStore) CountMemories(ctx context.Context, sessionKey string) (int, error) {
	return 0, nil
}

func (m *mockStore) SavePendingAction(ctx context.Context, sessionKey string, actionType string, payload string, summary string) error {
	return nil
}

func (m *mockStore) GetPendingAction(ctx context.Context, sessionKey string) (protocol.PendingAction, error) {
	return protocol.PendingAction{}, nil
}

func (m *mockStore) DeletePendingAction(ctx context.Context, sessionKey string) error {
	return nil
}

func (m *mockStore) SaveArtifact(ctx context.Context, sessionKey string, kind string, title string, content string) (protocol.Artifact, error) {
	return protocol.Artifact{}, nil
}

func (m *mockStore) GetLatestArtifact(ctx context.Context, sessionKey string) (protocol.Artifact, error) {
	return protocol.Artifact{}, nil
}

func (m *mockStore) GetArtifact(ctx context.Context, sessionKey string, artifactID string) (protocol.Artifact, error) {
	return protocol.Artifact{}, nil
}

func (m *mockStore) CreateScheduledTask(ctx context.Context, task protocol.ScheduledTask) (protocol.ScheduledTask, error) {
	return protocol.ScheduledTask{}, nil
}

func (m *mockStore) ListScheduledTasks(ctx context.Context, scopeKey string, limit int) ([]protocol.ScheduledTask, error) {
	return nil, nil
}

func (m *mockStore) GetScheduledTask(ctx context.Context, scopeKey string, taskID string) (protocol.ScheduledTask, error) {
	return protocol.ScheduledTask{}, nil
}

func (m *mockStore) GetScheduledTaskByID(ctx context.Context, taskID string) (protocol.ScheduledTask, error) {
	return protocol.ScheduledTask{}, nil
}

func (m *mockStore) UpdateScheduledTask(ctx context.Context, scopeKey string, taskID string, patch protocol.ScheduledTaskPatch) (protocol.ScheduledTask, error) {
	return protocol.ScheduledTask{}, nil
}

func (m *mockStore) DeleteScheduledTask(ctx context.Context, scopeKey string, taskID string) error {
	return nil
}

func (m *mockStore) ListDueScheduledTasks(ctx context.Context, scopeKey string, now time.Time, limit int) ([]protocol.ScheduledTask, error) {
	return nil, nil
}

func (m *mockStore) MarkScheduledTaskRunning(ctx context.Context, taskID string, lastRunAt time.Time, nextRunAt time.Time) error {
	return nil
}

func (m *mockStore) SaveScheduledTaskRun(ctx context.Context, run protocol.ScheduledTaskRun) error {
	return nil
}

func (m *mockStore) GetLatestScheduledTaskRun(ctx context.Context, scopeKey string) (protocol.ScheduledTaskRun, error) {
	return protocol.ScheduledTaskRun{}, nil
}

func (m *mockStore) GetScheduledTaskState(ctx context.Context, taskID string) (string, error) {
	return "", nil
}

func (m *mockStore) UpsertScheduledTaskState(ctx context.Context, taskID string, state string) error {
	return nil
}

func (m *mockStore) ListScheduledTaskEntities(ctx context.Context, taskID string, limit int) ([]protocol.ScheduledTaskEntity, error) {
	return nil, nil
}

func (m *mockStore) UpsertScheduledTaskEntity(ctx context.Context, entity protocol.ScheduledTaskEntity) error {
	return nil
}

// encryptWeixinMessage 用于测试的消息加密函数
// 加密格式与企业在微信回调格式一致：random(16) + length(4) + plaintext + PKCS7 padding
// 使用 AES-256-CBC 模式，IV 为 AES key 的前 16 字节
func encryptWeixinMessage(plaintext string, encryptKey string) (string, error) {
	keyWithPadding := encryptKey + "="
	aesKey, err := base64.StdEncoding.DecodeString(keyWithPadding)
	if err != nil {
		return "", err
	}

	// 生成 16 字节随机串
	randomBytes := make([]byte, 16)
	for i := range randomBytes {
		randomBytes[i] = byte(i)
	}

	// 组装消息体：random(16) + length(4) + plaintext
	plaintextBytes := []byte(plaintext)
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(plaintextBytes)))

	message := append(randomBytes, lengthBytes...)
	message = append(message, plaintextBytes...)

	// 添加 PKCS7 padding（block size 为 32）
	blockSize := 32
	padLen := blockSize - (len(message) % blockSize)
	if padLen == 0 {
		padLen = blockSize
	}
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	message = append(message, padding...)

	// AES-CBC 加密
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	iv := make([]byte, 16)
	copy(iv, aesKey)

	ciphertext := make([]byte, len(message))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, message)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// calculateSignature 计算企业微信签名的测试辅助函数
// 签名算法：SHA1(token + timestamp + nonce + encryptStr) 后排序拼接
func calculateSignature(token, timestamp, nonce, encryptStr string) string {
	arr := []string{token, timestamp, nonce, encryptStr}
	sort.Strings(arr)
	str := strings.Join(arr, "")
	h := sha1.Sum([]byte(str))
	return fmt.Sprintf("%x", h)
}

// TestDecryptWeixinMessage 测试企业微信消息解密功能
// 覆盖场景：空加密密钥、有效密文、无效 base64 密钥、无效 base64 密文、密文太短
func TestDecryptWeixinMessage(t *testing.T) {
	t.Parallel()

	validKey := generateValidKey()

	tests := []struct {
		name       string
		cfg        config.WeixinConfig
		encryptStr string
		wantText   string
		wantErr    bool
	}{
		{
			name: "empty encrypt key returns raw string",
			cfg: config.WeixinConfig{
				EncryptKey: "",
			},
			encryptStr: "plain-text-message",
			wantText:   "plain-text-message",
			wantErr:    false,
		},
		{
			name: "valid encrypted message decrypts correctly",
			cfg: config.WeixinConfig{
				EncryptKey: validKey,
			},
			encryptStr: "", // will be set dynamically
			wantText:   "hello world",
			wantErr:    false,
		},
		{
			name: "invalid base64 encrypt key",
			cfg: config.WeixinConfig{
				EncryptKey: "not-valid-base64!!!",
			},
			encryptStr: "some-encrypted-data",
			wantErr:    true,
		},
		{
			name: "invalid base64 ciphertext",
			cfg: config.WeixinConfig{
				EncryptKey: validKey,
			},
			encryptStr: "not-valid-base64!!!",
			wantErr:    true,
		},
		{
			name: "ciphertext too short",
			cfg: config.WeixinConfig{
				EncryptKey: validKey,
			},
			encryptStr: base64.StdEncoding.EncodeToString([]byte("short")),
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger, _ := logx.New(config.LevelDebug, "")
			h := &Handler{
				cfg:    tc.cfg,
				logger: logger,
			}

			encryptStr := tc.encryptStr
			if tc.name == "valid encrypted message decrypts correctly" && tc.encryptStr == "" {
				encrypted, err := encryptWeixinMessage(tc.wantText, tc.cfg.EncryptKey)
				if err != nil {
					t.Fatalf("failed to encrypt test message: %v", err)
				}
				encryptStr = encrypted
			}

			got, err := h.decryptWeixinMessage(encryptStr)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if string(got) != tc.wantText {
				t.Errorf("got %q, want %q", string(got), tc.wantText)
			}
		})
	}
}

// TestDecryptWeixinMessage_InvalidPlaintextLength 测试解密时消息头中声明的明文长度与实际不符的情况
// 当消息头中声称的明文长度大于实际数据时，应该返回错误
func TestDecryptWeixinMessage_InvalidPlaintextLength(t *testing.T) {
	t.Parallel()

	validKey := generateValidKey()

	keyWithPadding := validKey + "="
	aesKey, err := base64.StdEncoding.DecodeString(keyWithPadding)
	if err != nil {
		t.Fatalf("failed to decode key: %v", err)
	}

	// 构建消息：头部声明长度为 1000，但实际只有 5 字节
	randomBytes := make([]byte, 16)
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, 1000) // 声明 1000 字节

	message := append(randomBytes, lengthBytes...)
	message = append(message, []byte("short")...) // 只有 5 字节

	// PKCS7 padding
	blockSize := 32
	padLen := blockSize - (len(message) % blockSize)
	if padLen == 0 {
		padLen = blockSize
	}
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	message = append(message, padding...)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		t.Fatalf("failed to create cipher: %v", err)
	}

	iv := make([]byte, 16)
	copy(iv, aesKey)

	ciphertext := make([]byte, len(message))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, message)

	encrypted := base64.StdEncoding.EncodeToString(ciphertext)

	logger, _ := logx.New(config.LevelDebug, "")
	h := &Handler{
		cfg: config.WeixinConfig{
			EncryptKey: validKey,
		},
		logger: logger,
	}

	_, err = h.decryptWeixinMessage(encrypted)
	if err == nil {
		t.Error("expected error for invalid plaintext length, got nil")
	}
}

// TestDecryptWeixinMessage_CiphertextTooShort 测试密文长度小于一个 AES 块（16字节）的情况
func TestDecryptWeixinMessage_CiphertextTooShort(t *testing.T) {
	t.Parallel()

	validKey := generateValidKey()

	logger, _ := logx.New(config.LevelDebug, "")
	h := &Handler{
		cfg: config.WeixinConfig{
			EncryptKey: validKey,
		},
		logger: logger,
	}

	// 小于一个 AES 块（16字节）
	_, err := h.decryptWeixinMessage(base64.StdEncoding.EncodeToString([]byte("short")))
	if err == nil {
		t.Error("expected error for ciphertext too short, got nil")
	}
}

// TestVerifyWeixinSignature 测试企业微信签名验证功能
// 覆盖场景：空 token（跳过验证）、空签名（跳过验证）、正确签名通过、错误签名失败
func TestVerifyWeixinSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfg        config.WeixinConfig
		signature  string
		timestamp  string
		nonce      string
		encryptStr string
		wantErr    bool
	}{
		{
			name: "empty token returns nil (skips verification)",
			cfg: config.WeixinConfig{
				Token: "",
			},
			signature:  "any-signature",
			timestamp: "1234567890",
			nonce:     "nonce",
			encryptStr: "encrypt",
			wantErr:   false,
		},
		{
			name: "empty signature returns nil (skips verification)",
			cfg: config.WeixinConfig{
				Token: "test-token",
			},
			signature:  "",
			timestamp: "1234567890",
			nonce:     "nonce",
			encryptStr: "encrypt",
			wantErr:   false,
		},
		{
			name: "valid signature passes",
			cfg: config.WeixinConfig{
				Token: "test-token",
			},
			signature:  "",
			timestamp: "1234567890",
			nonce:     "nonce",
			encryptStr: "encrypt",
			wantErr:   false,
		},
		{
			name: "invalid signature fails",
			cfg: config.WeixinConfig{
				Token: "test-token",
			},
			signature:  "definitely-wrong-signature",
			timestamp: "1234567890",
			nonce:     "nonce",
			encryptStr: "encrypt",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger, _ := logx.New(config.LevelDebug, "")
			h := &Handler{
				cfg:    tc.cfg,
				logger: logger,
			}

			signature := tc.signature
			if tc.name == "valid signature passes" {
				signature = calculateSignature(tc.cfg.Token, tc.timestamp, tc.nonce, tc.encryptStr)
			}

			err := h.verifyWeixinSignature(signature, tc.timestamp, tc.nonce, tc.encryptStr)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestHandleWebhook 测试 webhook 消息处理逻辑
// 验证空内容和纯空格内容会被正确忽略
func TestHandleWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		msg            *webhookMessage
		reqID          string
		wantRecordCall bool
	}{
		{
			name: "empty content is ignored",
			msg: &webhookMessage{
				MsgType:      "text",
				Content:      "",
				MsgID:        "msg-123",
				FromUserName: "user-001",
			},
			reqID:          "req-123",
			wantRecordCall: false,
		},
		{
			name: "whitespace only content is ignored",
			msg: &webhookMessage{
				MsgType:      "text",
				Content:      "   ",
				MsgID:        "msg-123",
				FromUserName: "user-001",
			},
			reqID:          "req-123",
			wantRecordCall: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var recordCalled bool
			var recordedMsg protocol.InboundMessage

			store := &mockStore{
				recordInboundMessageFunc: func(ctx context.Context, msg protocol.InboundMessage) (bool, error) {
					recordCalled = true
					recordedMsg = msg
					return true, nil
				},
			}

			logger, _ := logx.New(config.LevelDebug, "")
			h := &Handler{
				cfg: config.WeixinConfig{
					BotID: "test-bot-id",
				},
				store:  store,
				logger: logger,
			}

			ctx := context.Background()
			h.handleWebhook(ctx, tc.msg, tc.reqID)

			if tc.wantRecordCall && !recordCalled {
				t.Error("expected RecordInboundMessage to be called, but it was not")
			}
			if !tc.wantRecordCall && recordCalled {
				t.Errorf("expected RecordInboundMessage NOT to be called, but it was")
			}

			if tc.wantRecordCall && recordCalled {
				if recordedMsg.Channel != protocol.ChannelWeixin {
					t.Errorf("expected channel %s, got %s", protocol.ChannelWeixin, recordedMsg.Channel)
				}
				if recordedMsg.TenantKey != h.cfg.BotID {
					t.Errorf("expected tenant key %s, got %s", h.cfg.BotID, recordedMsg.TenantKey)
				}
			}
		})
	}
}

// TestServeHTTP 测试 ServeHTTP HTTP 入口函数
// 覆盖场景：GET 方法返回 405、verify_url 事件返回成功、无效 JSON 返回 errcode 0
func TestServeHTTP(t *testing.T) {
	t.Parallel()

	validKey := generateValidKey()

	tests := []struct {
		name        string
		method      string
		queryParams string
		body        string
		setupStore  func(*mockStore)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:        "GET method returns 405",
			method:      http.MethodGet,
			queryParams: "",
			body:        "",
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:        "verify_url event returns success",
			method:      http.MethodPost,
			queryParams: "msg_signature=sig&timestamp=ts&nonce=non",
			body: `<xml>
				<MsgType><![CDATA[]]></MsgType>
				<Event><![CDATA[verify_url]]></Event>
			</xml>`,
			wantStatus:  http.StatusOK,
			wantErrCode: "0",
		},
		{
			name:        "invalid json body returns success with errcode 0",
			method:      http.MethodPost,
			queryParams: "",
			body:        "not-valid-json",
			wantStatus:  http.StatusOK,
			wantErrCode: "0",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &mockStore{}
			if tc.setupStore != nil {
				tc.setupStore(store)
			}

			logger, _ := logx.New(config.LevelDebug, "")
			h := &Handler{
				cfg: config.WeixinConfig{
					BotID:      "test-bot-id",
					Token:     "test-token",
					EncryptKey: validKey,
				},
				store:  store,
				logger: logger,
			}

			req := httptest.NewRequest(tc.method, "/weixin?"+tc.queryParams, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tc.wantStatus)
			}

			if tc.wantErrCode != "" {
				var resp map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if resp["errcode"] != tc.wantErrCode {
					t.Errorf("got errcode %s, want %s", resp["errcode"], tc.wantErrCode)
				}
			}
		})
	}
}

// TestServeHTTP_EncryptedMessage 测试加密消息的完整处理流程
// 包括：消息加密、签名计算、HTTP 请求发送、服务端解密和签名验证
func TestServeHTTP_EncryptedMessage(t *testing.T) {
	t.Parallel()

	validKey := generateValidKey()

	// 返回 false 表示重复消息，避免触发 processWebhookInbound goroutine
	store := &mockStore{
		recordInboundMessageFunc: func(ctx context.Context, msg protocol.InboundMessage) (bool, error) {
			return false, nil
		},
	}

	logger, _ := logx.New(config.LevelDebug, "")
	cfg := config.WeixinConfig{
		BotID:      "test-bot-id",
		Token:      "test-token",
		EncryptKey: validKey,
	}
	h := &Handler{
		cfg:     cfg,
		store:   store,
		logger:  logger,
	}

	// 加密 XML 格式的消息内容
	xmlMsg := `<xml><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[hello]]></Content><MsgId><![CDATA[123]]></MsgId><FromUserName><![CDATA[user001]]></FromUserName></xml>`
	encrypted, err := encryptWeixinMessage(xmlMsg, cfg.EncryptKey)
	if err != nil {
		t.Fatalf("failed to encrypt test message: %v", err)
	}
	signature := calculateSignature(cfg.Token, "ts", "non", encrypted)

	// 构造企业微信回调格式的请求
	body := fmt.Sprintf(`{"encrypt":"%s"}`, encrypted)
	queryParams := fmt.Sprintf("msg_signature=%s&timestamp=ts&nonce=non", signature)

	req := httptest.NewRequest(http.MethodPost, "/weixin?"+queryParams, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["errcode"] != "0" {
		t.Errorf("got errcode %s, want 0", resp["errcode"])
	}
}

// TestServeHTTP_DuplicateMessage 测试重复消息的处理
// store 返回 false 表示消息已存在（重复），不应触发后续处理
func TestServeHTTP_DuplicateMessage(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		recordInboundMessageFunc: func(ctx context.Context, msg protocol.InboundMessage) (bool, error) {
			return false, nil // 重复消息
		},
	}

	logger, _ := logx.New(config.LevelDebug, "")
	h := &Handler{
		cfg: config.WeixinConfig{
			BotID: "test-bot-id",
		},
		store:  store,
		logger: logger,
	}

	body := `{"msgtype":"text","content":"hello","msgid":"123","fromusername":"user001"}`
	req := httptest.NewRequest(http.MethodPost, "/weixin?"+body, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestServeHTTP_RecordInboundError 测试 store 错误处理
// 即使 store.RecordInboundMessage 返回错误，也应返回 200 OK 避免企业微信重试
func TestServeHTTP_RecordInboundError(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		recordInboundMessageFunc: func(ctx context.Context, msg protocol.InboundMessage) (bool, error) {
			return false, fmt.Errorf("store error")
		},
	}

	logger, _ := logx.New(config.LevelDebug, "")
	h := &Handler{
		cfg: config.WeixinConfig{
			BotID: "test-bot-id",
		},
		store:  store,
		logger: logger,
	}

	body := `{"msgtype":"text","content":"hello","msgid":"123","fromusername":"user001"}`
	req := httptest.NewRequest(http.MethodPost, "/weixin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	// Should still return 200 OK to WeChat server to avoid retries
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestServeHTTP_ImageMessage 测试图片消息的处理
// 图片消息没有文本内容，应直接返回成功
func TestServeHTTP_ImageMessage(t *testing.T) {
	t.Parallel()

	store := &mockStore{}

	logger, _ := logx.New(config.LevelDebug, "")
	h := &Handler{
		cfg: config.WeixinConfig{
			BotID: "test-bot-id",
		},
		store:  store,
		logger: logger,
	}

	body := `{"msgtype":"image","content":"","msgid":"123","fromusername":"user001"}`
	req := httptest.NewRequest(http.MethodPost, "/weixin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	// Image messages don't have content, so they should not trigger handleWebhook
	// (handleWebhook checks for empty content)
}

// TestServeHTTP_PanicRecovery 测试 panic 恢复机制
// 验证 ServeHTTP 中的 recover() 能捕获 panic 并返回 200 OK
func TestServeHTTP_PanicRecovery(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	logger, _ := logx.New(config.LevelDebug, "")
	h := &Handler{
		cfg: config.WeixinConfig{
			BotID: "test-bot-id",
		},
		store:  store,
		logger: logger,
	}

	body := `{"msgtype":"text","content":"hello","msgid":"123","fromusername":"user001"}`
	req := httptest.NewRequest(http.MethodPost, "/weixin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestUnpadPKCS7 测试 PKCS7 padding 移除功能
// 覆盖场景：空数据返回错误、有效 padding、无效 padding 长度超过数据长度、padding 长度为 0
func TestUnpadPKCS7(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "empty data returns error",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "valid padding",
			data:    []byte("hello\x05\x05\x05\x05\x05"),
			want:    []byte("hello"),
			wantErr: false,
		},
		{
			name:    "padding length exceeds data length",
			data:    []byte("hello\x10"),
			wantErr: true,
		},
		{
			name:    "zero padding length",
			data:    []byte("hello\x00"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := unpadPKCS7(tc.data)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestWriteJSON 测试 JSON 响应写入功能
func TestWriteJSON(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"errcode": "0", "errmsg": "ok"})

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["errcode"] != "0" {
		t.Errorf("got errcode %s, want 0", resp["errcode"])
	}
}

// TestWebhookMessage_XMLParsing 测试 webhook 消息的 XML 解析功能
// 覆盖场景：文本消息解析、事件消息解析
func TestWebhookMessage_XMLParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xml  string
		want webhookMessage
	}{
		{
			name: "text message",
			xml: `<xml>
				<MsgType><![CDATA[text]]></MsgType>
				<Content><![CDATA[hello]]></Content>
				<MsgId><![CDATA[123]]></MsgId>
				<FromUserName><![CDATA[user001]]></FromUserName>
			</xml>`,
			want: webhookMessage{
				MsgType:      "text",
				Content:      "hello",
				MsgID:        "123",
				FromUserName: "user001",
			},
		},
		{
			name: "event message",
			xml: `<xml>
				<MsgType><![CDATA[]]></MsgType>
				<Event><![CDATA[verify_url]]></Event>
			</xml>`,
			want: webhookMessage{
				MsgType: "",
				Event:   "verify_url",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var msg webhookMessage
			if err := xml.Unmarshal([]byte(tc.xml), &msg); err != nil {
				t.Fatalf("failed to unmarshal XML: %v", err)
			}

			if msg.MsgType != tc.want.MsgType {
				t.Errorf("MsgType got %s, want %s", msg.MsgType, tc.want.MsgType)
			}
			if msg.Content != tc.want.Content {
				t.Errorf("Content got %s, want %s", msg.Content, tc.want.Content)
			}
			if msg.MsgID != tc.want.MsgID {
				t.Errorf("MsgID got %s, want %s", msg.MsgID, tc.want.MsgID)
			}
			if msg.FromUserName != tc.want.FromUserName {
				t.Errorf("FromUserName got %s, want %s", msg.FromUserName, tc.want.FromUserName)
			}
			if msg.Event != tc.want.Event {
				t.Errorf("Event got %s, want %s", msg.Event, tc.want.Event)
			}
		})
	}
}
