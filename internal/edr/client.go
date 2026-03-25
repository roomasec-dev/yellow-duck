package edr

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"rm_ai_agent/internal/config"
	"strconv"
	"strings"
	"sync"
	"time"
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

	ListIsolateFiles(ctx context.Context, req ListIsolateFilesRequest) (ListIsolateFilesResponse, error)
	DeleteIsolateFiles(ctx context.Context, guids []string) error
	ReleaseIsolateFiles(ctx context.Context, req ReleaseIsolateFilesRequest) error

	ListIOCs(ctx context.Context, req ListIOCsRequest) (ListIOCsResponse, error)
	AddIOC(ctx context.Context, req AddIOCRequest) error
	UpdateIOC(ctx context.Context, req UpdateIOCRequest) error
	DeleteIOC(ctx context.Context, id string) error
	GetIOCDetail(ctx context.Context, id string) (IOC, error)

	ListTasks(ctx context.Context, req ListTasksRequest) (ListTasksResponse, error)
	GetTaskResult(ctx context.Context, taskID string) (TaskResult, error)
	SendInstruction(ctx context.Context, clientID string, instructionName string, taskName string) (InstructionResult, error)
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
	BusinessType int      `json:"business_type,omitempty"`
	ClientID     string   `json:"client_id,omitempty"`
	ClientIDs    []string `json:"client_ids,omitempty"`
	ClientIP     string   `json:"client_ip,omitempty"`
	Hostname     string   `json:"hostname,omitempty"`
	IPAddress    string   `json:"ip_address,omitempty"`
	MacAddress   string   `json:"mac_address,omitempty"`
	OSType       int      `json:"os_type,omitempty"`
	Platform     string   `json:"platform,omitempty"`
	WinVersion   string   `json:"win_version,omitempty"`
	Importance   int      `json:"importance,omitempty"`
	GIDs         []string `json:"gids,omitempty"`
	Isolate      *bool    `json:"isolate,omitempty"`
	Remarks      string   `json:"remarks,omitempty"`
	IsExport     int      `json:"is_export,omitempty"`
	Type         int      `json:"type,omitempty"`
	OrgConnectIP string   `json:"orgconnectip,omitempty"`
	RMConnectIP  string   `json:"rmconnectip,omitempty"`
	Page         int      `json:"page"`
	Limit        int      `json:"limit"`
}

type listHostsAPIRequest struct {
	BusinessType int      `json:"business_type,omitempty"`
	ClientID     string   `json:"client_id,omitempty"`
	ClientIDs    []string `json:"client_ids,omitempty"`
	ClientIP     string   `json:"client_ip,omitempty"`
	Hostname     string   `json:"hostname,omitempty"`
	IPAddress    string   `json:"ip_address,omitempty"`
	MacAddress   string   `json:"mac_address,omitempty"`
	OSType       int      `json:"os_type,omitempty"`
	Platform     string   `json:"platform,omitempty"`
	WinVersion   string   `json:"win_version,omitempty"`
	Importance   int      `json:"importance,omitempty"`
	GIDs         []string `json:"gids,omitempty"`
	Isolate      *bool    `json:"isolate,omitempty"`
	Remarks      string   `json:"remarks,omitempty"`
	IsExport     int      `json:"is_export,omitempty"`
	Type         int      `json:"type,omitempty"`
	OrgConnectIP string   `json:"orgconnectip,omitempty"`
	RMConnectIP  string   `json:"rmconnectip,omitempty"`
	Page         int      `json:"page"`
	Limit        int      `json:"limit"`
}

type ListHostsResponse struct {
	Total int    `json:"total"`
	Pages int    `json:"pages"`
	Hosts []Host `json:"hosts"`
}

type Host struct {
	ID              string   `json:"id"`
	ClientID        string   `json:"client_id"`
	Hostname        string   `json:"hostname"`
	ClientIP        string   `json:"client_ip"`
	OrgConnectIP    string   `json:"orgconnectip"`
	RMConnectIP     string   `json:"rmconnectip"`
	MacAddress      string   `json:"mac_address"`
	OSType          int      `json:"os_type"`
	OSVersion       string   `json:"os_version"`
	WinVersion      string   `json:"win_version"`
	Platform        int      `json:"platform"`
	Status          string   `json:"status"`
	KeepAliveStatus int      `json:"keep_alive_status"`
	Importance      int      `json:"importance"`
	CreateTime      int64    `json:"create_time"`
	LastActive      int64    `json:"last_active"`
	LastLogonTime   int64    `json:"lastlogontime"`
	Username        string   `json:"username"`
	GroupName       []string `json:"group_name"`
	ClientVersion   string   `json:"client_version"`
	UninstallTask   any      `json:"uninstall_task"`
	Remarks         string   `json:"remarks"`
}

type InstructionResult struct {
	TaskID   string `json:"task_id"`
	HostName string `json:"host_name"`
	Repeat   string `json:"repeat"`
}

// Isolate File Management

type ListIsolateFilesRequest struct {
	ClientID      string           `json:"client_id,omitempty"`
	Hostname      string           `json:"hostname,omitempty"`
	Username      string           `json:"username,omitempty"`
	FileName      string           `json:"file_name,omitempty"`
	Path          string           `json:"path,omitempty"`
	MD5           string           `json:"md5,omitempty"`
	SHA1          string           `json:"sha1,omitempty"`
	RecoverStatus string           `json:"recover_status,omitempty"`
	TaskID        string           `json:"task_id,omitempty"`
	CreateTime    *QuickTimeFilter `json:"create_time,omitempty"`
	Page          int              `json:"page"`
	Limit         int              `json:"limit"`
}

type ListIsolateFilesResponse struct {
	Total   int           `json:"total"`
	Results []IsolateFile `json:"results"`
}

type IsolateFile struct {
	ClientID          string `json:"client_id"`
	CreateTime        int64  `json:"create_time"`
	FileName          string `json:"file_name"`
	GUID              string `json:"guid"`
	Hostname          string `json:"hostname"`
	MD5               string `json:"md5"`
	OrgName           string `json:"org_name"`
	RecoverStatus     int    `json:"recover_status"`
	RemediationStatus int    `json:"remediation_status"`
	SHA1              string `json:"sha1"`
	ShowAction        any    `json:"show_action"`
}

type ReleaseIsolateFilesRequest struct {
	GUIDs          []string `json:"guids"`
	IsAddExclusion bool     `json:"is_add_exclusion"`
	ReleaseAllHash bool     `json:"relase_all_hash"`
}

// IOC Management

type ListIOCsRequest struct {
	Action       string           `json:"action,omitempty"`
	Hash         string           `json:"hash,omitempty"`
	GroupIDs     []int            `json:"group_ids,omitempty"`
	HostType     string           `json:"host_type,omitempty"`
	DateAdd      *QuickTimeFilter `json:"date_add,omitempty"`
	LastModified *QuickTimeFilter `json:"last_modified,omitempty"`
	Page         int              `json:"page"`
	Limit        int              `json:"limit"`
}

type ListIOCsResponse struct {
	Total   int   `json:"total"`
	Results []IOC `json:"results"`
}

type IOC struct {
	Action         string `json:"action"`
	DateAdded      any    `json:"date_added"`
	Description    string `json:"description"`
	DetectionCount any    `json:"detection_count"`
	ExclusionID    any    `json:"exclusion_id"`
	ExpirationDate any    `json:"expiration_date"`
	FileName       string `json:"file_name"`
	GroupIDs       []int  `json:"group_ids"`
	Hash           string `json:"hash"`
	HostType       string `json:"host_type"`
	IOCID          string `json:"ioc_id"`
	LastModified   any    `json:"last_modified"`
	LastSeen       any    `json:"last_seen"`
}

type AddIOCRequest struct {
	Action         string `json:"action"`
	Description    string `json:"description,omitempty"`
	ExpirationDate string `json:"expiration_date,omitempty"`
	FileName       string `json:"file_name,omitempty"`
	GroupIDs       []int  `json:"group_ids,omitempty"`
	Hash           string `json:"hash"`
	HostType       string `json:"host_type,omitempty"`
}

type UpdateIOCRequest struct {
	ID             string `json:"id"`
	Action         string `json:"action,omitempty"`
	Description    string `json:"description,omitempty"`
	ExpirationDate string `json:"expiration_date,omitempty"`
	GroupIDs       []int  `json:"group_ids,omitempty"`
	Hash           string `json:"hash,omitempty"`
	HostType       string `json:"host_type,omitempty"`
}

// Instructions Tasks

type ListTasksRequest struct {
	Page            int         `json:"page"`
	Limit           int         `json:"limit"`
	ID              string      `json:"id,omitempty"`
	ClientID        string      `json:"client_id,omitempty"`
	InstructionName string      `json:"instruction_name,omitempty"`
	User            string      `json:"user,omitempty"`
	Content         string      `json:"content,omitempty"`
	Status          string      `json:"status,omitempty"`
	InstructionType int         `json:"instruction_type,omitempty"`
	PolicyName      string      `json:"policy_name,omitempty"`
	CreateTime      *TimeFilter `json:"create_time,omitempty"`
	UpdateTime      *TimeFilter `json:"update_time,omitempty"`
}

type ListTasksResponse struct {
	Total   int    `json:"total"`
	Results []Task `json:"results"`
}

type Task struct {
	ID                string `json:"id"`
	InstructionName   string `json:"instruction_name"`
	Contents          string `json:"contents"`
	Status            int    `json:"status"`
	OrgName           string `json:"org_name"`
	Key               string `json:"key"`
	ClientID          string `json:"client_id"`
	HostName          string `json:"host_name"`
	ResponseContent   string `json:"response_content"`
	ResponseTime      int64  `json:"response_time"`
	CreateTime        int64  `json:"create_time"`
	UpdateTime        int64  `json:"update_time"`
	ActivityTime      int64  `json:"activity_time"`
	OperationUser     string `json:"operation_user"`
	AllowDownload     bool   `json:"allow_download"`
	ErrorCode         int    `json:"error_code"`
	ErrorMessage      string `json:"error_message"`
	InstructionType   int    `json:"instruction_type"`
	PolicyName        string `json:"policy_name"`
	SearchContent     string `json:"search_content"`
	SearchContentList any    `json:"search_content_list"`
	FileID            string `json:"file_id"`
}

type TaskResult struct {
	CollectTime     int64           `json:"collect_time"`
	HostName        string          `json:"host_name"`
	InstructionName string          `json:"instruction_name"`
	Message         string          `json:"message"`
	ImageDetail     []ImageDetail   `json:"image_detail"`
	Process         []ProcessInfo   `json:"process"`
	ProcessDetail   []ProcessDetail `json:"process_detail"`
}

type ImageDetail struct {
	ImagePath      string `json:"image_path"`
	ImageLevel     int    `json:"image_level"`
	ImageSHA1      string `json:"image_sha1"`
	ImageSignature string `json:"image_signature"`
	IsSystem       int    `json:"is_system"`
}

type ProcessInfo struct {
	IsSystem  int    `json:"is_system"`
	Level     int    `json:"level"`
	Path      string `json:"path"`
	PID       int    `json:"pid"`
	PName     string `json:"pname"`
	SHA1      string `json:"sha1"`
	Signature string `json:"signature"`
}

type ProcessDetail struct {
	ThreadID       int    `json:"thread_id"`
	ThreadRIP      string `json:"thread_rip"`
	ThreadSymbol   string `json:"thread_symbol"`
	ThreadFeature  int    `json:"thread_feature"`
	CodeFeature    int    `json:"code_feature"`
	AddressFeature int    `json:"address_feature"`
}

type TimeFilter struct {
	TimeRange *TimeRange    `json:"time_range,omitempty"`
	QuickTime *QuickTimeVal `json:"quick_time,omitempty"`
}

type TimeRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type QuickTimeFilter struct {
	QuickTime *QuickTimeVal `json:"quick_time,omitempty"`
}

type QuickTimeVal struct {
	TimeNum  int    `json:"time_num"`
	TimeSpan string `json:"time_span"`
	TimeType string `json:"time_type"`
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
		BusinessType: req.BusinessType,
		ClientID:     req.ClientID,
		ClientIDs:    req.ClientIDs,
		ClientIP:     req.ClientIP,
		Hostname:     req.Hostname,
		IPAddress:    req.IPAddress,
		MacAddress:   req.MacAddress,
		OSType:       req.OSType,
		Platform:     req.Platform,
		WinVersion:   req.WinVersion,
		Importance:   req.Importance,
		GIDs:         req.GIDs,
		Isolate:      req.Isolate,
		Remarks:      req.Remarks,
		IsExport:     req.IsExport,
		Type:         req.Type,
		OrgConnectIP: req.OrgConnectIP,
		RMConnectIP:  req.RMConnectIP,
		Page:         req.Page,
		Limit:        req.Limit,
	}
	if payload.OrgConnectIP == "" {
		payload.OrgConnectIP = c.cfg.DefaultConnectIP
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
	log.Printf("ListLogs req: %+v", req)
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

	jsonBytes, _ := json.MarshalIndent(payload, "", "  ")
	log.Printf("ListLogs request body:\n%s", string(jsonBytes))

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

func (c *OpenAPIClient) ListIsolateFiles(ctx context.Context, req ListIsolateFilesRequest) (ListIsolateFilesResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = c.cfg.DefaultPageSize
	}
	payload := map[string]any{
		"page":  req.Page,
		"limit": req.Limit,
	}
	if req.ClientID != "" {
		payload["client_id"] = req.ClientID
	}
	if req.Hostname != "" {
		payload["hostname"] = req.Hostname
	}
	if req.Username != "" {
		payload["username"] = req.Username
	}
	if req.FileName != "" {
		payload["file_name"] = req.FileName
	}
	if req.Path != "" {
		payload["path"] = req.Path
	}
	if req.MD5 != "" {
		payload["md5"] = req.MD5
	}
	if req.SHA1 != "" {
		payload["sha1"] = req.SHA1
	}
	if req.RecoverStatus != "" {
		payload["recover_status"] = req.RecoverStatus
	}
	if req.TaskID != "" {
		payload["task_id"] = req.TaskID
	}
	if req.CreateTime != nil {
		payload["create_time"] = req.CreateTime
	}

	var envelope apiEnvelope[ListIsolateFilesResponse]
	if err := c.post(ctx, "/isolate_file/get_list", payload, &envelope); err != nil {
		return ListIsolateFilesResponse{}, err
	}
	if envelope.Error != 0 {
		return ListIsolateFilesResponse{}, fmt.Errorf("isolate file list failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) DeleteIsolateFiles(ctx context.Context, guids []string) error {
	payload := map[string]any{"guids": guids}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/isolate_file/delete", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("delete isolate files failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) ReleaseIsolateFiles(ctx context.Context, req ReleaseIsolateFilesRequest) error {
	payload := map[string]any{
		"guids":            req.GUIDs,
		"is_add_exclusion": req.IsAddExclusion,
		"relase_all_hash":  req.ReleaseAllHash,
	}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/isolate_file/release", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("release isolate files failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) ListIOCs(ctx context.Context, req ListIOCsRequest) (ListIOCsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = c.cfg.DefaultPageSize
	}
	payload := map[string]any{
		"page":  req.Page,
		"limit": req.Limit,
	}
	if req.Action != "" {
		payload["action"] = req.Action
	}
	if req.Hash != "" {
		payload["hash"] = req.Hash
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.HostType != "" {
		payload["host_type"] = req.HostType
	}
	if req.DateAdd != nil {
		payload["date_add"] = req.DateAdd
	}
	if req.LastModified != nil {
		payload["last_modified"] = req.LastModified
	}

	var envelope apiEnvelope[ListIOCsResponse]
	if err := c.post(ctx, "/configure/ioc/list", payload, &envelope); err != nil {
		return ListIOCsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListIOCsResponse{}, fmt.Errorf("ioc list failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) AddIOC(ctx context.Context, req AddIOCRequest) error {
	payload := map[string]any{
		"action": req.Action,
		"hash":   req.Hash,
	}
	if req.Description != "" {
		payload["description"] = req.Description
	}
	if req.ExpirationDate != "" {
		payload["expiration_date"] = req.ExpirationDate
	}
	if req.FileName != "" {
		payload["file_name"] = req.FileName
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.HostType != "" {
		payload["host_type"] = req.HostType
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioc/add", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("add ioc failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) UpdateIOC(ctx context.Context, req UpdateIOCRequest) error {
	payload := map[string]any{
		"id": req.ID,
	}
	if req.Action != "" {
		payload["action"] = req.Action
	}
	if req.Description != "" {
		payload["description"] = req.Description
	}
	if req.ExpirationDate != "" {
		payload["expiration_date"] = req.ExpirationDate
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.Hash != "" {
		payload["hash"] = req.Hash
	}
	if req.HostType != "" {
		payload["host_type"] = req.HostType
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioc/update", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("update ioc failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) DeleteIOC(ctx context.Context, id string) error {
	payload := map[string]any{"id": id}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioc/delete", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("delete ioc failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) GetIOCDetail(ctx context.Context, id string) (IOC, error) {
	payload := map[string]any{"id": id}
	var envelope apiEnvelope[IOC]
	if err := c.post(ctx, "/configure/ioc/detail", payload, &envelope); err != nil {
		return IOC{}, err
	}
	if envelope.Error != 0 {
		return IOC{}, fmt.Errorf("ioc detail failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) ListTasks(ctx context.Context, req ListTasksRequest) (ListTasksResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = c.cfg.DefaultPageSize
	}
	payload := map[string]any{
		"page":  req.Page,
		"limit": req.Limit,
	}
	if req.ID != "" {
		payload["id"] = req.ID
	}
	if req.ClientID != "" {
		payload["client_id"] = req.ClientID
	}
	if req.InstructionName != "" {
		payload["instruction_name"] = req.InstructionName
	}
	if req.User != "" {
		payload["user"] = req.User
	}
	if req.Content != "" {
		payload["content"] = req.Content
	}
	if req.Status != "" {
		payload["status"] = req.Status
	}
	if req.InstructionType != 0 {
		payload["instruction_type"] = req.InstructionType
	}
	if req.PolicyName != "" {
		payload["policy_name"] = req.PolicyName
	}
	if req.CreateTime != nil {
		payload["create_time"] = req.CreateTime
	}
	if req.UpdateTime != nil {
		payload["update_time"] = req.UpdateTime
	}

	var envelope apiEnvelope[ListTasksResponse]
	if err := c.post(ctx, "/instructions/tasks", payload, &envelope); err != nil {
		return ListTasksResponse{}, err
	}
	if envelope.Error != 0 {
		return ListTasksResponse{}, fmt.Errorf("list tasks failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) GetTaskResult(ctx context.Context, taskID string) (TaskResult, error) {
	payload := map[string]any{"task_id": taskID}
	var envelope apiEnvelope[TaskResult]
	if err := c.post(ctx, "/instructions/task_result", payload, &envelope); err != nil {
		return TaskResult{}, err
	}
	if envelope.Error != 0 {
		return TaskResult{}, fmt.Errorf("task result failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) SendInstruction(ctx context.Context, clientID string, instructionName string, taskName string) (InstructionResult, error) {
	return c.sendInstruction(ctx, clientID, instructionName, taskName)
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
