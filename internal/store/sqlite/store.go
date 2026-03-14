package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/store"
)

type SQLiteStore struct {
	db *sql.DB
}

var _ store.Store = (*SQLiteStore)(nil)

func New(cfg config.StorageConfig) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir storage dir: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	store := &SQLiteStore{db: db}
	if err := store.migrate(context.Background()); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA busy_timeout = 5000;`,
		`CREATE TABLE IF NOT EXISTS inbound_messages (
			message_id TEXT PRIMARY KEY,
			channel TEXT NOT NULL,
			tenant_key TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			chat_type TEXT NOT NULL,
			thread_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			content TEXT NOT NULL,
			raw_json TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS conversation_turns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_key TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_conversation_turns_session_created_at ON conversation_turns(session_key, created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			session_key TEXT PRIMARY KEY,
			public_id TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_public_id ON sessions(public_id);`,
		`CREATE TABLE IF NOT EXISTS conversation_sessions (
			session_key TEXT PRIMARY KEY,
			scope_key TEXT NOT NULL,
			public_id TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_conversation_sessions_scope_updated ON conversation_sessions(scope_key, updated_at DESC);`,
		`CREATE TABLE IF NOT EXISTS active_scope_sessions (
			scope_key TEXT PRIMARY KEY,
			session_key TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS session_summaries (
			session_key TEXT PRIMARY KEY,
			summary TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS memories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_key TEXT NOT NULL,
			memory_key TEXT NOT NULL,
			memory_value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			UNIQUE(session_key, memory_key)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_memories_session_updated ON memories(session_key, updated_at DESC);`,
		`CREATE TABLE IF NOT EXISTS pending_actions (
			session_key TEXT PRIMARY KEY,
			action_id TEXT NOT NULL,
			action_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			summary TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS artifacts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_key TEXT NOT NULL,
			artifact_id TEXT NOT NULL UNIQUE,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_artifacts_session_created ON artifacts(session_key, created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS scheduled_tasks (
			task_id TEXT PRIMARY KEY,
			scope_key TEXT NOT NULL,
			session_key TEXT NOT NULL,
			channel TEXT NOT NULL,
			tenant_key TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			thread_id TEXT NOT NULL,
			creator_id TEXT NOT NULL,
			title TEXT NOT NULL,
			prompt TEXT NOT NULL,
			interval_seconds INTEGER NOT NULL,
			status TEXT NOT NULL,
			last_summary TEXT NOT NULL DEFAULT '',
			next_run_at TIMESTAMP NOT NULL,
			last_run_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_scope_updated ON scheduled_tasks(scope_key, updated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_due ON scheduled_tasks(status, next_run_at);`,
		`CREATE TABLE IF NOT EXISTS scheduled_task_runs (
			run_id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			scope_key TEXT NOT NULL,
			session_key TEXT NOT NULL,
			status TEXT NOT NULL,
			summary TEXT NOT NULL,
			report TEXT NOT NULL,
			entities_json TEXT NOT NULL,
			started_at TIMESTAMP NOT NULL,
			finished_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_task_runs_scope_started ON scheduled_task_runs(scope_key, started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_task_runs_task_started ON scheduled_task_runs(task_id, started_at DESC);`,
		`CREATE TABLE IF NOT EXISTS scheduled_task_state (
			task_id TEXT PRIMARY KEY,
			state_json TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS scheduled_task_entities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			entity_key TEXT NOT NULL,
			kind TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			title TEXT NOT NULL,
			host_name TEXT NOT NULL,
			client_id TEXT NOT NULL,
			severity TEXT NOT NULL,
			status TEXT NOT NULL,
			note TEXT NOT NULL,
			last_summary TEXT NOT NULL,
			first_seen_at TIMESTAMP NOT NULL,
			last_seen_at TIMESTAMP NOT NULL,
			last_reported_at TIMESTAMP,
			UNIQUE(task_id, entity_key)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_task_entities_task_seen ON scheduled_task_entities(task_id, last_seen_at DESC);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("run migration %q: %w", stmt, err)
		}
	}

	return nil
}

func (s *SQLiteStore) RecordInboundMessage(ctx context.Context, msg protocol.InboundMessage) (bool, error) {
	query := `INSERT INTO inbound_messages(message_id, channel, tenant_key, chat_id, chat_type, thread_id, sender_id, content, raw_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query,
		msg.MessageID,
		string(msg.Channel),
		msg.TenantKey,
		msg.ChatID,
		msg.ChatType,
		msg.ThreadID,
		msg.SenderID,
		msg.Text,
		msg.RawJSON,
		msg.ReceivedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: inbound_messages.message_id") {
			return false, nil
		}
		return false, fmt.Errorf("insert inbound message: %w", err)
	}
	return true, nil
}

func (s *SQLiteStore) EnsureSession(ctx context.Context, sessionKey string) (protocol.SessionRef, error) {
	row := s.db.QueryRowContext(ctx, `SELECT public_id, created_at FROM sessions WHERE session_key = ?`, sessionKey)
	var session protocol.SessionRef
	if err := row.Scan(&session.PublicID, &session.CreatedAt); err == nil {
		session.Key = sessionKey
		return session, nil
	} else if err != sql.ErrNoRows {
		return protocol.SessionRef{}, fmt.Errorf("get session: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.SessionRef{}, fmt.Errorf("begin session tx: %w", err)
	}
	defer tx.Rollback()

	row = tx.QueryRowContext(ctx, `SELECT public_id, created_at FROM sessions WHERE session_key = ?`, sessionKey)
	if err := row.Scan(&session.PublicID, &session.CreatedAt); err == nil {
		session.Key = sessionKey
		if err := tx.Commit(); err != nil {
			return protocol.SessionRef{}, fmt.Errorf("commit session tx: %w", err)
		}
		return session, nil
	} else if err != sql.ErrNoRows {
		return protocol.SessionRef{}, fmt.Errorf("get session in tx: %w", err)
	}

	now := time.Now().UTC()
	prefix := "S-" + now.Format("20060102") + "-"
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM sessions WHERE public_id LIKE ?`, prefix+"%").Scan(&count); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("count daily sessions: %w", err)
	}

	session = protocol.SessionRef{
		Key:       sessionKey,
		PublicID:  fmt.Sprintf("%s%03d", prefix, count+1),
		CreatedAt: now,
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sessions(session_key, public_id, created_at) VALUES(?, ?, ?)`, session.Key, session.PublicID, session.CreatedAt); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("insert session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("commit session tx: %w", err)
	}
	return session, nil
}

func (s *SQLiteStore) EnsureActiveSession(ctx context.Context, scopeKey string) (protocol.SessionRef, error) {
	item, err := s.activeSession(ctx, scopeKey)
	if err != nil {
		return protocol.SessionRef{}, err
	}
	if item.Key != "" {
		return item, nil
	}
	return s.CreateSession(ctx, scopeKey, "")
}

func (s *SQLiteStore) CreateSession(ctx context.Context, scopeKey string, title string) (protocol.SessionRef, error) {
	now := time.Now().UTC()
	prefix := "S-" + now.Format("20060102") + "-"
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM conversation_sessions WHERE public_id LIKE ?`, prefix+"%").Scan(&count); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("count conversation sessions: %w", err)
	}
	publicID := fmt.Sprintf("%s%03d", prefix, count+1)
	if strings.TrimSpace(title) == "" {
		title = publicID
	}
	sessionKey := scopeKey + "::" + publicID
	item := protocol.SessionRef{
		Key:       sessionKey,
		ScopeKey:  scopeKey,
		PublicID:  publicID,
		Title:     title,
		Status:    "open",
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.SessionRef{}, fmt.Errorf("begin create session tx: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_sessions(session_key, scope_key, public_id, title, status, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, item.Key, item.ScopeKey, item.PublicID, item.Title, item.Status, item.CreatedAt, item.UpdatedAt); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("insert conversation session: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO active_scope_sessions(scope_key, session_key, updated_at) VALUES(?, ?, ?)
		ON CONFLICT(scope_key) DO UPDATE SET session_key = excluded.session_key, updated_at = excluded.updated_at`, scopeKey, item.Key, now); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("upsert active scope session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("commit create session tx: %w", err)
	}
	return item, nil
}

func (s *SQLiteStore) ListSessions(ctx context.Context, scopeKey string, limit int) ([]protocol.SessionRef, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT cs.session_key, cs.scope_key, cs.public_id, cs.title, cs.status, cs.created_at, cs.updated_at,
		CASE WHEN ass.session_key IS NOT NULL THEN 1 ELSE 0 END AS active
		FROM conversation_sessions cs
		LEFT JOIN active_scope_sessions ass ON ass.scope_key = cs.scope_key AND ass.session_key = cs.session_key
		WHERE cs.scope_key = ?
		ORDER BY cs.updated_at DESC
		LIMIT ?`, scopeKey, limit)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	var items []protocol.SessionRef
	for rows.Next() {
		var item protocol.SessionRef
		var active int
		if err := rows.Scan(&item.Key, &item.ScopeKey, &item.PublicID, &item.Title, &item.Status, &item.CreatedAt, &item.UpdatedAt, &active); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		item.Active = active == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) SetActiveSession(ctx context.Context, scopeKey string, publicID string) (protocol.SessionRef, error) {
	item, err := s.sessionByPublicID(ctx, scopeKey, publicID)
	if err != nil {
		return protocol.SessionRef{}, err
	}
	if item.Key == "" {
		return protocol.SessionRef{}, nil
	}
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `UPDATE conversation_sessions SET status = 'open', updated_at = ? WHERE session_key = ?`, now, item.Key); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("update session status: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO active_scope_sessions(scope_key, session_key, updated_at) VALUES(?, ?, ?)
		ON CONFLICT(scope_key) DO UPDATE SET session_key = excluded.session_key, updated_at = excluded.updated_at`, scopeKey, item.Key, now); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("set active session: %w", err)
	}
	item.Active = true
	item.Status = "open"
	item.UpdatedAt = now
	return item, nil
}

func (s *SQLiteStore) CloseActiveSession(ctx context.Context, scopeKey string) (protocol.SessionRef, error) {
	item, err := s.activeSession(ctx, scopeKey)
	if err != nil || item.Key == "" {
		return item, err
	}
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM active_scope_sessions WHERE scope_key = ?`, scopeKey); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("delete active scope session: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE conversation_sessions SET status = 'closed', updated_at = ? WHERE session_key = ?`, now, item.Key); err != nil {
		return protocol.SessionRef{}, fmt.Errorf("close session: %w", err)
	}
	item.Active = false
	item.Status = "closed"
	item.UpdatedAt = now
	return item, nil
}

func (s *SQLiteStore) DeleteSession(ctx context.Context, scopeKey string, publicID string) error {
	item, err := s.sessionByPublicID(ctx, scopeKey, publicID)
	if err != nil || item.Key == "" {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete session tx: %w", err)
	}
	defer tx.Rollback()
	stmts := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM active_scope_sessions WHERE scope_key = ? AND session_key = ?`, []any{scopeKey, item.Key}},
		{`DELETE FROM conversation_turns WHERE session_key = ?`, []any{item.Key}},
		{`DELETE FROM session_summaries WHERE session_key = ?`, []any{item.Key}},
		{`DELETE FROM memories WHERE session_key = ?`, []any{item.Key}},
		{`DELETE FROM pending_actions WHERE session_key = ?`, []any{item.Key}},
		{`DELETE FROM artifacts WHERE session_key = ?`, []any{item.Key}},
		{`DELETE FROM conversation_sessions WHERE session_key = ?`, []any{item.Key}},
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt.query, stmt.args...); err != nil {
			return fmt.Errorf("delete session data: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete session tx: %w", err)
	}
	return nil
}

func (s *SQLiteStore) activeSession(ctx context.Context, scopeKey string) (protocol.SessionRef, error) {
	row := s.db.QueryRowContext(ctx, `SELECT cs.session_key, cs.scope_key, cs.public_id, cs.title, cs.status, cs.created_at, cs.updated_at
		FROM active_scope_sessions ass
		JOIN conversation_sessions cs ON cs.session_key = ass.session_key
		WHERE ass.scope_key = ?`, scopeKey)
	var item protocol.SessionRef
	if err := row.Scan(&item.Key, &item.ScopeKey, &item.PublicID, &item.Title, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return protocol.SessionRef{}, nil
		}
		return protocol.SessionRef{}, fmt.Errorf("get active session: %w", err)
	}
	item.Active = true
	return item, nil
}

func (s *SQLiteStore) sessionByPublicID(ctx context.Context, scopeKey string, publicID string) (protocol.SessionRef, error) {
	row := s.db.QueryRowContext(ctx, `SELECT session_key, scope_key, public_id, title, status, created_at, updated_at FROM conversation_sessions WHERE scope_key = ? AND public_id = ? LIMIT 1`, scopeKey, publicID)
	var item protocol.SessionRef
	if err := row.Scan(&item.Key, &item.ScopeKey, &item.PublicID, &item.Title, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return protocol.SessionRef{}, nil
		}
		return protocol.SessionRef{}, fmt.Errorf("get session by public id: %w", err)
	}
	active, err := s.activeSession(ctx, scopeKey)
	if err == nil && active.Key == item.Key {
		item.Active = true
	}
	return item, nil
}

func (s *SQLiteStore) AppendTurn(ctx context.Context, sessionKey string, role string, content string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO conversation_turns(session_key, role, content, created_at) VALUES(?, ?, ?, ?)`,
		sessionKey,
		role,
		content,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("append turn: %w", err)
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE conversation_sessions SET updated_at = ? WHERE session_key = ?`, time.Now().UTC(), sessionKey)
	return nil
}

func (s *SQLiteStore) ListRecentTurns(ctx context.Context, sessionKey string, limit int) ([]protocol.Turn, error) {
	return s.listTurns(ctx, sessionKey, limit)
}

func (s *SQLiteStore) ListTurns(ctx context.Context, sessionKey string, limit int) ([]protocol.Turn, error) {
	return s.listTurns(ctx, sessionKey, limit)
}

func (s *SQLiteStore) listTurns(ctx context.Context, sessionKey string, limit int) ([]protocol.Turn, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT role, content, created_at FROM conversation_turns WHERE session_key = ? ORDER BY created_at DESC LIMIT ?`,
		sessionKey,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query recent turns: %w", err)
	}
	defer rows.Close()

	var reversed []protocol.Turn
	for rows.Next() {
		var turn protocol.Turn
		if err := rows.Scan(&turn.Role, &turn.Content, &turn.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recent turn: %w", err)
		}
		reversed = append(reversed, turn)
	}

	turns := make([]protocol.Turn, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		turns = append(turns, reversed[i])
	}

	return turns, rows.Err()
}

func (s *SQLiteStore) CountTurns(ctx context.Context, sessionKey string) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM conversation_turns WHERE session_key = ?`, sessionKey)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count turns: %w", err)
	}
	return count, nil
}

func (s *SQLiteStore) GetSessionSummary(ctx context.Context, sessionKey string) (string, error) {
	row := s.db.QueryRowContext(ctx, `SELECT summary FROM session_summaries WHERE session_key = ?`, sessionKey)
	var summary string
	if err := row.Scan(&summary); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("get session summary: %w", err)
	}
	return summary, nil
}

func (s *SQLiteStore) UpsertSessionSummary(ctx context.Context, sessionKey string, summary string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO session_summaries(session_key, summary, updated_at) VALUES(?, ?, ?)
		 ON CONFLICT(session_key) DO UPDATE SET summary = excluded.summary, updated_at = excluded.updated_at`,
		sessionKey,
		summary,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert session summary: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListMemories(ctx context.Context, sessionKey string, limit int) ([]protocol.MemoryEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT memory_key, memory_value, updated_at FROM memories WHERE session_key = ? ORDER BY updated_at DESC LIMIT ?`, sessionKey, limit)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()

	var items []protocol.MemoryEntry
	for rows.Next() {
		var item protocol.MemoryEntry
		item.SessionKey = sessionKey
		if err := rows.Scan(&item.Key, &item.Value, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) UpsertMemory(ctx context.Context, sessionKey string, key string, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO memories(session_key, memory_key, memory_value, updated_at) VALUES(?, ?, ?, ?)
		ON CONFLICT(session_key, memory_key) DO UPDATE SET memory_value = excluded.memory_value, updated_at = excluded.updated_at`, sessionKey, key, value, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("upsert memory: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteMemory(ctx context.Context, sessionKey string, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM memories WHERE session_key = ? AND memory_key = ?`, sessionKey, key)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CountMemories(ctx context.Context, sessionKey string) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM memories WHERE session_key = ?`, sessionKey)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count memories: %w", err)
	}
	return count, nil
}

func (s *SQLiteStore) SavePendingAction(ctx context.Context, sessionKey string, actionType string, payload string, summary string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO pending_actions(session_key, action_id, action_type, payload, summary, created_at) VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_key) DO UPDATE SET action_id = excluded.action_id, action_type = excluded.action_type, payload = excluded.payload, summary = excluded.summary, created_at = excluded.created_at`, sessionKey, fmt.Sprintf("pa-%d", time.Now().UnixNano()), actionType, payload, summary, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("save pending action: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetPendingAction(ctx context.Context, sessionKey string) (protocol.PendingAction, error) {
	row := s.db.QueryRowContext(ctx, `SELECT action_id, action_type, payload, summary, created_at FROM pending_actions WHERE session_key = ?`, sessionKey)
	var item protocol.PendingAction
	item.SessionKey = sessionKey
	if err := row.Scan(&item.ActionID, &item.ActionType, &item.Payload, &item.Summary, &item.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return protocol.PendingAction{}, nil
		}
		return protocol.PendingAction{}, fmt.Errorf("get pending action: %w", err)
	}
	return item, nil
}

func (s *SQLiteStore) DeletePendingAction(ctx context.Context, sessionKey string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pending_actions WHERE session_key = ?`, sessionKey)
	if err != nil {
		return fmt.Errorf("delete pending action: %w", err)
	}
	return nil
}

func (s *SQLiteStore) SaveArtifact(ctx context.Context, sessionKey string, kind string, title string, content string) (protocol.Artifact, error) {
	item := protocol.Artifact{
		SessionKey: sessionKey,
		ArtifactID: fmt.Sprintf("af-%d", time.Now().UnixNano()),
		Kind:       kind,
		Title:      title,
		Content:    content,
		CreatedAt:  time.Now().UTC(),
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO artifacts(session_key, artifact_id, kind, title, content, created_at) VALUES(?, ?, ?, ?, ?, ?)`, item.SessionKey, item.ArtifactID, item.Kind, item.Title, item.Content, item.CreatedAt)
	if err != nil {
		return protocol.Artifact{}, fmt.Errorf("save artifact: %w", err)
	}
	return item, nil
}

func (s *SQLiteStore) GetLatestArtifact(ctx context.Context, sessionKey string) (protocol.Artifact, error) {
	row := s.db.QueryRowContext(ctx, `SELECT artifact_id, kind, title, content, created_at FROM artifacts WHERE session_key = ? ORDER BY created_at DESC LIMIT 1`, sessionKey)
	return scanArtifactRow(row, sessionKey)
}

func (s *SQLiteStore) GetArtifact(ctx context.Context, sessionKey string, artifactID string) (protocol.Artifact, error) {
	row := s.db.QueryRowContext(ctx, `SELECT artifact_id, kind, title, content, created_at FROM artifacts WHERE session_key = ? AND artifact_id = ? LIMIT 1`, sessionKey, artifactID)
	return scanArtifactRow(row, sessionKey)
}

func scanArtifactRow(row *sql.Row, sessionKey string) (protocol.Artifact, error) {
	var item protocol.Artifact
	item.SessionKey = sessionKey
	if err := row.Scan(&item.ArtifactID, &item.Kind, &item.Title, &item.Content, &item.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return protocol.Artifact{}, nil
		}
		return protocol.Artifact{}, fmt.Errorf("get latest artifact: %w", err)
	}
	return item, nil
}

func (s *SQLiteStore) CreateScheduledTask(ctx context.Context, task protocol.ScheduledTask) (protocol.ScheduledTask, error) {
	now := time.Now().UTC()
	prefix := "T-" + now.Format("20060102") + "-"
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM scheduled_tasks WHERE task_id LIKE ?`, prefix+"%").Scan(&count); err != nil {
		return protocol.ScheduledTask{}, fmt.Errorf("count scheduled tasks: %w", err)
	}
	task.TaskID = fmt.Sprintf("%s%03d", prefix, count+1)
	task.CreatedAt = now
	task.UpdatedAt = now
	if task.NextRunAt.IsZero() {
		task.NextRunAt = now.Add(time.Duration(task.IntervalSeconds) * time.Second)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO scheduled_tasks(task_id, scope_key, session_key, channel, tenant_key, chat_id, thread_id, creator_id, title, prompt, interval_seconds, status, last_summary, next_run_at, last_run_at, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.TaskID, task.ScopeKey, task.SessionKey, string(task.Channel), task.TenantKey, task.ChatID, task.ThreadID, task.CreatorID,
		task.Title, task.Prompt, task.IntervalSeconds, task.Status, task.LastSummary, task.NextRunAt, nullableTime(task.LastRunAt), task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return protocol.ScheduledTask{}, fmt.Errorf("insert scheduled task: %w", err)
	}
	return task, nil
}

func (s *SQLiteStore) ListScheduledTasks(ctx context.Context, scopeKey string, limit int) ([]protocol.ScheduledTask, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT task_id, scope_key, session_key, channel, tenant_key, chat_id, thread_id, creator_id, title, prompt, interval_seconds, status, last_summary, next_run_at, last_run_at, created_at, updated_at
		FROM scheduled_tasks WHERE scope_key = ? AND status != 'deleted' ORDER BY updated_at DESC LIMIT ?`, scopeKey, limit)
	if err != nil {
		return nil, fmt.Errorf("list scheduled tasks: %w", err)
	}
	defer rows.Close()
	var items []protocol.ScheduledTask
	for rows.Next() {
		item, err := scanScheduledTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) GetScheduledTask(ctx context.Context, scopeKey string, taskID string) (protocol.ScheduledTask, error) {
	row := s.db.QueryRowContext(ctx, `SELECT task_id, scope_key, session_key, channel, tenant_key, chat_id, thread_id, creator_id, title, prompt, interval_seconds, status, last_summary, next_run_at, last_run_at, created_at, updated_at
		FROM scheduled_tasks WHERE scope_key = ? AND task_id = ? LIMIT 1`, scopeKey, taskID)
	return scanScheduledTaskRow(row)
}

func (s *SQLiteStore) GetScheduledTaskByID(ctx context.Context, taskID string) (protocol.ScheduledTask, error) {
	row := s.db.QueryRowContext(ctx, `SELECT task_id, scope_key, session_key, channel, tenant_key, chat_id, thread_id, creator_id, title, prompt, interval_seconds, status, last_summary, next_run_at, last_run_at, created_at, updated_at
		FROM scheduled_tasks WHERE task_id = ? LIMIT 1`, taskID)
	return scanScheduledTaskRow(row)
}

func (s *SQLiteStore) UpdateScheduledTask(ctx context.Context, scopeKey string, taskID string, patch protocol.ScheduledTaskPatch) (protocol.ScheduledTask, error) {
	item, err := s.GetScheduledTask(ctx, scopeKey, taskID)
	if err != nil || item.TaskID == "" {
		return item, err
	}
	if strings.TrimSpace(patch.Title) != "" {
		item.Title = strings.TrimSpace(patch.Title)
	}
	if strings.TrimSpace(patch.Prompt) != "" {
		item.Prompt = strings.TrimSpace(patch.Prompt)
	}
	if patch.IntervalSeconds > 0 {
		item.IntervalSeconds = patch.IntervalSeconds
	}
	if strings.TrimSpace(patch.Status) != "" {
		item.Status = strings.TrimSpace(patch.Status)
	}
	if strings.TrimSpace(patch.LastSummary) != "" {
		item.LastSummary = strings.TrimSpace(patch.LastSummary)
	}
	if !patch.NextRunAt.IsZero() {
		item.NextRunAt = patch.NextRunAt
	}
	if !patch.LastRunAt.IsZero() {
		item.LastRunAt = patch.LastRunAt
	}
	item.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `UPDATE scheduled_tasks SET title=?, prompt=?, interval_seconds=?, status=?, last_summary=?, next_run_at=?, last_run_at=?, updated_at=? WHERE scope_key=? AND task_id=?`,
		item.Title, item.Prompt, item.IntervalSeconds, item.Status, item.LastSummary, item.NextRunAt, nullableTime(item.LastRunAt), item.UpdatedAt, scopeKey, taskID)
	if err != nil {
		return protocol.ScheduledTask{}, fmt.Errorf("update scheduled task: %w", err)
	}
	return item, nil
}

func (s *SQLiteStore) DeleteScheduledTask(ctx context.Context, scopeKey string, taskID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE scheduled_tasks SET status='deleted', updated_at=? WHERE scope_key=? AND task_id=?`, time.Now().UTC(), scopeKey, taskID)
	if err != nil {
		return fmt.Errorf("delete scheduled task: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListDueScheduledTasks(ctx context.Context, now time.Time, limit int) ([]protocol.ScheduledTask, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT task_id, scope_key, session_key, channel, tenant_key, chat_id, thread_id, creator_id, title, prompt, interval_seconds, status, last_summary, next_run_at, last_run_at, created_at, updated_at
		FROM scheduled_tasks WHERE status='active' AND next_run_at <= ? ORDER BY next_run_at ASC LIMIT ?`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("list due scheduled tasks: %w", err)
	}
	defer rows.Close()
	var items []protocol.ScheduledTask
	for rows.Next() {
		item, err := scanScheduledTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) MarkScheduledTaskRunning(ctx context.Context, taskID string, lastRunAt time.Time, nextRunAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE scheduled_tasks SET last_run_at=?, next_run_at=?, updated_at=? WHERE task_id=?`, lastRunAt, nextRunAt, time.Now().UTC(), taskID)
	if err != nil {
		return fmt.Errorf("mark scheduled task running: %w", err)
	}
	return nil
}

func (s *SQLiteStore) SaveScheduledTaskRun(ctx context.Context, run protocol.ScheduledTaskRun) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO scheduled_task_runs(run_id, task_id, scope_key, session_key, status, summary, report, entities_json, started_at, finished_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, run.RunID, run.TaskID, run.ScopeKey, run.SessionKey, run.Status, run.Summary, run.Report, run.EntitiesJSON, run.StartedAt, run.FinishedAt)
	if err != nil {
		return fmt.Errorf("save scheduled task run: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetLatestScheduledTaskRun(ctx context.Context, scopeKey string) (protocol.ScheduledTaskRun, error) {
	row := s.db.QueryRowContext(ctx, `SELECT run_id, task_id, scope_key, session_key, status, summary, report, entities_json, started_at, finished_at
		FROM scheduled_task_runs WHERE scope_key = ? ORDER BY started_at DESC LIMIT 1`, scopeKey)
	var item protocol.ScheduledTaskRun
	if err := row.Scan(&item.RunID, &item.TaskID, &item.ScopeKey, &item.SessionKey, &item.Status, &item.Summary, &item.Report, &item.EntitiesJSON, &item.StartedAt, &item.FinishedAt); err != nil {
		if err == sql.ErrNoRows {
			return protocol.ScheduledTaskRun{}, nil
		}
		return protocol.ScheduledTaskRun{}, fmt.Errorf("get latest scheduled task run: %w", err)
	}
	return item, nil
}

func (s *SQLiteStore) GetScheduledTaskState(ctx context.Context, taskID string) (string, error) {
	row := s.db.QueryRowContext(ctx, `SELECT state_json FROM scheduled_task_state WHERE task_id = ?`, taskID)
	var state string
	if err := row.Scan(&state); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("get scheduled task state: %w", err)
	}
	return state, nil
}

func (s *SQLiteStore) UpsertScheduledTaskState(ctx context.Context, taskID string, state string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO scheduled_task_state(task_id, state_json, updated_at) VALUES(?, ?, ?)
		ON CONFLICT(task_id) DO UPDATE SET state_json=excluded.state_json, updated_at=excluded.updated_at`, taskID, state, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("upsert scheduled task state: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListScheduledTaskEntities(ctx context.Context, taskID string, limit int) ([]protocol.ScheduledTaskEntity, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT task_id, entity_key, kind, entity_id, title, host_name, client_id, severity, status, note, last_summary, first_seen_at, last_seen_at, last_reported_at
		FROM scheduled_task_entities WHERE task_id = ? ORDER BY last_seen_at DESC LIMIT ?`, taskID, limit)
	if err != nil {
		return nil, fmt.Errorf("list scheduled task entities: %w", err)
	}
	defer rows.Close()
	var items []protocol.ScheduledTaskEntity
	for rows.Next() {
		var item protocol.ScheduledTaskEntity
		var lastReported sql.NullTime
		if err := rows.Scan(&item.TaskID, &item.EntityKey, &item.Kind, &item.EntityID, &item.Title, &item.HostName, &item.ClientID, &item.Severity, &item.Status, &item.Note, &item.LastSummary, &item.FirstSeenAt, &item.LastSeenAt, &lastReported); err != nil {
			return nil, fmt.Errorf("scan scheduled task entity: %w", err)
		}
		if lastReported.Valid {
			item.LastReportedAt = lastReported.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) UpsertScheduledTaskEntity(ctx context.Context, entity protocol.ScheduledTaskEntity) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO scheduled_task_entities(task_id, entity_key, kind, entity_id, title, host_name, client_id, severity, status, note, last_summary, first_seen_at, last_seen_at, last_reported_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id, entity_key) DO UPDATE SET kind=excluded.kind, entity_id=excluded.entity_id, title=excluded.title, host_name=excluded.host_name, client_id=excluded.client_id, severity=excluded.severity, status=excluded.status, note=excluded.note, last_summary=excluded.last_summary, last_seen_at=excluded.last_seen_at, last_reported_at=excluded.last_reported_at`,
		entity.TaskID, entity.EntityKey, entity.Kind, entity.EntityID, entity.Title, entity.HostName, entity.ClientID, entity.Severity, entity.Status, entity.Note, entity.LastSummary, entity.FirstSeenAt, entity.LastSeenAt, nullableTime(entity.LastReportedAt))
	if err != nil {
		return fmt.Errorf("upsert scheduled task entity: %w", err)
	}
	return nil
}

func scanScheduledTask(rows *sql.Rows) (protocol.ScheduledTask, error) {
	var item protocol.ScheduledTask
	var channel string
	var lastRun sql.NullTime
	if err := rows.Scan(&item.TaskID, &item.ScopeKey, &item.SessionKey, &channel, &item.TenantKey, &item.ChatID, &item.ThreadID, &item.CreatorID, &item.Title, &item.Prompt, &item.IntervalSeconds, &item.Status, &item.LastSummary, &item.NextRunAt, &lastRun, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return protocol.ScheduledTask{}, fmt.Errorf("scan scheduled task: %w", err)
	}
	item.Channel = protocol.Channel(channel)
	if lastRun.Valid {
		item.LastRunAt = lastRun.Time
	}
	return item, nil
}

func scanScheduledTaskRow(row *sql.Row) (protocol.ScheduledTask, error) {
	var item protocol.ScheduledTask
	var channel string
	var lastRun sql.NullTime
	if err := row.Scan(&item.TaskID, &item.ScopeKey, &item.SessionKey, &channel, &item.TenantKey, &item.ChatID, &item.ThreadID, &item.CreatorID, &item.Title, &item.Prompt, &item.IntervalSeconds, &item.Status, &item.LastSummary, &item.NextRunAt, &lastRun, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return protocol.ScheduledTask{}, nil
		}
		return protocol.ScheduledTask{}, fmt.Errorf("scan scheduled task row: %w", err)
	}
	item.Channel = protocol.Channel(channel)
	if lastRun.Valid {
		item.LastRunAt = lastRun.Time
	}
	return item, nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
