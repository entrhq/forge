package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Logger provides structured debug logging for Forge components.
// All logs are written to a session-specific file in ~/.forge/logs/
//
// All log methods (Debugf, Infof, Warnf, Errorf) write unconditionally.
// There is currently no log level filtering.
type Logger struct {
	sessionID  string
	component  string
	file       *os.File
	logger     *log.Logger
	mu         sync.Mutex
	logPath    string
	closeOnce  sync.Once
}

var (
	// Global session ID for the current execution
	sessionID     string
	sessionIDOnce sync.Once

	// logDir is the directory where log files are stored
	logDir string

	// initOnce ensures directory initialization happens once
	initOnce sync.Once
	
	// initErr stores any error from directory initialization
	initErr error
)

// getSessionID returns or creates the session ID for this execution
func getSessionID() string {
	sessionIDOnce.Do(func() {
		sessionID = uuid.New().String()
	})
	return sessionID
}

// initLogDirectory ensures the log directory exists
func initLogDirectory() error {
	initOnce.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			initErr = fmt.Errorf("failed to get home directory: %w", err)
			return
		}

		logDir = filepath.Join(homeDir, ".forge", "logs")
		if err := os.MkdirAll(logDir, 0750); err != nil {
			initErr = fmt.Errorf("failed to create log directory: %w", err)
			return
		}
	})
	return initErr
}

// NewLogger creates a new logger for a specific component.
// The logger writes to ~/.forge/logs/<session-id>-forge.log
// 
// If the log directory cannot be created or the log file cannot be opened,
// it returns a fallback logger that writes to stderr along with the error.
// Callers can check the error to detect fallback mode and log warnings.
func NewLogger(component string) (*Logger, error) {
	if err := initLogDirectory(); err != nil {
		// Fallback to stderr if we can't create the log directory
		return newFallbackLogger(component, err), err
	}

	sessID := getSessionID()
	logFileName := fmt.Sprintf("%s-forge.log", sessID)
	logPath := filepath.Join(logDir, logFileName)

	// Open log file in append mode (multiple components may write to same file)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return newFallbackLogger(component, fmt.Errorf("failed to open log file: %w", err)), err
	}

	logger := log.New(file, "", 0) // We'll format timestamps ourselves

	return &Logger{
		sessionID: sessID,
		component: component,
		file:      file,
		logger:    logger,
		logPath:   logPath,
	}, nil
}

// newFallbackLogger creates a logger that writes to stderr when file logging fails
func newFallbackLogger(component string, err error) *Logger {
	logger := log.New(os.Stderr, fmt.Sprintf("[%s] ", component), log.LstdFlags|log.Lshortfile)
	logger.Printf("WARNING: Failed to initialize file logging: %v", err)
	logger.Printf("Falling back to stderr logging")

	return &Logger{
		sessionID: getSessionID(),
		component: component,
		file:      nil, // No file, using stderr
		logger:    logger,
		logPath:   "",
	}
}

// formatLogEntry creates a structured log entry with timestamp, component, and level
func (l *Logger) formatLogEntry(level, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	return fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, l.component, level, message)
}

// Printf logs a formatted message
func (l *Logger) Printf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	message := fmt.Sprintf(format, v...)
	entry := l.formatLogEntry("INFO", message)
	l.logger.Println(entry)
}

// Debugf logs a debug-level message
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	message := fmt.Sprintf(format, v...)
	entry := l.formatLogEntry("DEBUG", message)
	l.logger.Println(entry)
}

// Infof logs an info-level message
func (l *Logger) Infof(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	message := fmt.Sprintf(format, v...)
	entry := l.formatLogEntry("INFO", message)
	l.logger.Println(entry)
}

// Warnf logs a warning-level message
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	message := fmt.Sprintf(format, v...)
	entry := l.formatLogEntry("WARN", message)
	l.logger.Println(entry)
}

// Errorf logs an error-level message
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	message := fmt.Sprintf(format, v...)
	entry := l.formatLogEntry("ERROR", message)
	l.logger.Println(entry)
}

// Writer returns an io.Writer that writes to this logger
func (l *Logger) Writer() io.Writer {
	if l.file != nil {
		return l.file
	}
	return os.Stderr
}

// SessionID returns the current session ID
func (l *Logger) SessionID() string {
	return l.sessionID
}

// LogPath returns the path to the log file
func (l *Logger) LogPath() string {
	return l.logPath
}

// Close closes the log file. Safe to call multiple times.
func (l *Logger) Close() error {
	var err error
	l.closeOnce.Do(func() {
		if l.file != nil {
			err = l.file.Close()
		}
	})
	return err
}

// GetSessionID returns the current global session ID
func GetSessionID() string {
	return getSessionID()
}

// GetLogDirectory returns the directory where logs are stored
func GetLogDirectory() (string, error) {
	if err := initLogDirectory(); err != nil {
		return "", err
	}
	return logDir, nil
}
