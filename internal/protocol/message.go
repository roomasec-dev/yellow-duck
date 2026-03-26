package protocol

import "time"

type Channel string

const (
	ChannelFeishu  Channel = "feishu"
	ChannelDingtalk Channel = "dingtalk"
)

type InboundMessage struct {
	Channel    Channel
	TenantKey  string
	ChatID     string
	ChatType   string
	ThreadID   string
	MessageID  string
	SenderID   string
	Text       string
	RawJSON    string
	ReceivedAt time.Time
}

type SessionRef struct {
	Key       string
	ScopeKey  string
	PublicID  string
	Title     string
	Status    string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Turn struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

type MemoryEntry struct {
	SessionKey string
	Key        string
	Value      string
	UpdatedAt  time.Time
}

type PendingAction struct {
	SessionKey string
	ActionID   string
	ActionType string
	Payload    string
	Summary    string
	CreatedAt  time.Time
}

type Artifact struct {
	SessionKey string
	ArtifactID string
	Kind       string
	Title      string
	Content    string
	CreatedAt  time.Time
}

type ArtifactMatch struct {
	Line    int
	Snippet string
}

type ScheduledTask struct {
	TaskID          string    `json:"task_id"`
	ScopeKey        string    `json:"scope_key"`
	SessionKey      string    `json:"session_key"`
	Channel         Channel   `json:"channel"`
	TenantKey       string    `json:"tenant_key"`
	ChatID          string    `json:"chat_id"`
	ThreadID        string    `json:"thread_id"`
	CreatorID       string    `json:"creator_id"`
	Title           string    `json:"title"`
	Prompt          string    `json:"prompt"`
	IntervalSeconds int       `json:"interval_seconds"`
	Status          string    `json:"status"`
	LastSummary     string    `json:"last_summary"`
	NextRunAt       time.Time `json:"next_run_at"`
	LastRunAt       time.Time `json:"last_run_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ScheduledTaskPatch struct {
	Title           string
	Prompt          string
	IntervalSeconds int
	Status          string
	LastSummary     string
	NextRunAt       time.Time
	LastRunAt       time.Time
}

type ScheduledTaskRun struct {
	RunID        string    `json:"run_id"`
	TaskID       string    `json:"task_id"`
	ScopeKey     string    `json:"scope_key"`
	SessionKey   string    `json:"session_key"`
	Status       string    `json:"status"`
	Summary      string    `json:"summary"`
	Report       string    `json:"report"`
	EntitiesJSON string    `json:"entities_json"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
}

type ScheduledTaskEntity struct {
	TaskID         string    `json:"task_id,omitempty"`
	EntityKey      string    `json:"entity_key,omitempty"`
	Kind           string    `json:"kind,omitempty"`
	EntityID       string    `json:"entity_id,omitempty"`
	Title          string    `json:"title,omitempty"`
	HostName       string    `json:"host_name,omitempty"`
	ClientID       string    `json:"client_id,omitempty"`
	Severity       string    `json:"severity,omitempty"`
	Status         string    `json:"status,omitempty"`
	Note           string    `json:"note,omitempty"`
	LastSummary    string    `json:"last_summary,omitempty"`
	FirstSeenAt    time.Time `json:"first_seen_at,omitempty"`
	LastSeenAt     time.Time `json:"last_seen_at,omitempty"`
	LastReportedAt time.Time `json:"last_reported_at,omitempty"`
}

type ScheduledTaskExecution struct {
	Summary  string                `json:"summary"`
	Message  string                `json:"message"`
	Run      ScheduledTaskRun      `json:"run"`
	Entities []ScheduledTaskEntity `json:"entities"`
}
