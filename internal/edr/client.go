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
	AddHostBlacklist(ctx context.Context, clientIDs []string, reason string) error
	RemoveHost(ctx context.Context, clientIDs []string) error
	IsolateHost(ctx context.Context, clientID string, time int) (InstructionResult, error)
	ReleaseHost(ctx context.Context, clientID string) (InstructionResult, error)
	ListIncidents(ctx context.Context, req ListIncidentsRequest) (ListIncidentsResponse, error)
	BatchDealIncident(ctx context.Context, req BatchDealIncidentRequest) (BatchDealIncidentResponse, error)
	IncidentR2Summary(ctx context.Context, incidentID string) (IncidentR2SummaryResponse, error)
	ListDetections(ctx context.Context, req ListDetectionsRequest) (ListDetectionsResponse, error)
	ListDetectionsProxy(ctx context.Context, req ListDetectionsProxyRequest) (ListDetectionsProxyResponse, error)
	UpdateDetectionStatus(ctx context.Context, req UpdateDetectionStatusRequest) error
	ListEventLogAlarms(ctx context.Context, req ListEventLogAlarmsRequest) (ListEventLogAlarmsResponse, error)
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
	SendInstruction(ctx context.Context, req SendInstructionRequest) (InstructionResult, error)

	// Virus Statistics
	ListVirusByHost(ctx context.Context, req ListVirusByHostRequest) (ListVirusByHostResponse, error)
	ListVirusByHash(ctx context.Context, req ListVirusByHashRequest) (ListVirusByHashResponse, error)
	ListVirusHashHosts(ctx context.Context, req ListVirusHashHostsRequest) (ListVirusHashHostsResponse, error)
	ListVirusScanRecords(ctx context.Context, req ListVirusScanRecordsRequest) (ListVirusScanRecordsResponse, error)

	// Plan Management
	AddPlan(ctx context.Context, req AddPlanRequest) error
	EditPlan(ctx context.Context, req EditPlanRequest) error
	CancelPlan(ctx context.Context, rid string) error
	ListPlans(ctx context.Context, req ListPlansRequest) (ListPlansResponse, error)

	// Client Setting (Host Offline)
	GetHostOfflineConf(ctx context.Context) (HostOfflineConf, error)
	SaveHostOfflineConf(ctx context.Context, req SaveHostOfflineConfRequest) error

	// IOA Configuration
	ListIOAs(ctx context.Context, req ListIOAsRequest) (ListIOAsResponse, error)
	AddIOA(ctx context.Context, req AddIOARequest) error
	UpdateIOA(ctx context.Context, req UpdateIOARequest) error
	DeleteIOA(ctx context.Context, id string) error
	ListIOAAuditLogs(ctx context.Context, req ListIOAAuditLogsRequest) (ListIOAAuditLogsResponse, error)

	// IOA Network Exclusion
	ListIOANetworks(ctx context.Context, req ListIOANetworksRequest) (ListIOANetworksResponse, error)
	AddIOANetwork(ctx context.Context, req AddIOANetworkRequest) error
	UpdateIOANetwork(ctx context.Context, req UpdateIOANetworkRequest) error
	DeleteIOANetwork(ctx context.Context, id string) error

	// Strategy Management
	GetStrategySingle(ctx context.Context, strategyType string) (Strategy, error)
	ListStrategies(ctx context.Context, req ListStrategiesRequest) (ListStrategiesResponse, error)
	GetStrategyDetail(ctx context.Context, req GetStrategyDetailRequest) (Strategy, error)
	CreateStrategy(ctx context.Context, req CreateStrategyRequest) (CreateStrategyResponse, error)
	UpdateStrategy(ctx context.Context, req UpdateStrategyRequest) error
	DeleteStrategy(ctx context.Context, strategyID string, strategyType string) error
	GetStrategyState(ctx context.Context) (StrategyState, error)
	SortStrategies(ctx context.Context, sortIDs []string, strategyType string) error
	UpdateStrategyStatus(ctx context.Context, req UpdateStrategyStatusRequest) error

	// Instruction Policy (Auto Response)
	ListInstructionPolicies(ctx context.Context, req ListInstructionPoliciesRequest) (ListInstructionPoliciesResponse, error)
	UpdateInstructionPolicy(ctx context.Context, req UpdateInstructionPolicyRequest) error
	SaveInstructionPolicyStatus(ctx context.Context, req SaveInstructionPolicyStatusRequest) (SaveInstructionPolicyStatusResponse, error)
	DeleteInstructionPolicy(ctx context.Context, rid string) (DeleteInstructionPolicyResponse, error)
	SortInstructionPolicies(ctx context.Context, rids []string) error
	AddInstructionPolicy(ctx context.Context, req AddInstructionPolicyRequest) (AddInstructionPolicyResponse, error)
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
	Repeat   bool   `json:"repeat"`
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

type SendInstructionRequest struct {
	ClientID        string       `json:"client_id"`
	InstructionName string       `json:"instruction_name"`
	IsOnline        int          `json:"is_online,omitempty"`    // for list_ps
	IsBatch         int          `json:"is_batch,omitempty"`     // for get_suspicious_file
	BatchParams     []BatchParam `json:"batch_params,omitempty"` // for get_suspicious_file
	Params          *Params      `json:"params,omitempty"`       // for quarantine_network
}

type Params struct {
	Time int `json:"time,omitempty"` // for quarantine_network
}

type BatchParam struct {
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
	SHA1 string `json:"sha1,omitempty"`
	Pid  int    `json:"pid,omitempty"`
}

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

// Virus Scan Management

// Virus Statistics

type ListVirusByHostRequest struct {
	ClientID        string      `json:"client_id,omitempty"`
	Username        string      `json:"username,omitempty"`
	HostName        string      `json:"host_name,omitempty"`
	Importance      int         `json:"importance,omitempty"`
	MacAddress      string      `json:"mac_address,omitempty"`
	ClientIP        string      `json:"client_ip,omitempty"`
	RMConnectIP     string      `json:"rmconnectip,omitempty"`
	Status          int         `json:"status,omitempty"`
	LastCheckedTime *TimeFilter `json:"last_checked_time,omitempty"`
	Page            int         `json:"page"`
	Limit           int         `json:"limit"`
}

type VirusByHost struct {
	HostName         string `json:"host_name"`
	ClientID         string `json:"client_id"`
	Status           int    `json:"status"`
	Username         string `json:"username"`
	Importance       int    `json:"importance"`
	ClientIP         string `json:"client_ip"`
	RMConnectIP      string `json:"rm_connect_ip"`
	MacAddress       string `json:"mac_address"`
	VirusFileCount   int    `json:"virus_file_count"`
	VirusMemoryCount int    `json:"virus_memory_count"`
	LastCheckedTime  int64  `json:"last_checked_time"`
	LastActive       int64  `json:"last_active"`
	HostStatus       string `json:"host_status"`
	Path             string `json:"path"`
	SHA1             string `json:"sha1,omitempty"`
	MD5              string `json:"md5,omitempty"`
}

type ListVirusByHostResponse struct {
	Total   int           `json:"total"`
	Results []VirusByHost `json:"results"`
}

type ListVirusByHashRequest struct {
	LastCheckedTime *TimeFilter `json:"last_checked_time,omitempty"`
	Name            string      `json:"name,omitempty"`
	SHA1            string      `json:"sha1,omitempty"`
	MD5             string      `json:"md5,omitempty"`
	Page            int         `json:"page"`
	Limit           int         `json:"limit"`
}

type VirusByHash struct {
	Name      string   `json:"name"`
	SHA1      string   `json:"sha1"`
	MD5       string   `json:"md5"`
	Size      int64    `json:"size"`
	HostCount int      `json:"host_count"`
	ClientIDs []string `json:"client_ids"`
	EndTime   int64    `json:"end_time"`
	ID        string   `json:"id"`
}

type ListVirusByHashResponse struct {
	Total   int           `json:"total"`
	Results []VirusByHash `json:"results"`
}

type ListVirusHashHostsRequest struct {
	SHA1            string      `json:"sha1,omitempty"`
	ClientID        string      `json:"client_id,omitempty"`
	Username        string      `json:"username,omitempty"`
	HostName        string      `json:"host_name,omitempty"`
	Importance      int         `json:"importance,omitempty"`
	MacAddress      string      `json:"mac_address,omitempty"`
	ClientIP        string      `json:"client_ip,omitempty"`
	RMConnectIP     string      `json:"rmconnectip,omitempty"`
	Status          int         `json:"status,omitempty"`
	HostStatus      string      `json:"host_status,omitempty"`
	Path            string      `json:"path,omitempty"`
	LastCheckedTime *TimeFilter `json:"last_checked_time,omitempty"`
	Page            int         `json:"page"`
	Limit           int         `json:"limit"`
}

type VirusHashHost struct {
	HostName         string `json:"host_name"`
	ClientID         string `json:"client_id"`
	Status           int    `json:"status"`
	Username         string `json:"username"`
	Importance       int    `json:"importance"`
	ClientIP         string `json:"client_ip"`
	RMConnectIP      string `json:"rm_connect_ip"`
	MacAddress       string `json:"mac_address"`
	VirusFileCount   int    `json:"virus_file_count"`
	VirusMemoryCount int    `json:"virus_memory_count"`
	LastCheckedTime  int64  `json:"last_checked_time"`
	LastActive       int64  `json:"last_active"`
	HostStatus       string `json:"host_status"`
	Path             string `json:"path"`
	SHA1             string `json:"sha1"`
	MD5              string `json:"md5"`
}

type ListVirusHashHostsResponse struct {
	Total   int             `json:"total"`
	Results []VirusHashHost `json:"results"`
}

type ListVirusScanRecordsRequest struct {
	Page           int         `json:"page,omitempty"`
	Limit          int         `json:"limit,omitempty"`
	RID            string      `json:"rid,omitempty"`
	TaskID         string      `json:"task_id,omitempty"`
	ExecutionBatch string      `json:"execution_batch,omitempty"`
	HostName       string      `json:"host_name,omitempty"`
	ClientID       string      `json:"client_id,omitempty"`
	ScanType       string      `json:"scan_type,omitempty"`
	Status         string      `json:"status,omitempty"`
	StartTime      *TimeFilter `json:"start_time,omitempty"`
	EndTime        *TimeFilter `json:"end_time,omitempty"`
}

type VirusScanRecord struct {
	ID             string `json:"id"`
	TaskID         string `json:"task_id"`
	RID            string `json:"rid"`
	OrgName        string `json:"org_name"`
	ExecutionBatch string `json:"execution_batch"`
	ClientID       string `json:"client_id"`
	HostName       string `json:"host_name"`
	ScanType       string `json:"scan_type"`
	Contents       string `json:"contents"`
	Status         int    `json:"status"`
	CreateTime     int64  `json:"create_time"`
	StartTime      int64  `json:"start_time"`
	EndTime        int64  `json:"end_time"`
	UpdateTime     int64  `json:"update_time"`
	VirusFileNum   int    `json:"virus_file_num"`
	MemoryVirusNum int    `json:"memory_virus_num"`
	ResponseTime   int64  `json:"response_time"`
	PlanName       string `json:"plan_name"`
	HostStatus     string `json:"host_status"`
}

type ListVirusScanRecordsResponse struct {
	Total   int               `json:"total"`
	Results []VirusScanRecord `json:"results"`
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

// Plan Management

type AddPlanRequest struct {
	RID              string         `json:"rid,omitempty"`
	ScanType         int            `json:"scan_type"`
	PlanName         string         `json:"plan_name,omitempty"`
	PlanType         int            `json:"plan_type"`
	Scope            int            `json:"scope"`
	Contents         map[string]any `json:"contents,omitempty"`
	ExecuteStartTime int64          `json:"execute_start_time,omitempty"`
	ExecuteCycle     int            `json:"execute_cycle,omitempty"`
	RepeatCycle      []int          `json:"repeat_cycle,omitempty"`
	ExecutionTime    string         `json:"execution_time,omitempty"`
	GroupIDs         []int          `json:"group_ids,omitempty"`
	Type             string         `json:"type"`
	DeviceClientIDs  []string       `json:"device_client_ids,omitempty"`
	ExpiredSetting   int            `json:"expired_setting,omitempty"`
	ExpiredTime      int64          `json:"expired_time,omitempty"`
	SearchContent    []string       `json:"search_content,omitempty"`
}

type EditPlanRequest struct {
	RID              string         `json:"rid,omitempty"`
	ClientID         string         `json:"client_id,omitempty"`
	ScanType         int            `json:"scan_type,omitempty"`
	PlanName         string         `json:"plan_name,omitempty"`
	PlanType         int            `json:"plan_type,omitempty"`
	Scope            int            `json:"scope,omitempty"`
	Contents         map[string]any `json:"contents,omitempty"`
	ExecuteStartTime int64          `json:"execute_start_time,omitempty"`
	ExecuteCycle     int            `json:"execute_cycle,omitempty"`
	RepeatCycle      []int          `json:"repeat_cycle,omitempty"`
	ExecutionTime    string         `json:"execution_time,omitempty"`
	GroupIDs         []int          `json:"group_ids,omitempty"`
	Type             string         `json:"type,omitempty"`
	DeviceClientIDs  []string       `json:"device_client_ids,omitempty"`
	ExpiredSetting   int            `json:"expired_setting,omitempty"`
	ExpiredTime      int64          `json:"expired_time,omitempty"`
	SearchContent    []string       `json:"search_content,omitempty"`
}

type ListPlansRequest struct {
	ScanType      int    `json:"scan_type,omitempty"`
	PlanType      int    `json:"plan_type,omitempty"`
	Type          string `json:"type,omitempty"`
	SearchContent string `json:"search_content,omitempty"`
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
}

type Plan struct {
	RID              string   `json:"rid"`
	OrgName          string   `json:"org_name"`
	ClientID         string   `json:"client_id"`
	PlanName         string   `json:"plan_name"`
	PlanType         int      `json:"plan_type"`
	ScanType         int      `json:"scan_type"`
	Scope            int      `json:"scope"`
	ScopeContent     string   `json:"scope_content"`
	Contents         string   `json:"contents"`
	ExecuteStartTime int64    `json:"execute_start_time"`
	ExecuteCycle     int      `json:"execute_cycle"`
	RepeatCycle      []int    `json:"repeat_cycle"`
	ExecutionTime    string   `json:"execution_time"`
	GroupIDs         []int    `json:"group_ids"`
	Type             string   `json:"type"`
	DeviceClientIDs  []string `json:"device_client_ids"`
	ExpiredSetting   int      `json:"expired_setting"`
	ExpiredTime      int64    `json:"expired_time"`
	SearchContent    []string `json:"search_content"`
	CreateTime       int64    `json:"create_time"`
	UpdateTime       int64    `json:"update_time"`
	OperationUser    string   `json:"operation_user"`
	OperationUID     string   `json:"operation_uid"`
	Status           int      `json:"status"`
	IsDeleted        int      `json:"is_deleted"`
}

type ListPlansResponse struct {
	Total int    `json:"total"`
	Items []Plan `json:"items"`
}

// Instruction Policy (Auto Response)

type ListInstructionPoliciesRequest struct {
	PolicyType    int         `json:"policy_type,omitempty"`
	Name          string      `json:"name,omitempty"`
	OperationUser string      `json:"operation_user,omitempty"`
	Scopes        string      `json:"scopes,omitempty"`
	Action        int         `json:"action,omitempty"`
	Status        int         `json:"status,omitempty"`
	CreateTime    *TimeFilter `json:"create_time,omitempty"`
	UpdateTime    *TimeFilter `json:"update_time,omitempty"`
}

type InstructionPolicy struct {
	RID           string                     `json:"rid"`
	PolicyType    int                        `json:"policy_type"`
	Name          string                     `json:"name"`
	NameEn        string                     `json:"name_en"`
	ConditionList InstructionPolicyCondition `json:"condition_list"`
	OperatorList  InstructionPolicyCondition `json:"operator_list,omitempty"`
	Action        []int                      `json:"action"`
	FuncList      []string                   `json:"func_list,omitempty"`
	ClientID      string                     `json:"client_id"`
	Scope         int                        `json:"scope"`
	ScopeContent  string                     `json:"scope_content"`
	GroupIDs      []int                      `json:"group_ids"`
	TQGroup       TQGroup                    `json:"tq_group"`
	OperationUser string                     `json:"operation_user"`
	OperationUID  string                     `json:"operation_uid,omitempty"`
	CreateTime    int64                      `json:"create_time"`
	UpdateTime    int64                      `json:"update_time"`
	Status        int                        `json:"status"`
	TaskNum       int                        `json:"task_num"`
	TaskStartTime int64                      `json:"task_start_time"`
	TaskEndTime   int64                      `json:"task_end_time"`
	Index         int                        `json:"index,omitempty"`
	HitContinue   string                     `json:"hit_continue,omitempty"`
	IsDeleted     int                        `json:"is_deleted,omitempty"`
	SubType       string                     `json:"sub_type,omitempty"`
	HaveResp      bool                       `json:"have_resp,omitempty"`
}

type InstructionPolicyCondition struct {
	Sets    []InstructionPolicySet `json:"sets"`
	Version string                 `json:"version"`
	Metas   map[string]any         `json:"metas"`
}

type InstructionPolicySet struct {
	SubSetsLogical string                  `json:"sub_sets_logical"`
	SubSets        []InstructionPolicySet  `json:"sub_sets,omitempty"`
	AccessRules    []InstructionPolicyRule `json:"access_rules,omitempty"`
	Metas          map[string]any          `json:"metas"`
}

type InstructionPolicyRule struct {
	Key           string `json:"key"`
	Value         string `json:"value"`
	CompareMethod string `json:"compare_method"`
}

type TQGroup struct {
	Groups   []int  `json:"groups"`
	ShowData string `json:"show_data"`
}

type ListInstructionPoliciesResponse struct {
	Result []InstructionPolicy `json:"result"`
}

type UpdateInstructionPolicyRequest struct {
	RID           string                     `json:"rid"`
	Name          string                     `json:"name"`
	ConditionList InstructionPolicyCondition `json:"condition_list"`
	Action        []int                      `json:"action"`
	Scope         int                        `json:"scope"`
	ClientID      string                     `json:"client_id"`
	GroupIDs      []int                      `json:"group_ids"`
	PolicyType    int                        `json:"policy_type"`
	TQGroup       TQGroup                    `json:"tq_group"`
	ScopeContent  string                     `json:"scope_content"`
	OperationUser string                     `json:"operation_user"`
	CreateTime    int64                      `json:"create_time"`
	UpdateTime    int64                      `json:"update_time"`
	Status        int                        `json:"status"`
	TaskNum       int                        `json:"task_num"`
	TaskStartTime int64                      `json:"task_start_time"`
	TaskEndTime   int64                      `json:"task_end_time"`
	Index         int                        `json:"index"`
}

type SaveInstructionPolicyStatusRequest struct {
	RID  string   `json:"rid"`
	RIDs []string `json:"rids"`
}

type SaveInstructionPolicyStatusResponse struct {
	Name string `json:"name"`
}

type DeleteInstructionPolicyResponse struct {
	RID           string                     `json:"rid"`
	HitContinue   string                     `json:"hit_continue"`
	IsDeleted     int                        `json:"is_deleted"`
	PolicyType    int                        `json:"policy_type"`
	SubType       string                     `json:"sub_type"`
	HaveResp      bool                       `json:"have_resp"`
	Name          string                     `json:"name"`
	NameEn        string                     `json:"name_en"`
	ConditionList InstructionPolicyCondition `json:"condition_list"`
	OperatorList  InstructionPolicyCondition `json:"operator_list"`
	Action        []int                      `json:"action"`
	FuncList      []string                   `json:"func_list"`
	ClientID      string                     `json:"client_id"`
	GroupIDs      []int                      `json:"group_ids"`
	Scope         int                        `json:"scope"`
	TQGroup       TQGroup                    `json:"tq_group"`
	ScopeContent  string                     `json:"scope_content"`
	OperationUser string                     `json:"operation_user"`
	OperationUID  string                     `json:"operation_uid"`
	CreateTime    int64                      `json:"create_time"`
	UpdateTime    int64                      `json:"update_time"`
}

type AddInstructionPolicyRequest struct {
	Name          string                     `json:"name"`
	ConditionList InstructionPolicyCondition `json:"condition_list"`
	Action        []int                      `json:"action"`
	Scope         int                        `json:"scope"`
	ClientID      string                     `json:"client_id"`
	GroupIDs      []int                      `json:"group_ids"`
	PolicyType    int                        `json:"policy_type"`
	TQGroup       TQGroup                    `json:"tq_group"`
	ScopeContent  string                     `json:"scope_content"`
	OperationUser string                     `json:"operation_user"`
	Status        int                        `json:"status"`
	TaskNum       int                        `json:"task_num"`
	TaskStartTime int64                      `json:"task_start_time"`
	TaskEndTime   int64                      `json:"task_end_time"`
	Index         int                        `json:"index"`
}

type AddInstructionPolicyResponse struct {
	RID string `json:"rid"`
}

// Client Setting (Host Offline)

type HostOfflineConf struct {
	ID         string             `json:"id"`
	OrgName    string             `json:"org_name"`
	Status     int                `json:"status"`
	Type       string             `json:"type"`
	Setting    HostOfflineSetting `json:"setting"`
	CreateTime int64              `json:"create_time"`
	UpdateTime int64              `json:"update_time"`
}

type HostOfflineSetting struct {
	Timeout int `json:"offline_days"`
}

type SaveHostOfflineConfRequest struct {
	Status  int                `json:"status"`
	Setting HostOfflineSetting `json:"setting"`
}

// IOA Configuration

type ListIOAsRequest struct {
	CommandLine  string      `json:"command_line,omitempty"`
	FileName     string      `json:"file_name,omitempty"`
	GroupIDs     []int       `json:"group_ids,omitempty"`
	HostType     string      `json:"host_type,omitempty"`
	IOAName      string      `json:"ioa_name,omitempty"`
	LastModified *TimeFilter `json:"last_modified,omitempty"`
	ModifiedBy   string      `json:"modified_by,omitempty"`
	Name         string      `json:"name,omitempty"`
	Page         int         `json:"page"`
	Limit        int         `json:"limit"`
	TAID         string      `json:"ta_id,omitempty"`
	TID          string      `json:"t_id,omitempty"`
}

type IOA struct {
	CommandLine  string `json:"command_line"`
	CreateTime   int64  `json:"create_time"`
	Description  string `json:"description"`
	ExclusionID  string `json:"exclusion_id"`
	FileName     string `json:"file_name"`
	GroupIDs     []int  `json:"group_ids"`
	HostType     string `json:"host_type"`
	IOAID        string `json:"ioa_id"`
	IOAName      string `json:"ioa_name"`
	ModifiedByID string `json:"modified_by_id"`
	OperateUser  string `json:"operate_user"`
	Severity     string `json:"severity"`
	TAName       string `json:"ta_name"`
	TName        string `json:"t_name"`
	TAID         string `json:"taid"`
	TID          string `json:"tid"`
	UpdateTime   int64  `json:"update_time"`
}

type ListIOAsResponse struct {
	Total   int   `json:"total"`
	Results []IOA `json:"results"`
}

type AddIOARequest struct {
	CommandLine   string `json:"command_line,omitempty"`
	Description   string `json:"description,omitempty"`
	ExclusionName string `json:"exclusion_name,omitempty"`
	FileName      string `json:"file_name,omitempty"`
	GroupIDs      []int  `json:"group_ids,omitempty"`
	HostType      string `json:"host_type,omitempty"`
	IOAID         string `json:"ioa_id,omitempty"`
	Severity      string `json:"severity,omitempty"`
	TAID          string `json:"ta_id,omitempty"`
	TID           string `json:"t_id,omitempty"`
}

type UpdateIOARequest struct {
	ID            string `json:"id"`
	CommandLine   string `json:"command_line,omitempty"`
	Description   string `json:"description,omitempty"`
	ExclusionName string `json:"exclusion_name,omitempty"`
	FileName      string `json:"file_name,omitempty"`
	GroupIDs      []int  `json:"group_ids,omitempty"`
	HostType      string `json:"host_type,omitempty"`
	TAID          string `json:"ta_id,omitempty"`
	TID           string `json:"t_id,omitempty"`
}

type ListIOAAuditLogsRequest struct {
	CommandLine string      `json:"command_line,omitempty"`
	EventTime   *TimeFilter `json:"event_time,omitempty"`
	FileName    string      `json:"file_name,omitempty"`
	HostName    string      `json:"host_name,omitempty"`
	IOAName     string      `json:"ioa_name,omitempty"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
}

type IOAAuditLog struct {
	CommandLine string `json:"command_line"`
	EventTime   int64  `json:"event_time"`
	FileName    string `json:"file_name"`
	HostName    string `json:"host_name"`
	ID          string `json:"id"`
	IOAName     string `json:"ioa_name"`
}

type ListIOAAuditLogsResponse struct {
	Total   int           `json:"total"`
	Results []IOAAuditLog `json:"results"`
}

// IOA Network Exclusion

type ListIOANetworksRequest struct {
	GroupIDs     []int       `json:"group_ids,omitempty"`
	HostType     string      `json:"host_type,omitempty"`
	IP           string      `json:"ip,omitempty"`
	LastModified *TimeFilter `json:"last_modified,omitempty"`
	ModifiedBy   string      `json:"modified_by,omitempty"`
	Name         string      `json:"name,omitempty"`
	Page         int         `json:"page"`
	Limit        int         `json:"limit"`
}

type IOANetwork struct {
	CreateTime    int64  `json:"create_time"`
	ExclusionName string `json:"exclusion_name"`
	GroupIDs      []int  `json:"group_ids"`
	HostType      string `json:"host_type"`
	ID            string `json:"id"`
	IP            string `json:"ip"`
	IsSystem      bool   `json:"is_system"`
	ModifiedBy    string `json:"modified_by"`
	ModifiedByID  string `json:"modified_by_id"`
	OperateUser   string `json:"operate_user"`
	OrgName       string `json:"org_name"`
	UpdateTime    int64  `json:"update_time"`
}

type ListIOANetworksResponse struct {
	Total   int          `json:"total"`
	Results []IOANetwork `json:"results"`
}

type AddIOANetworkRequest struct {
	ExclusionName string `json:"exclusion_name"`
	GroupIDs      []int  `json:"group_ids,omitempty"`
	HostType      string `json:"host_type,omitempty"`
	IP            string `json:"ip,omitempty"`
}

type UpdateIOANetworkRequest struct {
	ID            string `json:"id"`
	ExclusionName string `json:"exclusion_name,omitempty"`
	GroupIDs      []int  `json:"group_ids,omitempty"`
	HostType      string `json:"host_type,omitempty"`
	IP            string `json:"ip,omitempty"`
}

// Strategy Management

type Strategy struct {
	ConfigContent  string          `json:"config_content,omitempty"`
	Content        string          `json:"content,omitempty"`
	CreateTime     int64           `json:"create_time,omitempty"`
	ExcludeObjects *ExcludeObjects `json:"exclude_objects,omitempty"`
	Excludes       []string        `json:"excludes,omitempty"`
	GroupIDs       []int           `json:"group_ids,omitempty"`
	GroupInfos     []GroupInfo     `json:"group_infos,omitempty"`
	IncludesObject []IncludeObject `json:"includes_object,omitempty"`
	IsDefault      string          `json:"is_default,omitempty"`
	LastUpdateTime int64           `json:"last_update_time,omitempty"`
	Name           string          `json:"name,omitempty"`
	OperatorID     string          `json:"operator_id,omitempty"`
	OperatorName   string          `json:"operator_name,omitempty"`
	RangeType      int             `json:"range_type,omitempty"`
	Status         int             `json:"status,omitempty"`
	StrategyID     string          `json:"strategy_id,omitempty"`
	Type           string          `json:"type,omitempty"`
	VersionID      string          `json:"version_id,omitempty"`
}

type ExcludeObjects struct {
	ClientIDs  []string `json:"client_ids,omitempty"`
	Departs    []string `json:"departs,omitempty"`
	HostGroups []string `json:"host_groups,omitempty"`
	UserGroups []string `json:"user_groups,omitempty"`
	Users      []string `json:"users,omitempty"`
}

type GroupInfo struct {
	GID       int    `json:"gid"`
	GroupName string `json:"group_name"`
	GroupType string `json:"group_type"`
	Status    int    `json:"status"`
}

type IncludeObject struct {
	Object []string `json:"object,omitempty"`
	Type   string   `json:"type,omitempty"`
}

type ListStrategiesRequest struct {
	Content        string      `json:"content,omitempty"`
	CreateTime     *TimeFilter `json:"create_time,omitempty"`
	GroupIDs       []int       `json:"group_ids,omitempty"`
	Includes       []string    `json:"includes,omitempty"`
	LastUpdateTime *TimeFilter `json:"last_update_time,omitempty"`
	Limit          int         `json:"limit"`
	Name           string      `json:"name,omitempty"`
	Page           int         `json:"page"`
	RangeType      int         `json:"range_type,omitempty"`
	Status         int         `json:"status,omitempty"`
	StrategyID     string      `json:"strategy_id,omitempty"`
	Type           string      `json:"type"`
}

type ListStrategiesResponse struct {
	Items []Strategy `json:"items"`
	Total int        `json:"total"`
}

type GetStrategyDetailRequest struct {
	StrategyID string `json:"strategy_id"`
	Type       string `json:"type"`
}

type CreateStrategyRequest struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Content        string          `json:"content,omitempty"`
	RangeType      int             `json:"range_type"`
	GroupIDs       []int           `json:"group_ids,omitempty"`
	ConfigContent  string          `json:"config_content,omitempty"`
	Status         int             `json:"status,omitempty"`
	Includes       []string        `json:"includes,omitempty"`
	Excludes       []string        `json:"excludes,omitempty"`
	ExcludeObjects *ExcludeObjects `json:"exclude_objects,omitempty"`
}

type CreateStrategyResponse struct {
	StrategyID string `json:"strategy_id"`
}

type UpdateStrategyRequest struct {
	StrategyID     string          `json:"strategy_id"`
	Name           string          `json:"name,omitempty"`
	Type           string          `json:"type,omitempty"`
	Content        string          `json:"content,omitempty"`
	RangeType      int             `json:"range_type,omitempty"`
	GroupIDs       []int           `json:"group_ids,omitempty"`
	ConfigContent  string          `json:"config_content,omitempty"`
	Status         int             `json:"status,omitempty"`
	Includes       []string        `json:"includes,omitempty"`
	Excludes       []string        `json:"excludes,omitempty"`
	ExcludeObjects *ExcludeObjects `json:"exclude_objects,omitempty"`
}

type StrategyState struct {
	ActiveStrategy     int `json:"active_strategy"`
	AllStrategy        int `json:"all_strategy"`
	AlarmTerminalCount int `json:"alarm_terminal_count"`
	BanInternetAccess  int `json:"ban_internet_access"`
	DetectionPeriod    int `json:"detection_period"`
}

type UpdateStrategyStatusRequest struct {
	StrategyID string `json:"strategy_id,omitempty"`
	Type       string `json:"type,omitempty"`
	Status     int    `json:"status"`
}

type ListDetectionsRequest struct {
	Page     int
	PageSize int
}

type ListDetectionsResponse struct {
	Total      int         `json:"total"`
	Detections []Detection `json:"data"`
}

type ListDetectionsProxyRequest struct {
	Page            int         `json:"page"`
	Limit           int         `json:"limit"`
	From            string      `json:"from,omitempty"`
	ThreatLevel     string      `json:"threat_level,omitempty"`
	TAID            string      `json:"ta_id,omitempty"`
	TID             string      `json:"t_id,omitempty"`
	Hash            string      `json:"hash,omitempty"`
	PName           string      `json:"p_name,omitempty"`
	DetectTime      *TimeFilter `json:"detect_time,omitempty"`
	Hostname        string      `json:"hostname,omitempty"`
	ClientID        string      `json:"client_id,omitempty"`
	Username        string      `json:"username,omitempty"`
	DealStatus      string      `json:"deal_status,omitempty"`
	IncidentID      string      `json:"incident_id,omitempty"`
	DealStatusArray []int       `json:"deal_status_array,omitempty"`
	StartTime       *TimeFilter `json:"start_time,omitempty"`
	RootName        string      `json:"root_name,omitempty"`
	MainPName       string      `json:"main_p_name,omitempty"`
	MalwareName     string      `json:"malware_name,omitempty"`
	DetectionSource string      `json:"detection_source,omitempty"`
	ViewType        string      `json:"view_type,omitempty"`
	DetectionIDs    []string    `json:"detection_ids,omitempty"`
	RMConnectIP     string      `json:"rm_connect_ip,omitempty"`
	OrgConnectIP    string      `json:"org_connect_ip,omitempty"`
	ClientIP        string      `json:"client_ip,omitempty"`
}

type ListDetectionsProxyResponse struct {
	Total   int              `json:"total"`
	Results []map[string]any `json:"results"`
}

type UpdateDetectionStatusRequest struct {
	IDs        []string `json:"ids"`
	DealStatus int      `json:"deal_status"`
}

type ListEventLogAlarmsRequest struct {
	AlarmName   string      `json:"alarm_name,omitempty"`
	RiskLevel   string      `json:"risk_level,omitempty"`
	ClientID    string      `json:"client_id,omitempty"`
	HostTokenID string      `json:"host_token_id,omitempty"`
	Channel     string      `json:"channel,omitempty"`
	HostStatus  string      `json:"host_status,omitempty"`
	DateTime    *TimeFilter `json:"date_time,omitempty"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
	IsExport    int         `json:"is_export,omitempty"`
}

type EventLogAlarm struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	OrgName       string `json:"org_name"`
	RuleID        string `json:"rule_id"`
	Filter        any    `json:"filter"`
	LogNum        int    `json:"log_num"`
	RiskLevel     string `json:"risk_level"`
	ClientID      string `json:"client_id"`
	TokenID       string `json:"token_id"`
	CreateTime    int64  `json:"create_time"`
	CreateTimeUTC string `json:"create_time_utc"`
}

type ListEventLogAlarmsResponse struct {
	Total   int             `json:"total"`
	Results []EventLogAlarm `json:"results"`
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

type BatchDealIncidentRequest struct {
	IDs     []string `json:"ids"`
	Allow   bool     `json:"allow"`
	Status  int      `json:"status"`
	Scene   string   `json:"scene"`
	Comment string   `json:"comment,omitempty"`
}

type BatchDealIncidentResponse struct {
	IncidentName   string   `json:"incident_name"`
	Status         int      `json:"status"`
	TotalDetection int      `json:"total_detection"`
	TotalIncident  int      `json:"total_incident"`
	IncidentNames  []string `json:"incident_names"`
}

type IncidentR2SummaryRequest struct {
	IncidentID string `json:"incident_id"`
}

type IncidentR2SummaryResponse struct {
	ID              string   `json:"id"`
	ClientID        string   `json:"client_id"`
	IncidentID      string   `json:"incident_id"`
	IncidentName    string   `json:"incident_name"`
	Status          int      `json:"status"`
	Score           float64  `json:"score"`
	Comment         string   `json:"comment"`
	Remarks         string   `json:"remarks"`
	Tags            []string `json:"tags"`
	TTP             TTPInfo  `json:"ttp"`
	Release         int      `json:"release"`
	Actors          []string `json:"actors"`
	MultiHost       bool     `json:"multihost"`
	Scene           int      `json:"scene"`
	AssociatedHosts []string `json:"associated_hosts"`
	HostID          string   `json:"host_id"`
	HostName        string   `json:"host_name"`
	OperatingSystem string   `json:"operating_system"`
	Username        string   `json:"username"`
	ExternalIP      string   `json:"external_ip"`
	ConnectionIP    string   `json:"connection_ip"`
	ClientVersion   string   `json:"client_version"`
	Isolation       int      `json:"isolation"`
	HostStatus      string   `json:"host_status"`
	Actor           string   `json:"actor"`
	ActorType       string   `json:"actor_type"`
	TNames          []string `json:"t_names"`
	StartTime       int64    `json:"start_time"`
	EndTime         int64    `json:"end_time"`
	KeepAliveStatus int      `json:"keep_alive_status"`
	Platform        int      `json:"platform"`
}

type TTPInfo struct {
	Target    string `json:"target"`
	Technique string `json:"technique"`
	Course    string `json:"course"`
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

func (c *OpenAPIClient) AddHostBlacklist(ctx context.Context, clientIDs []string, reason string) error {
	payload := map[string]any{
		"client_ids": clientIDs,
		"reason":     reason,
	}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/hosts/add_blacklist", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("add host blacklist failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) RemoveHost(ctx context.Context, clientIDs []string) error {
	payload := map[string]any{
		"client_ids": clientIDs,
	}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/hosts/remove_host", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("remove host failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) IsolateHost(ctx context.Context, clientID string, time int) (InstructionResult, error) {
	req := SendInstructionRequest{
		ClientID:        clientID,
		InstructionName: "quarantine_network",
	}
	if time > 0 {
		req.Params = &Params{Time: time}
	}
	return c.sendInstruction(ctx, req)
}

func (c *OpenAPIClient) ReleaseHost(ctx context.Context, clientID string) (InstructionResult, error) {
	return c.sendInstruction(ctx, SendInstructionRequest{
		ClientID:        clientID,
		InstructionName: "recover_network",
	})
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

func (c *OpenAPIClient) ListEventLogAlarms(ctx context.Context, req ListEventLogAlarmsRequest) (ListEventLogAlarmsResponse, error) {
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
	if req.AlarmName != "" {
		payload["alarm_name"] = req.AlarmName
	}
	if req.RiskLevel != "" {
		payload["risk_level"] = req.RiskLevel
	}
	if req.ClientID != "" {
		payload["client_id"] = req.ClientID
	}
	if req.HostTokenID != "" {
		payload["host_token_id"] = req.HostTokenID
	}
	if req.Channel != "" {
		payload["channel"] = req.Channel
	}
	if req.HostStatus != "" {
		payload["host_status"] = req.HostStatus
	}
	if req.DateTime != nil {
		payload["date_time"] = req.DateTime
	}
	if req.IsExport > 0 {
		payload["is_export"] = req.IsExport
	}

	var envelope apiEnvelope[ListEventLogAlarmsResponse]
	if err := c.postPlatform(ctx, "/detection/alarms/events_log/alarm_list", payload, &envelope); err != nil {
		return ListEventLogAlarmsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListEventLogAlarmsResponse{}, fmt.Errorf("event log alarms list failed: %s", envelope.Message)
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
			"sort_key": "score",
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

func (c *OpenAPIClient) BatchDealIncident(ctx context.Context, req BatchDealIncidentRequest) (BatchDealIncidentResponse, error) {
	payload := map[string]any{
		"ids":    req.IDs,
		"allow":  req.Allow,
		"status": req.Status,
		"scene":  req.Scene,
	}
	if req.Comment != "" {
		payload["comment"] = req.Comment
	}
	var envelope apiEnvelope[BatchDealIncidentResponse]
	if err := c.post(ctx, "/incident/batch_deal", payload, &envelope); err != nil {
		return BatchDealIncidentResponse{}, err
	}
	if envelope.Error != 0 {
		return BatchDealIncidentResponse{}, fmt.Errorf("batch deal incident failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) IncidentR2Summary(ctx context.Context, incidentID string) (IncidentR2SummaryResponse, error) {
	payload := map[string]any{
		"incident_id": incidentID,
	}
	var envelope apiEnvelope[IncidentR2SummaryResponse]
	if err := c.post(ctx, "/incident/r2/summary", payload, &envelope); err != nil {
		return IncidentR2SummaryResponse{}, err
	}
	if envelope.Error != 0 {
		return IncidentR2SummaryResponse{}, fmt.Errorf("incident r2 summary failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) ListDetectionsProxy(ctx context.Context, req ListDetectionsProxyRequest) (ListDetectionsProxyResponse, error) {
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
	if req.From != "" {
		payload["from"] = req.From
	}
	if req.ThreatLevel != "" {
		payload["threat_level"] = req.ThreatLevel
	}
	if req.TAID != "" {
		payload["ta_id"] = req.TAID
	}
	if req.TID != "" {
		payload["t_id"] = req.TID
	}
	if req.Hash != "" {
		payload["hash"] = req.Hash
	}
	if req.PName != "" {
		payload["p_name"] = req.PName
	}
	if req.DetectTime != nil {
		payload["detect_time"] = req.DetectTime
	}
	if req.Hostname != "" {
		payload["hostname"] = req.Hostname
	}
	if req.ClientID != "" {
		payload["client_id"] = req.ClientID
	}
	if req.Username != "" {
		payload["username"] = req.Username
	}
	if req.DealStatus != "" {
		payload["deal_status"] = req.DealStatus
	}
	if req.IncidentID != "" {
		payload["incident_id"] = req.IncidentID
	}
	if len(req.DealStatusArray) > 0 {
		payload["deal_status_array"] = req.DealStatusArray
	}
	if req.StartTime != nil {
		payload["start_time"] = req.StartTime
	}
	if req.RootName != "" {
		payload["root_name"] = req.RootName
	}
	if req.MainPName != "" {
		payload["main_p_name"] = req.MainPName
	}
	if req.MalwareName != "" {
		payload["malware_name"] = req.MalwareName
	}
	if req.DetectionSource != "" {
		payload["detection_source"] = req.DetectionSource
	}
	if req.ViewType != "" {
		payload["view_type"] = req.ViewType
	}
	if len(req.DetectionIDs) > 0 {
		payload["detection_ids"] = req.DetectionIDs
	}
	if req.RMConnectIP != "" {
		payload["rm_connect_ip"] = req.RMConnectIP
	}
	if req.OrgConnectIP != "" {
		payload["org_connect_ip"] = req.OrgConnectIP
	}
	if req.ClientIP != "" {
		payload["client_ip"] = req.ClientIP
	}

	var envelope apiEnvelope[ListDetectionsProxyResponse]
	if err := c.post(ctx, "/detection/get_list", payload, &envelope); err != nil {
		return ListDetectionsProxyResponse{}, err
	}
	if envelope.Error != 0 {
		return ListDetectionsProxyResponse{}, fmt.Errorf("detection proxy list failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) UpdateDetectionStatus(ctx context.Context, req UpdateDetectionStatusRequest) error {
	payload := map[string]any{
		"ids":         req.IDs,
		"deal_status": req.DealStatus,
	}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/detection/deal_status", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("update detection status failed: %s", envelope.Message)
	}
	return nil
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

func (c *OpenAPIClient) SendInstruction(ctx context.Context, req SendInstructionRequest) (InstructionResult, error) {
	return c.sendInstruction(ctx, req)
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

func (c *OpenAPIClient) sendInstruction(ctx context.Context, req SendInstructionRequest) (InstructionResult, error) {
	payload := map[string]any{
		"client_id":        req.ClientID,
		"instruction_name": req.InstructionName,
	}
	if req.IsOnline != 0 {
		payload["is_online"] = req.IsOnline
	}
	if req.IsBatch != 0 {
		payload["is_batch"] = req.IsBatch
	}
	if len(req.BatchParams) > 0 {
		batchParams := make([]map[string]any, len(req.BatchParams))
		for i, bp := range req.BatchParams {
			m := map[string]any{}
			if bp.ID != "" {
				m["id"] = bp.ID
			}
			if bp.Path != "" {
				m["path"] = bp.Path
			}
			if bp.SHA1 != "" {
				m["sha1"] = bp.SHA1
			}
			if bp.Pid != 0 {
				m["pid"] = bp.Pid
			}
			batchParams[i] = m
		}
		payload["batch_params"] = batchParams
	}
	if req.Params != nil {
		params := map[string]any{}
		if req.Params.Time != 0 {
			params["time"] = req.Params.Time
		}
		if len(params) > 0 {
			payload["params"] = params
		}
	}

	jsonBytes, _ := json.MarshalIndent(payload, "", "  ")
	log.Printf("send_instruction request body:\n%s", string(jsonBytes))

	// First parse as apiEnvelope[any] to get error info before unmarshaling data
	var envelopeRaw apiEnvelope[any]
	if err := c.post(ctx, "/instructions/send_instruction", payload, &envelopeRaw); err != nil {
		return InstructionResult{}, err
	}
	if envelopeRaw.Error != 0 {
		return InstructionResult{}, fmt.Errorf("edr send instruction failed: error=%d %s", envelopeRaw.Error, envelopeRaw.Message)
	}

	// Now unmarshal data into InstructionResult
	var envelope apiEnvelope[InstructionResult]
	if err := c.post(ctx, "/instructions/send_instruction", payload, &envelope); err != nil {
		return InstructionResult{}, err
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

func (c *OpenAPIClient) put(ctx context.Context, path string, payload any, out any) error {
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
	return c.putWithHeaders(ctx, c.baseURL+path, headers, payload, out)
}

func (c *OpenAPIClient) putWithHeaders(ctx context.Context, url string, headers map[string]string, payload any, out any) error {
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal edr payload: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
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
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("edr http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	rawBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(rawBody, out); err != nil {
		return fmt.Errorf("unmarshal edr response: %w", err)
	}
	return nil
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

	rawBody, _ := io.ReadAll(resp.Body)
	// fmt.Printf("===edr raw response: %s\n", string(rawBody))
	if err := json.NewDecoder(bytes.NewReader(rawBody)).Decode(out); err != nil {
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

// Virus Statistics

func (c *OpenAPIClient) ListVirusByHost(ctx context.Context, req ListVirusByHostRequest) (ListVirusByHostResponse, error) {
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
	if req.Username != "" {
		payload["username"] = req.Username
	}
	if req.HostName != "" {
		payload["host_name"] = req.HostName
	}
	if req.Importance > 0 {
		payload["importance"] = req.Importance
	}
	if req.MacAddress != "" {
		payload["mac_address"] = req.MacAddress
	}
	if req.ClientIP != "" {
		payload["client_ip"] = req.ClientIP
	}
	if req.RMConnectIP != "" {
		payload["rmconnectip"] = req.RMConnectIP
	}
	if req.Status > 0 {
		payload["status"] = req.Status
	}
	if req.LastCheckedTime != nil {
		payload["last_checked_time"] = req.LastCheckedTime
	}

	var envelope apiEnvelope[ListVirusByHostResponse]
	if err := c.post(ctx, "/virus/host/list", payload, &envelope); err != nil {
		return ListVirusByHostResponse{}, err
	}
	if envelope.Error != 0 {
		return ListVirusByHostResponse{}, fmt.Errorf("list virus by host failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) ListVirusByHash(ctx context.Context, req ListVirusByHashRequest) (ListVirusByHashResponse, error) {
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
	if req.LastCheckedTime != nil {
		payload["last_checked_time"] = req.LastCheckedTime
	}
	if req.Name != "" {
		payload["name"] = req.Name
	}
	if req.SHA1 != "" {
		payload["sha1"] = req.SHA1
	}
	if req.MD5 != "" {
		payload["md5"] = req.MD5
	}

	var envelope apiEnvelope[ListVirusByHashResponse]
	if err := c.post(ctx, "/virus/hash/list", payload, &envelope); err != nil {
		return ListVirusByHashResponse{}, err
	}
	if envelope.Error != 0 {
		return ListVirusByHashResponse{}, fmt.Errorf("list virus by hash failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) ListVirusHashHosts(ctx context.Context, req ListVirusHashHostsRequest) (ListVirusHashHostsResponse, error) {
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
	if req.SHA1 != "" {
		payload["sha1"] = req.SHA1
	}
	if req.ClientID != "" {
		payload["client_id"] = req.ClientID
	}
	if req.Username != "" {
		payload["username"] = req.Username
	}
	if req.HostName != "" {
		payload["host_name"] = req.HostName
	}
	if req.Importance > 0 {
		payload["importance"] = req.Importance
	}
	if req.MacAddress != "" {
		payload["mac_address"] = req.MacAddress
	}
	if req.ClientIP != "" {
		payload["client_ip"] = req.ClientIP
	}
	if req.RMConnectIP != "" {
		payload["rmconnectip"] = req.RMConnectIP
	}
	if req.Status > 0 {
		payload["status"] = req.Status
	}
	if req.HostStatus != "" {
		payload["host_status"] = req.HostStatus
	}
	if req.Path != "" {
		payload["path"] = req.Path
	}
	if req.LastCheckedTime != nil {
		payload["last_checked_time"] = req.LastCheckedTime
	}

	var envelope apiEnvelope[ListVirusHashHostsResponse]
	if err := c.post(ctx, "/virus/hash/host/list", payload, &envelope); err != nil {
		return ListVirusHashHostsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListVirusHashHostsResponse{}, fmt.Errorf("list virus hash hosts failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) ListVirusScanRecords(ctx context.Context, req ListVirusScanRecordsRequest) (ListVirusScanRecordsResponse, error) {
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
	if req.RID != "" {
		payload["rid"] = req.RID
	}
	if req.TaskID != "" {
		payload["task_id"] = req.TaskID
	}
	if req.ExecutionBatch != "" {
		payload["execution_batch"] = req.ExecutionBatch
	}
	if req.HostName != "" {
		payload["host_name"] = req.HostName
	}
	if req.ClientID != "" {
		payload["client_id"] = req.ClientID
	}
	if req.ScanType != "" {
		payload["scan_type"] = req.ScanType
	}
	if req.Status != "" {
		payload["status"] = req.Status
	}
	if req.StartTime != nil {
		payload["start_time"] = req.StartTime
	}
	if req.EndTime != nil {
		payload["end_time"] = req.EndTime
	}

	var envelope apiEnvelope[ListVirusScanRecordsResponse]
	if err := c.post(ctx, "/virus_scan/scan_record", payload, &envelope); err != nil {
		return ListVirusScanRecordsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListVirusScanRecordsResponse{}, fmt.Errorf("list virus scan records failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

// Plan Management

func (c *OpenAPIClient) AddPlan(ctx context.Context, req AddPlanRequest) error {
	payload := map[string]any{
		"scan_type": req.ScanType,
		"plan_type": req.PlanType,
		"scope":     req.Scope,
		"type":      req.Type,
	}
	if req.RID != "" {
		payload["rid"] = req.RID
	}
	if req.PlanName != "" {
		payload["plan_name"] = req.PlanName
	}
	if req.Contents != nil {
		payload["contents"] = req.Contents
	}
	if req.ExecuteStartTime > 0 {
		payload["execute_start_time"] = req.ExecuteStartTime
	}
	if req.ExecuteCycle > 0 {
		payload["execute_cycle"] = req.ExecuteCycle
	}
	if len(req.RepeatCycle) > 0 {
		payload["repeat_cycle"] = req.RepeatCycle
	}
	if req.ExecutionTime != "" {
		payload["execution_time"] = req.ExecutionTime
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if len(req.DeviceClientIDs) > 0 {
		payload["device_client_ids"] = req.DeviceClientIDs
	}
	if req.ExpiredSetting > 0 {
		payload["expired_setting"] = req.ExpiredSetting
	}
	if req.ExpiredTime > 0 {
		payload["expired_time"] = req.ExpiredTime
	}
	if len(req.SearchContent) > 0 {
		payload["search_content"] = req.SearchContent
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/plan/add", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("add plan failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) EditPlan(ctx context.Context, req EditPlanRequest) error {
	payload := map[string]any{}
	if req.RID != "" {
		payload["rid"] = req.RID
	}
	if req.ClientID != "" {
		payload["client_id"] = req.ClientID
	}
	if req.ScanType > 0 {
		payload["scan_type"] = req.ScanType
	}
	if req.PlanName != "" {
		payload["plan_name"] = req.PlanName
	}
	if req.PlanType > 0 {
		payload["plan_type"] = req.PlanType
	}
	if req.Scope > 0 {
		payload["scope"] = req.Scope
	}
	if req.Contents != nil {
		payload["contents"] = req.Contents
	}
	if req.ExecuteStartTime > 0 {
		payload["execute_start_time"] = req.ExecuteStartTime
	}
	if req.ExecuteCycle > 0 {
		payload["execute_cycle"] = req.ExecuteCycle
	}
	if len(req.RepeatCycle) > 0 {
		payload["repeat_cycle"] = req.RepeatCycle
	}
	if req.ExecutionTime != "" {
		payload["execution_time"] = req.ExecutionTime
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.Type != "" {
		payload["type"] = req.Type
	}
	if len(req.DeviceClientIDs) > 0 {
		payload["device_client_ids"] = req.DeviceClientIDs
	}
	if req.ExpiredSetting > 0 {
		payload["expired_setting"] = req.ExpiredSetting
	}
	if req.ExpiredTime > 0 {
		payload["expired_time"] = req.ExpiredTime
	}
	if len(req.SearchContent) > 0 {
		payload["search_content"] = req.SearchContent
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/plan/edit", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("edit plan failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) CancelPlan(ctx context.Context, rid string) error {
	var envelope apiEnvelope[any]
	if err := c.put(ctx, "/plan/cancel/"+rid, nil, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("cancel plan failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) ListPlans(ctx context.Context, req ListPlansRequest) (ListPlansResponse, error) {
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
	if req.Type != "" {
		payload["type"] = req.Type
	}
	if req.ScanType > 0 {
		payload["scan_type"] = req.ScanType
	}
	if req.PlanType > 0 {
		payload["plan_type"] = req.PlanType
	}
	if req.SearchContent != "" {
		payload["search_content"] = req.SearchContent
	}

	var envelope apiEnvelope[ListPlansResponse]
	if err := c.post(ctx, "/plan/list", payload, &envelope); err != nil {
		return ListPlansResponse{}, err
	}
	if envelope.Error != 0 {
		return ListPlansResponse{}, fmt.Errorf("list plans failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

// Instruction Policy (Auto Response)

func (c *OpenAPIClient) ListInstructionPolicies(ctx context.Context, req ListInstructionPoliciesRequest) (ListInstructionPoliciesResponse, error) {
	payload := map[string]any{}
	if req.PolicyType > 0 {
		payload["policy_type"] = req.PolicyType
	}
	if req.Name != "" {
		payload["name"] = req.Name
	}
	if req.OperationUser != "" {
		payload["operation_user"] = req.OperationUser
	}
	if req.Scopes != "" {
		payload["scopes"] = req.Scopes
	}
	if req.Action > 0 {
		payload["action"] = req.Action
	}
	if req.Status > 0 {
		payload["status"] = req.Status
	}

	var envelope apiEnvelope[ListInstructionPoliciesResponse]
	if err := c.post(ctx, "/instruction_policy/list", payload, &envelope); err != nil {
		return ListInstructionPoliciesResponse{}, err
	}
	if envelope.Error != 0 {
		return ListInstructionPoliciesResponse{}, fmt.Errorf("list instruction policies failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) UpdateInstructionPolicy(ctx context.Context, req UpdateInstructionPolicyRequest) error {
	payload := map[string]any{
		"rid":             req.RID,
		"name":            req.Name,
		"condition_list":  req.ConditionList,
		"action":          req.Action,
		"scope":           req.Scope,
		"client_id":       req.ClientID,
		"group_ids":       req.GroupIDs,
		"policy_type":     req.PolicyType,
		"tq_group":        req.TQGroup,
		"scope_content":   req.ScopeContent,
		"operation_user":  req.OperationUser,
		"create_time":     req.CreateTime,
		"update_time":     req.UpdateTime,
		"status":          req.Status,
		"task_num":        req.TaskNum,
		"task_start_time": req.TaskStartTime,
		"task_end_time":   req.TaskEndTime,
		"index":           req.Index,
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/instruction_policy/update", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("update instruction policy failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) SaveInstructionPolicyStatus(ctx context.Context, req SaveInstructionPolicyStatusRequest) (SaveInstructionPolicyStatusResponse, error) {
	payload := map[string]any{
		"rid":  req.RID,
		"rids": req.RIDs,
	}

	var envelope apiEnvelope[SaveInstructionPolicyStatusResponse]
	if err := c.post(ctx, "/instruction_policy/save_status", payload, &envelope); err != nil {
		return SaveInstructionPolicyStatusResponse{}, err
	}
	if envelope.Error != 0 {
		return SaveInstructionPolicyStatusResponse{}, fmt.Errorf("save instruction policy status failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) DeleteInstructionPolicy(ctx context.Context, rid string) (DeleteInstructionPolicyResponse, error) {
	payload := map[string]any{
		"rid": rid,
	}

	var envelope apiEnvelope[DeleteInstructionPolicyResponse]
	if err := c.post(ctx, "/instruction_policy/delete", payload, &envelope); err != nil {
		return DeleteInstructionPolicyResponse{}, err
	}
	if envelope.Error != 0 {
		return DeleteInstructionPolicyResponse{}, fmt.Errorf("delete instruction policy failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) SortInstructionPolicies(ctx context.Context, rids []string) error {
	payload := map[string]any{
		"rids": rids,
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/instruction_policy/save_sort", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("sort instruction policies failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) AddInstructionPolicy(ctx context.Context, req AddInstructionPolicyRequest) (AddInstructionPolicyResponse, error) {
	payload := map[string]any{
		"name":            req.Name,
		"condition_list":  req.ConditionList,
		"action":          req.Action,
		"scope":           req.Scope,
		"client_id":       req.ClientID,
		"group_ids":       req.GroupIDs,
		"policy_type":     req.PolicyType,
		"tq_group":        req.TQGroup,
		"scope_content":   req.ScopeContent,
		"operation_user":  req.OperationUser,
		"status":          req.Status,
		"task_num":        req.TaskNum,
		"task_start_time": req.TaskStartTime,
		"task_end_time":   req.TaskEndTime,
		"index":           req.Index,
	}

	var envelope apiEnvelope[AddInstructionPolicyResponse]
	if err := c.post(ctx, "/instruction_policy/add_policy", payload, &envelope); err != nil {
		return AddInstructionPolicyResponse{}, err
	}
	if envelope.Error != 0 {
		return AddInstructionPolicyResponse{}, fmt.Errorf("add instruction policy failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

// Client Setting (Host Offline)

func (c *OpenAPIClient) GetHostOfflineConf(ctx context.Context) (HostOfflineConf, error) {
	var envelope apiEnvelope[HostOfflineConf]
	if err := c.get(ctx, "/client_setting/host_offline", nil, &envelope); err != nil {
		return HostOfflineConf{}, err
	}
	if envelope.Error != 0 {
		return HostOfflineConf{}, fmt.Errorf("get host offline conf failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) SaveHostOfflineConf(ctx context.Context, req SaveHostOfflineConfRequest) error {
	payload := map[string]any{
		"status":  req.Status,
		"setting": req.Setting,
	}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/client_setting/host_offline", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("save host offline conf failed: %s", envelope.Message)
	}
	return nil
}

// IOA Configuration

func (c *OpenAPIClient) ListIOAs(ctx context.Context, req ListIOAsRequest) (ListIOAsResponse, error) {
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
	if req.CommandLine != "" {
		payload["command_line"] = req.CommandLine
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
	if req.IOAName != "" {
		payload["ioa_name"] = req.IOAName
	}
	if req.LastModified != nil {
		payload["last_modified"] = req.LastModified
	}
	if req.ModifiedBy != "" {
		payload["modified_by"] = req.ModifiedBy
	}
	if req.Name != "" {
		payload["name"] = req.Name
	}
	if req.TAID != "" {
		payload["ta_id"] = req.TAID
	}
	if req.TID != "" {
		payload["t_id"] = req.TID
	}

	var envelope apiEnvelope[ListIOAsResponse]
	if err := c.post(ctx, "/configure/ioa/list", payload, &envelope); err != nil {
		return ListIOAsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListIOAsResponse{}, fmt.Errorf("list ioas failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) AddIOA(ctx context.Context, req AddIOARequest) error {
	payload := map[string]any{}
	if req.CommandLine != "" {
		payload["command_line"] = req.CommandLine
	}
	if req.Description != "" {
		payload["description"] = req.Description
	}
	if req.ExclusionName != "" {
		payload["exclusion_name"] = req.ExclusionName
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
	if req.IOAID != "" {
		payload["ioa_id"] = req.IOAID
	}
	if req.Severity != "" {
		payload["severity"] = req.Severity
	}
	if req.TAID != "" {
		payload["ta_id"] = req.TAID
	}
	if req.TID != "" {
		payload["t_id"] = req.TID
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioa/add", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("add ioa failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) UpdateIOA(ctx context.Context, req UpdateIOARequest) error {
	payload := map[string]any{
		"id": req.ID,
	}
	if req.CommandLine != "" {
		payload["command_line"] = req.CommandLine
	}
	if req.Description != "" {
		payload["description"] = req.Description
	}
	if req.ExclusionName != "" {
		payload["exclusion_name"] = req.ExclusionName
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
	if req.TAID != "" {
		payload["ta_id"] = req.TAID
	}
	if req.TID != "" {
		payload["t_id"] = req.TID
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioa/update", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("update ioa failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) DeleteIOA(ctx context.Context, id string) error {
	payload := map[string]any{"id": id}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioa/delete", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("delete ioa failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) ListIOAAuditLogs(ctx context.Context, req ListIOAAuditLogsRequest) (ListIOAAuditLogsResponse, error) {
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
	if req.CommandLine != "" {
		payload["command_line"] = req.CommandLine
	}
	if req.EventTime != nil {
		payload["event_time"] = req.EventTime
	}
	if req.FileName != "" {
		payload["file_name"] = req.FileName
	}
	if req.HostName != "" {
		payload["host_name"] = req.HostName
	}
	if req.IOAName != "" {
		payload["ioa_name"] = req.IOAName
	}

	var envelope apiEnvelope[ListIOAAuditLogsResponse]
	if err := c.post(ctx, "/configure/ioa/audit_log", payload, &envelope); err != nil {
		return ListIOAAuditLogsResponse{}, err
	}
	if envelope.Error != 0 {
		return ListIOAAuditLogsResponse{}, fmt.Errorf("list ioa audit logs failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

// IOA Network Exclusion

func (c *OpenAPIClient) ListIOANetworks(ctx context.Context, req ListIOANetworksRequest) (ListIOANetworksResponse, error) {
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
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.HostType != "" {
		payload["host_type"] = req.HostType
	}
	if req.IP != "" {
		payload["ip"] = req.IP
	}
	if req.LastModified != nil {
		payload["last_modified"] = req.LastModified
	}
	if req.ModifiedBy != "" {
		payload["modified_by"] = req.ModifiedBy
	}
	if req.Name != "" {
		payload["name"] = req.Name
	}

	var envelope apiEnvelope[ListIOANetworksResponse]
	if err := c.post(ctx, "/configure/ioa_network/list", payload, &envelope); err != nil {
		return ListIOANetworksResponse{}, err
	}
	if envelope.Error != 0 {
		return ListIOANetworksResponse{}, fmt.Errorf("list ioa networks failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) AddIOANetwork(ctx context.Context, req AddIOANetworkRequest) error {
	payload := map[string]any{
		"exclusion_name": req.ExclusionName,
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.HostType != "" {
		payload["host_type"] = req.HostType
	}
	if req.IP != "" {
		payload["ip"] = req.IP
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioa_network/add", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("add ioa network failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) UpdateIOANetwork(ctx context.Context, req UpdateIOANetworkRequest) error {
	payload := map[string]any{
		"id": req.ID,
	}
	if req.ExclusionName != "" {
		payload["exclusion_name"] = req.ExclusionName
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.HostType != "" {
		payload["host_type"] = req.HostType
	}
	if req.IP != "" {
		payload["ip"] = req.IP
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioa_network/update", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("update ioa network failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) DeleteIOANetwork(ctx context.Context, id string) error {
	payload := map[string]any{"id": id}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/configure/ioa_network/delete", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("delete ioa network failed: %s", envelope.Message)
	}
	return nil
}

// Strategy Management

func (c *OpenAPIClient) GetStrategySingle(ctx context.Context, strategyType string) (Strategy, error) {
	var envelope apiEnvelope[Strategy]
	if err := c.get(ctx, fmt.Sprintf("/strategy/%s/single", strategyType), nil, &envelope); err != nil {
		return Strategy{}, err
	}
	if envelope.Error != 0 {
		return Strategy{}, fmt.Errorf("get strategy single failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) ListStrategies(ctx context.Context, req ListStrategiesRequest) (ListStrategiesResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = c.cfg.DefaultPageSize
	}
	payload := map[string]any{
		"page":  req.Page,
		"limit": req.Limit,
		"type":  req.Type,
	}
	if req.Content != "" {
		payload["content"] = req.Content
	}
	if req.CreateTime != nil {
		payload["create_time"] = req.CreateTime
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if len(req.Includes) > 0 {
		payload["includes"] = req.Includes
	}
	if req.LastUpdateTime != nil {
		payload["last_update_time"] = req.LastUpdateTime
	}
	if req.Name != "" {
		payload["name"] = req.Name
	}
	if req.RangeType > 0 {
		payload["range_type"] = req.RangeType
	}
	if req.Status > 0 {
		payload["status"] = req.Status
	}
	if req.StrategyID != "" {
		payload["strategy_id"] = req.StrategyID
	}

	var envelope apiEnvelope[ListStrategiesResponse]
	if err := c.post(ctx, "/strategy/list", payload, &envelope); err != nil {
		return ListStrategiesResponse{}, err
	}
	if envelope.Error != 0 {
		return ListStrategiesResponse{}, fmt.Errorf("list strategies failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) GetStrategyDetail(ctx context.Context, req GetStrategyDetailRequest) (Strategy, error) {
	payload := map[string]any{
		"strategy_id": req.StrategyID,
		"type":        req.Type,
	}
	var envelope apiEnvelope[Strategy]
	if err := c.post(ctx, "/strategy/detail", payload, &envelope); err != nil {
		return Strategy{}, err
	}
	if envelope.Error != 0 {
		return Strategy{}, fmt.Errorf("get strategy detail failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) CreateStrategy(ctx context.Context, req CreateStrategyRequest) (CreateStrategyResponse, error) {
	payload := map[string]any{
		"name":       req.Name,
		"type":       req.Type,
		"range_type": req.RangeType,
	}
	if req.Content != "" {
		payload["content"] = req.Content
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.ConfigContent != "" {
		payload["config_content"] = req.ConfigContent
	}
	if req.Status > 0 {
		payload["status"] = req.Status
	}
	if len(req.Includes) > 0 {
		payload["includes"] = req.Includes
	}
	if len(req.Excludes) > 0 {
		payload["excludes"] = req.Excludes
	}
	if req.ExcludeObjects != nil {
		payload["exclude_objects"] = req.ExcludeObjects
	}

	var envelope apiEnvelope[CreateStrategyResponse]
	if err := c.post(ctx, "/strategy/create", payload, &envelope); err != nil {
		return CreateStrategyResponse{}, err
	}
	if envelope.Error != 0 {
		return CreateStrategyResponse{}, fmt.Errorf("create strategy failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) UpdateStrategy(ctx context.Context, req UpdateStrategyRequest) error {
	payload := map[string]any{
		"strategy_id": req.StrategyID,
	}
	if req.Name != "" {
		payload["name"] = req.Name
	}
	if req.Type != "" {
		payload["type"] = req.Type
	}
	if req.Content != "" {
		payload["content"] = req.Content
	}
	if req.RangeType > 0 {
		payload["range_type"] = req.RangeType
	}
	if len(req.GroupIDs) > 0 {
		payload["group_ids"] = req.GroupIDs
	}
	if req.ConfigContent != "" {
		payload["config_content"] = req.ConfigContent
	}
	if req.Status > 0 {
		payload["status"] = req.Status
	}
	if len(req.Includes) > 0 {
		payload["includes"] = req.Includes
	}
	if len(req.Excludes) > 0 {
		payload["excludes"] = req.Excludes
	}
	if req.ExcludeObjects != nil {
		payload["exclude_objects"] = req.ExcludeObjects
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/strategy/update", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("update strategy failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) DeleteStrategy(ctx context.Context, strategyID string, strategyType string) error {
	payload := map[string]any{
		"strategy_id": strategyID,
		"type":        strategyType,
	}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/strategy/delete", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("delete strategy failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) GetStrategyState(ctx context.Context) (StrategyState, error) {
	var envelope apiEnvelope[StrategyState]
	if err := c.get(ctx, "/strategy/state", nil, &envelope); err != nil {
		return StrategyState{}, err
	}
	if envelope.Error != 0 {
		return StrategyState{}, fmt.Errorf("get strategy state failed: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func (c *OpenAPIClient) SortStrategies(ctx context.Context, sortIDs []string, strategyType string) error {
	payload := map[string]any{
		"sort_ids": sortIDs,
		"type":     strategyType,
	}
	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/strategy/sort", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("sort strategies failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) UpdateStrategyStatus(ctx context.Context, req UpdateStrategyStatusRequest) error {
	payload := map[string]any{
		"status": req.Status,
	}
	if req.StrategyID != "" {
		payload["strategy_id"] = req.StrategyID
	}
	if req.Type != "" {
		payload["type"] = req.Type
	}

	var envelope apiEnvelope[any]
	if err := c.post(ctx, "/strategy/status", payload, &envelope); err != nil {
		return err
	}
	if envelope.Error != 0 {
		return fmt.Errorf("update strategy status failed: %s", envelope.Message)
	}
	return nil
}

func (c *OpenAPIClient) get(ctx context.Context, path string, payload any, out any) error {
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
	return c.getWithHeaders(ctx, c.baseURL+path, headers, payload, out)
}

func (c *OpenAPIClient) getWithHeaders(ctx context.Context, url string, headers map[string]string, payload any, out any) error {
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal edr payload: %w", err)
		}
		url += "?" + string(body)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
