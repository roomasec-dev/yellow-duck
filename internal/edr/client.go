package edr

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"rm_ai_agent/internal/config"
)

type Client interface {
	ListHosts(ctx context.Context, req ListHostsRequest) (ListHostsResponse, error)
	IsolateHost(ctx context.Context, clientID string) (InstructionResult, error)
	ReleaseHost(ctx context.Context, clientID string) (InstructionResult, error)
	ListIncidents(ctx context.Context, req ListIncidentsRequest) (ListIncidentsResponse, error)
	ListDetections(ctx context.Context, req ListDetectionsRequest) (ListDetectionsResponse, error)
	ListLogs(ctx context.Context, req ListLogsRequest) (ListLogsResponse, error)
	ViewIncident(ctx context.Context, req IncidentViewRequest) (map[string]any, error)
	ViewDetection(ctx context.Context, req DetectionViewRequest) (map[string]any, error)
}

type OpenAPIClient struct {
	baseURL         string
	headers         map[string]string
	http            *http.Client
	cfg             config.EDRConfig
	platformBaseURL string
	platformAppKey  string
	platformAppSK   string
	tokenMu         sync.Mutex
	platformToken   string
	tokenExpiresAt  time.Time
}

func NewClient(cfg config.EDRConfig) *OpenAPIClient {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8081/open_api/rm/v1"
	}
	headers := make(map[string]string, len(cfg.Headers)+1)
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	if cfg.AuthToken != "" {
		headers["Authorization"] = cfg.AuthToken
	}
	platformAppKey := strings.TrimSpace(cfg.Platform.AppKey)
	if platformAppKey == "" && cfg.Platform.AppKeyEnv != "" {
		platformAppKey = strings.TrimSpace(os.Getenv(cfg.Platform.AppKeyEnv))
	}
	platformAppSK := strings.TrimSpace(cfg.Platform.AppSecret)
	if platformAppSK == "" && cfg.Platform.AppSecretEnv != "" {
		platformAppSK = strings.TrimSpace(os.Getenv(cfg.Platform.AppSecretEnv))
	}

	return &OpenAPIClient{
		baseURL: baseURL,
		headers: headers,
		http: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
		cfg:             cfg,
		platformBaseURL: strings.TrimRight(cfg.Platform.BaseURL, "/"),
		platformAppKey:  platformAppKey,
		platformAppSK:   platformAppSK,
	}
}

type ListHostsRequest struct {
	Hostname string `json:"hostname,omitempty"`
	ClientIP string `json:"client_ip,omitempty"`
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
}

type listHostsAPIRequest struct {
	Hostname     string `json:"hostname,omitempty"`
	ClientIP     string `json:"client_ip,omitempty"`
	OrgConnectIP string `json:"orgconnectip,omitempty"`
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
}

type ListHostsResponse struct {
	Total int    `json:"total"`
	Pages int    `json:"pages"`
	Hosts []Host `json:"hosts"`
}

type Host struct {
	ID         string `json:"id"`
	ClientID   string `json:"client_id"`
	Hostname   string `json:"hostname"`
	ClientIP   string `json:"client_ip"`
	Platform   int    `json:"platform"`
	Status     string `json:"status"`
	Username   string `json:"username"`
	LastActive int64  `json:"last_active"`
}

type InstructionResult struct {
	TaskID   string `json:"task_id"`
	HostName string `json:"host_name"`
	Repeat   string `json:"repeat"`
}

type ListDetectionsRequest struct {
	Page     int
	PageSize int
}

type ListDetectionsResponse struct {
	Total      int         `json:"total"`
	Detections []Detection `json:"data"`
}

type Detection struct {
	DetectionID string `json:"detection_id"`
	ClientID    string `json:"client_id"`
	ThreatLevel any    `json:"threat_level"`
	HostName    string `json:"host_name"`
	HostStatus  string `json:"host_status"`
	UserName    string `json:"user_name"`
	DealStatus  any    `json:"deal_status"`
	RootName    string `json:"root_name"`
}

type ListIncidentsRequest struct {
	Page     int
	PageSize int
	ClientID string
}

type ListIncidentsResponse struct {
	Total     int        `json:"total"`
	Incidents []Incident `json:"data"`
}

type Incident struct {
	IncidentID   string  `json:"incident_id"`
	IncidentName string  `json:"incident_name"`
	Score        float64 `json:"score"`
	ClientID     string  `json:"client_id"`
	Status       int     `json:"status"`
	HostName     string  `json:"host_name"`
	HostStatus   string  `json:"host_status"`
	System       string  `json:"system"`
	ExternalIP   string  `json:"external_ip"`
}

type ListLogsRequest struct {
	Page           int
	PageSize       int
	ClientID       string
	OSType         string
	Operation      string
	StartTime      string
	EndTime        string
	FilterField    string
	FilterOperator string
	FilterValue    string
}

type ListLogsResponse struct {
	Total int              `json:"total"`
	Logs  []map[string]any `json:"data"`
}

type IncidentViewRequest struct {
	IncidentID string `json:"incident_id"`
	ClientID   string `json:"client_id"`
}

type DetectionViewRequest struct {
	DetectionID string `json:"detection_id"`
	ClientID    string `json:"client_id"`
	ViewType    string `json:"view_type,omitempty"`
	ProcessUUID string `json:"process_uuid,omitempty"`
}

type apiEnvelope[T any] struct {
	Error   int    `json:"error"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (c *OpenAPIClient) ListHosts(ctx context.Context, req ListHostsRequest) (ListHostsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = c.cfg.DefaultPageSize
	}

	payload := listHostsAPIRequest{
		Hostname:     req.Hostname,
		ClientIP:     req.ClientIP,
		OrgConnectIP: c.cfg.DefaultConnectIP,
		Page:         req.Page,
		Limit:        req.Limit,
	}

	var envelope apiEnvelope[ListHostsResponse]
	if err := c.post(ctx, "/hosts/globalization/list", payload, &envelope); err != nil {
		return ListHostsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListHostsResponse{}, fmt.Errorf("edr list hosts failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) IsolateHost(ctx context.Context, clientID string) (InstructionResult, error) {
	return c.sendInstruction(ctx, clientID, "quarantine_network", "AI 主 Chat 隔离主机")
}

func (c *OpenAPIClient) ReleaseHost(ctx context.Context, clientID string) (InstructionResult, error) {
	return c.sendInstruction(ctx, clientID, "recover_network", "AI 主 Chat 恢复主机")
}

func (c *OpenAPIClient) ListDetections(ctx context.Context, req ListDetectionsRequest) (ListDetectionsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = c.cfg.DefaultPageSize
	}
	payload := map[string]any{
		"page": map[string]any{
			"cur_page":  req.Page,
			"page_size": req.PageSize,
		},
		"sort": []map[string]string{{
			"order":    "desc",
			"sort_key": "end_time",
		}},
	}
	var envelope apiEnvelope[ListDetectionsResponse]
	if err := c.postPlatform(ctx, "/detections/list", payload, &envelope); err != nil {
		return ListDetectionsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListDetectionsResponse{}, fmt.Errorf("platform detections failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) ListIncidents(ctx context.Context, req ListIncidentsRequest) (ListIncidentsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = c.cfg.DefaultPageSize
	}
	payload := map[string]any{
		"page": map[string]any{
			"cur_page":  req.Page,
			"page_size": req.PageSize,
		},
		"sort": []map[string]string{{
			"order":    "desc",
			"sort_key": "end_time",
		}},
	}
	if strings.TrimSpace(req.ClientID) != "" {
		payload["param"] = []map[string]string{{
			"key":   "client_id",
			"value": req.ClientID,
		}}
	}
	var envelope apiEnvelope[ListIncidentsResponse]
	if err := c.postPlatform(ctx, "/incidents/list", payload, &envelope); err != nil {
		return ListIncidentsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListIncidentsResponse{}, fmt.Errorf("platform incidents failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) ListLogs(ctx context.Context, req ListLogsRequest) (ListLogsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = c.cfg.DefaultPageSize
	}
	payload := map[string]any{
		"page": map[string]any{
			"cur_page":  req.Page,
			"page_size": req.PageSize,
		},
		"sort": []map[string]string{{
			"order":    "desc",
			"sort_key": "time",
		}},
	}
	filters := make([]map[string]any, 0, 4)
	if strings.TrimSpace(req.ClientID) != "" {
		filters = append(filters, map[string]any{
			"operator":    "is",
			"field":       "client_id",
			"value":       req.ClientID,
			"is_disabled": 1,
		})
	}
	if strings.TrimSpace(req.OSType) != "" {
		filters = append(filters, map[string]any{
			"operator":    "is",
			"field":       "os_type",
			"value":       req.OSType,
			"is_disabled": 0,
		})
	}
	if strings.TrimSpace(req.Operation) != "" {
		filters = append(filters, map[string]any{
			"operator":    "is",
			"field":       "operation",
			"value":       req.Operation,
			"is_disabled": 0,
		})
	}
	if strings.TrimSpace(req.StartTime) != "" {
		if ts, err := normalizeLogTimestamp(req.StartTime, false); err == nil {
			filters = append(filters, map[string]any{
				"operator":    "gte",
				"field":       "timestamp",
				"value":       ts,
				"is_disabled": 0,
			})
		}
	}
	if strings.TrimSpace(req.EndTime) != "" {
		if ts, err := normalizeLogTimestamp(req.EndTime, true); err == nil {
			filters = append(filters, map[string]any{
				"operator":    "lte",
				"field":       "timestamp",
				"value":       ts,
				"is_disabled": 0,
			})
		}
	}
	if strings.TrimSpace(req.FilterField) != "" && strings.TrimSpace(req.FilterValue) != "" {
		op := strings.TrimSpace(req.FilterOperator)
		if op == "" {
			op = "is"
		}
		filters = append(filters, map[string]any{
			"operator":    op,
			"field":       strings.TrimSpace(req.FilterField),
			"value":       strings.TrimSpace(req.FilterValue),
			"is_disabled": 0,
		})
	}
	if len(filters) > 0 {
		payload["filter"] = filters
	}
	var envelope apiEnvelope[ListLogsResponse]
	if err := c.postPlatform(ctx, "/logs/list", payload, &envelope); err != nil {
		return ListLogsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListLogsResponse{}, fmt.Errorf("platform logs failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func normalizeLogTimestamp(value string, endOfDay bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty time")
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return value, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endOfDay {
			parsed = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		return strconv.FormatInt(parsed.Unix(), 10), nil
	}
	return "", fmt.Errorf("unsupported time format")
}

func (c *OpenAPIClient) ViewIncident(ctx context.Context, req IncidentViewRequest) (map[string]any, error) {
	payload := map[string]any{
		"incident_id": req.IncidentID,
		"client_id":   req.ClientID,
	}
	var envelope apiEnvelope[json.RawMessage]
	if err := c.post(ctx, "/incident/view", payload, &envelope); err != nil {
		return nil, err
	}
	if envelope.Error != 0 {
		return nil, fmt.Errorf("incident view failed: %s", envelope.Message)
	}
	return decodeDetailPayload(envelope.Data, "incident view", envelope.Message)
}

func (c *OpenAPIClient) ViewDetection(ctx context.Context, req DetectionViewRequest) (map[string]any, error) {
	payload := map[string]any{
		"detection_id": req.DetectionID,
		"client_id":    req.ClientID,
	}
	if strings.TrimSpace(req.ViewType) != "" {
		payload["view_type"] = req.ViewType
	}
	if strings.TrimSpace(req.ProcessUUID) != "" {
		payload["process_uuid"] = req.ProcessUUID
	}
	var envelope apiEnvelope[json.RawMessage]
	if err := c.post(ctx, "/detection/view", payload, &envelope); err != nil {
		return nil, err
	}
	if envelope.Error != 0 {
		return nil, fmt.Errorf("detection view failed: %s", envelope.Message)
	}
	return decodeDetailPayload(envelope.Data, "detection view", envelope.Message)
}

func decodeDetailPayload(raw json.RawMessage, operation string, message string) (map[string]any, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return map[string]any{}, nil
	}

	switch raw[0] {
	case '{':
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("%s decode object data failed: %w", operation, err)
		}
		return payload, nil
	case '"':
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, fmt.Errorf("%s decode string data failed: %w", operation, err)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return map[string]any{}, nil
		}
		if strings.HasPrefix(text, "{") {
			var payload map[string]any
			if err := json.Unmarshal([]byte(text), &payload); err == nil {
				return payload, nil
			}
		}
		return nil, fmt.Errorf("%s returned string data: %s%s", operation, shortenForError(text), formatMessageSuffix(message))
	default:
		return nil, fmt.Errorf("%s returned unsupported data shape: %s%s", operation, shortenForError(string(raw)), formatMessageSuffix(message))
	}
}

func formatMessageSuffix(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	return "; message=" + shortenForError(message)
}

func shortenForError(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 200 {
		return text
	}
	return text[:200] + "..."
}

func (c *OpenAPIClient) sendInstruction(ctx context.Context, clientID string, instructionName string, taskName string) (InstructionResult, error) {
	payload := map[string]any{
		"client_id":        clientID,
		"instruction_name": instructionName,
		"instruction_type": 0,
		"task_name":        taskName,
		"is_online":        "1",
	}

	var envelope apiEnvelope[InstructionResult]
	if err := c.post(ctx, "/instructions/send_instruction", payload, &envelope); err != nil {
		return InstructionResult{}, err
	}
	if envelope.Error != 0 {
		return InstructionResult{}, fmt.Errorf("edr send instruction failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) post(ctx context.Context, path string, payload any, out any) error {
	headers := make(map[string]string, len(c.headers)+1)
	for key, value := range c.headers {
		headers[key] = value
	}
	if headers["Authorization"] == "" && c.cfg.Platform.Enabled {
		token, err := c.platformTokenValue(ctx)
		if err != nil {
			return err
		}
		headers["Authorization"] = token
	}
	return c.postWithHeaders(ctx, c.baseURL+path, headers, payload, out)
}

func (c *OpenAPIClient) postPlatform(ctx context.Context, path string, payload any, out any) error {
	token, err := c.platformTokenValue(ctx)
	if err != nil {
		return err
	}
	headers := map[string]string{"Authorization": token}
	if err := c.postWithHeaders(ctx, c.platformBaseURL+path, headers, payload, out); err != nil {
		if strings.Contains(err.Error(), "http 401") || strings.Contains(strings.ToLower(err.Error()), "token") {
			c.invalidatePlatformToken()
			if retryToken, retryErr := c.platformTokenValue(ctx); retryErr == nil {
				headers["Authorization"] = retryToken
				return c.postWithHeaders(ctx, c.platformBaseURL+path, headers, payload, out)
			}
		}
		return err
	}
	return nil
}

func (c *OpenAPIClient) postWithHeaders(ctx context.Context, url string, headers map[string]string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal edr payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create edr request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute edr request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("edr http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode edr response: %w", err)
	}
	return nil
}

func (c *OpenAPIClient) platformTokenValue(ctx context.Context) (string, error) {
	if !c.cfg.Platform.Enabled {
		return "", fmt.Errorf("edr platform api is not enabled")
	}
	if c.platformBaseURL == "" || c.platformAppKey == "" || c.platformAppSK == "" {
		return "", fmt.Errorf("edr platform credentials are incomplete")
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.platformToken != "" && time.Now().Before(c.tokenExpiresAt) {
		return c.platformToken, nil
	}

	timestamp := time.Now().Unix()
	raw := c.platformAppKey + c.platformAppSK + strconv.FormatInt(timestamp, 10)
	sign := fmt.Sprintf("%x", md5.Sum([]byte(raw)))
	payload := map[string]any{
		"sign":    sign,
		"app_key": c.platformAppKey,
		"time":    timestamp,
	}
	var envelope apiEnvelope[struct {
		Token string `json:"token"`
	}]
	if err := c.postWithHeaders(ctx, c.platformBaseURL+"/get_open_api_token", nil, payload, &envelope); err != nil {
		return "", err
	}
	if envelope.Error != 0 {
		return "", fmt.Errorf("get platform token failed: %s", envelope.Message)
	}
	if strings.TrimSpace(envelope.Data.Token) == "" {
		return "", fmt.Errorf("get platform token failed: empty token")
	}
	c.platformToken = envelope.Data.Token
	c.tokenExpiresAt = time.Now().Add(10 * time.Minute)
	return c.platformToken, nil
}

func (c *OpenAPIClient) invalidatePlatformToken() {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.platformToken = ""
	c.tokenExpiresAt = time.Time{}
}
