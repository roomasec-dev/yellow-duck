package logx

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"rm_ai_agent/internal/config"
)

type Logger struct {
	base  *log.Logger
	level config.Level
	file  *os.File
}

func New(level config.Level, filePath string) (*Logger, error) {
	writer := io.Writer(os.Stdout)
	var file *os.File
	if strings.TrimSpace(filePath) != "" {
		dir := filepath.Dir(filePath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create log directory: %w", err)
			}
		}
		opened, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		file = opened
		writer = io.MultiWriter(os.Stdout, file)
	}

	return &Logger{
		base:  log.New(writer, "", log.LstdFlags),
		level: level,
		file:  file,
	}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *Logger) Debug(msg string, kv ...any) {
	l.log(config.LevelDebug, "DEBUG", msg, kv...)
}

func (l *Logger) Info(msg string, kv ...any) {
	l.log(config.LevelInfo, "INFO", msg, kv...)
}

func (l *Logger) Warn(msg string, kv ...any) {
	l.log(config.LevelWarn, "WARN", msg, kv...)
}

func (l *Logger) Error(msg string, kv ...any) {
	l.log(config.LevelError, "ERROR", msg, kv...)
}

func (l *Logger) log(level config.Level, label string, msg string, kv ...any) {
	if level < l.level {
		return
	}

	b := strings.Builder{}
	b.WriteString(label)
	b.WriteString(" ")
	b.WriteString(msg)
	for i := 0; i+1 < len(kv); i += 2 {
		b.WriteString(" ")
		b.WriteString(fmt.Sprint(kv[i]))
		b.WriteString("=")
		b.WriteString(fmt.Sprint(kv[i+1]))
	}
	if len(kv)%2 == 1 {
		b.WriteString(" extra=")
		b.WriteString(fmt.Sprint(kv[len(kv)-1]))
	}

	l.base.Println(b.String())
}
