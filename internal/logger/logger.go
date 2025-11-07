package logger

import (
	"log/slog"
	"os"
	"sync"
)

var (
	globalLogger *slog.Logger
	debugEnabled bool
	mu           sync.RWMutex
)

// SetGlobal sets the global logger and debug state
func SetGlobal(logger *slog.Logger, debug bool) {
	mu.Lock()
	defer mu.Unlock()
	globalLogger = logger
	debugEnabled = debug
}

// Get returns the global logger instance
func Get() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()

	if globalLogger != nil {
		return globalLogger
	}

	// Fallback logger
	level := slog.LevelInfo
	if debugEnabled {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	mu.RLock()
	defer mu.RUnlock()
	return debugEnabled
}