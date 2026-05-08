package session

import (
	"testing"
	"time"

	"rm_ai_agent/internal/planner"
)

func TestScopedToolCallCacheKeyIncludesSession(t *testing.T) {
	call := planner.ToolCall{Name: "edr_incidents", ClientID: "c-1", Page: 1, PageSize: 20}
	k1 := scopedToolCallCacheKey("scope:a::S-1", call)
	k2 := scopedToolCallCacheKey("scope:a::S-2", call)
	if k1 == k2 {
		t.Fatalf("expected different dedup keys for different sessions, got same key: %s", k1)
	}
}

func TestToolDedupCacheResetSessionOnlyClearsTargetSession(t *testing.T) {
	cache := newToolDedupCache(30 * time.Second)
	call := planner.ToolCall{Name: "edr_incidents", ClientID: "c-1", Page: 1, PageSize: 20}

	k1 := scopedToolCallCacheKey("scope:a::S-1", call)
	k2 := scopedToolCallCacheKey("scope:a::S-2", call)

	if _, _, hit := cache.GetOrSubmit(k1); hit {
		t.Fatal("expected first submit for session 1 to miss")
	}
	cache.Done(k1, "result-s1", nil)

	if _, _, hit := cache.GetOrSubmit(k2); hit {
		t.Fatal("expected first submit for session 2 to miss")
	}
	cache.Done(k2, "result-s2", nil)

	cache.ResetSession("scope:a::S-1")

	if _, _, hit := cache.GetOrSubmit(k1); hit {
		t.Fatal("expected session 1 key to be cleared")
	}
	if result, err, hit := cache.GetOrSubmit(k2); !hit || err != nil || result != "result-s2" {
		t.Fatalf("expected session 2 key to remain fresh, hit=%v err=%v result=%q", hit, err, result)
	}
}
