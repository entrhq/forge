package logging

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// setupTestDir creates a temporary directory for test logs and resets global state
func setupTestDir(t *testing.T) (cleanup func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "forge-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Save original state
	origLogDir := logDir
	origInitErr := initErr
	origInitOnce := initOnce
	origSessionID := sessionID
	origSessionIDOnce := sessionIDOnce

	// Reset global state
	logDir = tempDir
	initErr = nil
	initOnce = sync.Once{}
	sessionID = ""
	sessionIDOnce = sync.Once{}

	// Return cleanup function
	return func() {
		// Restore original state
		logDir = origLogDir
		initErr = origInitErr
		initOnce = origInitOnce
		sessionID = origSessionID
		sessionIDOnce = origSessionIDOnce

		// Remove temp directory
		os.RemoveAll(tempDir)
	}
}

func TestNewLogger(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	logger, err := NewLogger("test-component")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	if logger.component != "test-component" {
		t.Errorf("Expected component 'test-component', got %q", logger.component)
	}

	if logger.sessionID == "" {
		t.Error("Expected non-empty session ID")
	}

	if logger.logPath == "" {
		t.Error("Expected non-empty log path")
	}

	// Verify log file exists
	if _, err := os.Stat(logger.logPath); os.IsNotExist(err) {
		t.Errorf("Log file does not exist at %s", logger.logPath)
	}
}

func TestLoggerFormatting(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	logger, err := NewLogger("test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write a test message
	logger.Printf("Test message %d", 123)
	logger.Debugf("Debug message")
	logger.Infof("Info message")
	logger.Warnf("Warning message")
	logger.Errorf("Error message")

	// Give file system time to flush
	time.Sleep(50 * time.Millisecond)

	// Read log file
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify each log level appears
	expectedPatterns := []string{
		"[test] [INFO] Test message 123",
		"[test] [DEBUG] Debug message",
		"[test] [INFO] Info message",
		"[test] [WARN] Warning message",
		"[test] [ERROR] Error message",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(logContent, pattern) {
			t.Errorf("Log content missing expected pattern: %q\nContent:\n%s", pattern, logContent)
		}
	}
}

func TestMultipleComponents(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// Create two loggers with different components
	logger1, err := NewLogger("component1")
	if err != nil {
		t.Fatalf("Failed to create logger1: %v", err)
	}
	defer logger1.Close()

	logger2, err := NewLogger("component2")
	if err != nil {
		t.Fatalf("Failed to create logger2: %v", err)
	}
	defer logger2.Close()

	// They should share the same session ID and log file
	if logger1.sessionID != logger2.sessionID {
		t.Errorf("Expected same session ID, got %q and %q", logger1.sessionID, logger2.sessionID)
	}

	if logger1.logPath != logger2.logPath {
		t.Errorf("Expected same log path, got %q and %q", logger1.logPath, logger2.logPath)
	}

	// Write from both loggers
	logger1.Printf("Message from component1")
	logger2.Printf("Message from component2")

	time.Sleep(50 * time.Millisecond)

	// Read log file
	content, err := os.ReadFile(logger1.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify both components logged
	if !strings.Contains(logContent, "[component1]") {
		t.Error("Log missing component1 entries")
	}
	if !strings.Contains(logContent, "[component2]") {
		t.Error("Log missing component2 entries")
	}
}

func TestGetSessionID(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	id1 := GetSessionID()
	id2 := GetSessionID()

	if id1 != id2 {
		t.Errorf("Expected consistent session ID, got %q and %q", id1, id2)
	}

	if id1 == "" {
		t.Error("Expected non-empty session ID")
	}
}

func TestGetLogDirectory(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	dir, err := GetLogDirectory()
	if err != nil {
		t.Fatalf("Failed to get log directory: %v", err)
	}

	// In test mode, we use a temp directory, so just verify it exists and is a directory
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("Log directory does not exist or is not a directory: %s", dir)
	}
}

func TestLoggerClose(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	logger, err := NewLogger("test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Close once
	if err := logger.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}

	// Close again should be safe
	if err := logger.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

func TestLogPathFormat(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	logger, err := NewLogger("test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Verify log file name format: <session-id>-forge.log
	fileName := filepath.Base(logger.logPath)
	if !strings.HasSuffix(fileName, "-forge.log") {
		t.Errorf("Expected log file to end with '-forge.log', got %q", fileName)
	}

	// Verify it starts with a UUID-like session ID (has dashes)
	sessionPart := strings.TrimSuffix(fileName, "-forge.log")
	if !strings.Contains(sessionPart, "-") {
		t.Errorf("Expected session ID part to contain dashes (UUID format), got %q", sessionPart)
	}
}
