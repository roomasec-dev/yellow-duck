package store

import (
	"context"
	"time"

	"rm_ai_agent/internal/protocol"
)

type Store interface {
	RecordInboundMessage(ctx context.Context, msg protocol.InboundMessage) (bool, error)
	EnsureSession(ctx context.Context, sessionKey string) (protocol.SessionRef, error)
	EnsureActiveSession(ctx context.Context, scopeKey string) (protocol.SessionRef, error)
	CreateSession(ctx context.Context, scopeKey string, title string) (protocol.SessionRef, error)
	ListSessions(ctx context.Context, scopeKey string, limit int) ([]protocol.SessionRef, error)
	SetActiveSession(ctx context.Context, scopeKey string, publicID string) (protocol.SessionRef, error)
	CloseActiveSession(ctx context.Context, scopeKey string) (protocol.SessionRef, error)
	DeleteSession(ctx context.Context, scopeKey string, publicID string) error
	AppendTurn(ctx context.Context, sessionKey string, role string, content string) error
	ListRecentTurns(ctx context.Context, sessionKey string, limit int) ([]protocol.Turn, error)
	ListTurns(ctx context.Context, sessionKey string, limit int) ([]protocol.Turn, error)
	CountTurns(ctx context.Context, sessionKey string) (int, error)
	GetSessionSummary(ctx context.Context, sessionKey string) (string, error)
	UpsertSessionSummary(ctx context.Context, sessionKey string, summary string) error
	ListMemories(ctx context.Context, sessionKey string, limit int) ([]protocol.MemoryEntry, error)
	UpsertMemory(ctx context.Context, sessionKey string, key string, value string) error
	DeleteMemory(ctx context.Context, sessionKey string, key string) error
	CountMemories(ctx context.Context, sessionKey string) (int, error)
	SavePendingAction(ctx context.Context, sessionKey string, actionType string, payload string, summary string) error
	GetPendingAction(ctx context.Context, sessionKey string) (protocol.PendingAction, error)
	DeletePendingAction(ctx context.Context, sessionKey string) error
	SaveArtifact(ctx context.Context, sessionKey string, kind string, title string, content string) (protocol.Artifact, error)
	GetLatestArtifact(ctx context.Context, sessionKey string) (protocol.Artifact, error)
	GetArtifact(ctx context.Context, sessionKey string, artifactID string) (protocol.Artifact, error)
	CreateScheduledTask(ctx context.Context, task protocol.ScheduledTask) (protocol.ScheduledTask, error)
	ListScheduledTasks(ctx context.Context, limit int) ([]protocol.ScheduledTask, error)
	GetScheduledTask(ctx context.Context, scopeKey string, taskID string) (protocol.ScheduledTask, error)
	GetScheduledTaskByID(ctx context.Context, taskID string) (protocol.ScheduledTask, error)
	UpdateScheduledTask(ctx context.Context, scopeKey string, taskID string, patch protocol.ScheduledTaskPatch) (protocol.ScheduledTask, error)
	DeleteScheduledTask(ctx context.Context, taskID string) error
	ListDueScheduledTasks(ctx context.Context, now time.Time, limit int) ([]protocol.ScheduledTask, error)
	MarkScheduledTaskRunning(ctx context.Context, taskID string, lastRunAt time.Time, nextRunAt time.Time) error
	SaveScheduledTaskRun(ctx context.Context, run protocol.ScheduledTaskRun) error
	GetLatestScheduledTaskRun(ctx context.Context, scopeKey string) (protocol.ScheduledTaskRun, error)
	GetScheduledTaskState(ctx context.Context, taskID string) (string, error)
	UpsertScheduledTaskState(ctx context.Context, taskID string, state string) error
	ListScheduledTaskEntities(ctx context.Context, taskID string, limit int) ([]protocol.ScheduledTaskEntity, error)
	UpsertScheduledTaskEntity(ctx context.Context, entity protocol.ScheduledTaskEntity) error
}
