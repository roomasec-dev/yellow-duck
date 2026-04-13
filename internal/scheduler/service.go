package scheduler

import (
	"context"
	"fmt"
	"time"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/session"
	"rm_ai_agent/internal/store"
)

type Notifier interface {
	SendChatText(ctx context.Context, chatID string, text string) error
}

type Service struct {
	cfg      config.SchedulerConfig
	store    store.Store
	sessions *session.Service
	notify   Notifier
	logger   *logx.Logger
}

func NewService(cfg config.SchedulerConfig, dataStore store.Store, sessions *session.Service, notify Notifier, logger *logx.Logger) *Service {
	return &Service{cfg: cfg, store: dataStore, sessions: sessions, notify: notify, logger: logger}
}

func (s *Service) Start(ctx context.Context) {
	if s == nil || !s.cfg.Enabled || s.store == nil || s.sessions == nil || s.notify == nil {
		return
	}
	ticker := time.NewTicker(time.Duration(s.cfg.PollSeconds) * time.Second)
	defer ticker.Stop()
	s.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Service) runOnce(ctx context.Context) {
	now := time.Now().UTC()
	tasks, err := s.store.ListDueScheduledTasks(ctx, s.cfg.ScopeKey, now, 10)
	if err != nil {
		s.logger.Warn("list due scheduled tasks failed", "error", err)
		return
	}
	for _, task := range tasks {
		nextRunAt := now.Add(time.Duration(task.IntervalSeconds) * time.Second)
		if err := s.store.MarkScheduledTaskRunning(ctx, task.TaskID, now, nextRunAt); err != nil {
			s.logger.Warn("mark scheduled task running failed", "task_id", task.TaskID, "error", err)
			continue
		}
		exec, err := s.sessions.RunScheduledTask(ctx, task)
		if err != nil {
			s.logger.Warn("scheduled task execution failed", "task_id", task.TaskID, "error", err)
			run := protocol.ScheduledTaskRun{
				RunID:      fmt.Sprintf("tr-%d", time.Now().UnixNano()),
				TaskID:     task.TaskID,
				ScopeKey:   task.ScopeKey,
				SessionKey: task.SessionKey,
				Status:     "failed",
				Summary:    err.Error(),
				Report:     err.Error(),
				StartedAt:  now,
				FinishedAt: time.Now().UTC(),
			}
			_ = s.store.SaveScheduledTaskRun(ctx, run)
			_, _ = s.store.UpdateScheduledTask(ctx, task.ScopeKey, task.TaskID, protocol.ScheduledTaskPatch{LastSummary: "执行失败：" + err.Error()})
			continue
		}
		if exec.Message == "" {
			s.logger.Info("scheduled task finished without notification", "task_id", task.TaskID, "summary", exec.Summary)
			continue
		}
		if err := s.notify.SendChatText(ctx, task.ChatID, exec.Message); err != nil {
			s.logger.Warn("send scheduled task notification failed", "task_id", task.TaskID, "chat_id", task.ChatID, "error", err)
			continue
		}
		s.logger.Info("scheduled task notification sent", "task_id", task.TaskID, "chat_id", task.ChatID)
	}
}
