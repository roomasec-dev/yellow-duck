package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rm_ai_agent/internal/config"
)

func TestSearchTraversesMarkdownFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "linux", "hunt.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("# Linux Hunt\n\n这里记录 SSH 横向移动排查手册。\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService(config.KnowledgeBaseConfig{Enabled: true, Path: root, SearchLimit: 10, SnippetLength: 80})
	matches, err := svc.Search("SSH 横向移动")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("unexpected matches: %+v", matches)
	}
	if matches[0].RelPath != "linux/hunt.md" {
		t.Fatalf("unexpected rel path: %s", matches[0].RelPath)
	}
	if !strings.Contains(matches[0].Snippet, "SSH") {
		t.Fatalf("unexpected snippet: %s", matches[0].Snippet)
	}
}

func TestUpsertUpdatesExistingFileByHeading(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "custom-name.md")
	if err := os.WriteFile(path, []byte("# 应急响应手册\n\n旧内容\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService(config.KnowledgeBaseConfig{Enabled: true, Path: root, SearchLimit: 10, SnippetLength: 80})
	item, err := svc.Upsert("应急响应手册", "补充内容", "append", "", "")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if item.RelPath != "custom-name.md" {
		t.Fatalf("unexpected rel path: %s", item.RelPath)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "旧内容") || !strings.Contains(text, "补充内容") {
		t.Fatalf("unexpected content: %s", text)
	}
	if _, err := os.Stat(filepath.Join(root, "应急响应手册.md")); !os.IsNotExist(err) {
		t.Fatalf("unexpected new file created: %v", err)
	}
}

func TestDeleteByRelativePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "runbook", "windows", "rdp.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("# RDP 手册\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService(config.KnowledgeBaseConfig{Enabled: true, Path: root, SearchLimit: 10, SnippetLength: 80})
	if err := svc.Delete("runbook/windows/rdp.md"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
}

func TestReplaceTextRequiresExistingTargetText(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "kb.md")
	if err := os.WriteFile(path, []byte("# KB\n\nalpha beta\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService(config.KnowledgeBaseConfig{Enabled: true, Path: root, SearchLimit: 10, SnippetLength: 80})
	if _, err := svc.Upsert("kb.md", "", "replace_text", "gamma", "delta"); err == nil {
		t.Fatal("expected replace_text to fail when old text is missing")
	}
}
