package headless

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// LogLevel represents the logging verbosity level
type LogLevel int

const (
	// LogLevelQuiet shows only critical information (errors, warnings, final summary)
	LogLevelQuiet LogLevel = iota
	// LogLevelNormal shows standard execution progress (default)
	LogLevelNormal
	// LogLevelVerbose shows detailed execution information
	LogLevelVerbose
	// LogLevelDebug shows all internal details for debugging
	LogLevelDebug
)

// Logger provides structured, beautiful logging for headless execution
type Logger struct {
	level  LogLevel
	writer io.Writer

	// ANSI color codes
	colorReset     string
	colorGreen     string
	colorCyan      string
	colorSalmon    string
	colorYellow    string
	colorRed       string
	colorWhite     string
	colorGray      string
	colorBoldGreen string
	colorBoldRed   string
	colorBoldWhite string

	// Execution state
	startTime time.Time
	stepCount int
}

// NewLogger creates a new logger with the specified level
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:          level,
		writer:         os.Stdout,
		colorReset:     "\033[0m",
		colorGreen:     "\033[32m",
		colorCyan:      "\033[36m",
		colorSalmon:    "\033[38;5;217m", // Salmon pink #FFB3BA
		colorYellow:    "\033[33m",
		colorRed:       "\033[31m",
		colorWhite:     "\033[37m",
		colorGray:      "\033[90m",
		colorBoldGreen: "\033[1;32m",
		colorBoldRed:   "\033[1;31m",
		colorBoldWhite: "\033[1;37m",
		startTime:      time.Now(),
	}
}

// Header prints a prominent header message
func (l *Logger) Header(message string) {
	if l.level >= LogLevelNormal {
		fmt.Fprintf(l.writer, "\n%s%s%s\n", l.colorBoldWhite, strings.Repeat("=", 70), l.colorReset)
		fmt.Fprintf(l.writer, "%s  %s%s\n", l.colorBoldWhite, message, l.colorReset)
		fmt.Fprintf(l.writer, "%s%s%s\n", l.colorBoldWhite, strings.Repeat("=", 70), l.colorReset)
	}
}

// Section prints a section divider
func (l *Logger) Section(title string) {
	if l.level >= LogLevelNormal {
		fmt.Fprintln(l.writer)
		fmt.Fprintf(l.writer, "%s▶ %s%s\n", l.colorCyan, title, l.colorReset)
		fmt.Fprintf(l.writer, "%s%s%s\n", l.colorGray, strings.Repeat("─", 50), l.colorReset)
	}
}

// Step prints a numbered step in the execution
func (l *Logger) Step(message string) {
	if l.level >= LogLevelNormal {
		l.stepCount++
		fmt.Fprintf(l.writer, "\n%s[%d] %s%s\n", l.colorCyan, l.stepCount, message, l.colorReset)
	}
}

// Successf prints a success message with checkmark
func (l *Logger) Successf(format string, args ...interface{}) {
	if l.level >= LogLevelNormal {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(l.writer, "%s✓ %s%s\n", l.colorBoldGreen, msg, l.colorReset)
	}
}

// Infof prints an informational message
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.level >= LogLevelNormal {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(l.writer, "%s%s%s\n", l.colorSalmon, msg, l.colorReset)
	}
}

// Warningf prints a warning message
func (l *Logger) Warningf(format string, args ...interface{}) {
	if l.level >= LogLevelQuiet {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(l.writer, "%s⚠ Warning: %s%s\n", l.colorYellow, msg, l.colorReset)
	}
}

// Errorf prints an error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.level >= LogLevelQuiet {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(l.writer, "%s✗ Error: %s%s\n", l.colorBoldRed, msg, l.colorReset)
	}
}

// Verbosef prints detailed information (only in verbose mode)
func (l *Logger) Verbosef(format string, args ...interface{}) {
	if l.level >= LogLevelVerbose {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(l.writer, "%s→ %s%s\n", l.colorGray, msg, l.colorReset)
	}
}

// Debugf prints debug information (only in debug mode)
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.level >= LogLevelDebug {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(l.writer, "%s[DEBUG] %s%s\n", l.colorGray, msg, l.colorReset)
	}
}

// ToolCall logs a tool execution with formatting based on verbosity
func (l *Logger) ToolCall(toolName string, count int) {
	switch l.level {
	case LogLevelQuiet:
		// Don't log individual tool calls in quiet mode
	case LogLevelNormal:
		// Show compact progress indicator
		fmt.Fprintf(l.writer, "%s  • %s (#%d)%s\n", l.colorGray, toolName, count, l.colorReset)
	case LogLevelVerbose, LogLevelDebug:
		// Show detailed information
		fmt.Fprintf(l.writer, "%s  🔧 Tool: %s (call #%d)%s\n", l.colorCyan, toolName, count, l.colorReset)
	}
}

// FileModified logs a file modification
func (l *Logger) FileModified(path string, linesAdded, linesRemoved int) {
	if l.level >= LogLevelNormal {
		change := ""
		if linesAdded > 0 || linesRemoved > 0 {
			change = fmt.Sprintf(" (+%d/-%d)", linesAdded, linesRemoved)
		}
		fmt.Fprintf(l.writer, "%s  📝 Modified: %s%s%s\n", l.colorBoldGreen, path, change, l.colorReset)
	}
}

// QualityGate logs quality gate execution
func (l *Logger) QualityGate(name string, passed bool, message string) {
	if l.level >= LogLevelNormal {
		if passed {
			fmt.Fprintf(l.writer, "%s  ✓ %s: passed%s\n", l.colorBoldGreen, name, l.colorReset)
		} else {
			fmt.Fprintf(l.writer, "%s  ✗ %s: failed%s\n", l.colorBoldRed, name, l.colorReset)
			if message != "" {
				fmt.Fprintf(l.writer, "%s    %s%s\n", l.colorGray, message, l.colorReset)
			}
		}
	}
}

// QualityGateRetry logs a quality gate retry attempt
func (l *Logger) QualityGateRetry(attempt, maxRetries int) {
	if l.level >= LogLevelNormal {
		fmt.Fprintf(l.writer, "%s  Retrying quality gates (attempt %d/%d)%s\n", l.colorYellow, attempt, maxRetries, l.colorReset)
	}
}

// GitOperation logs a git operation
func (l *Logger) GitOperation(operation, details string) {
	if l.level >= LogLevelNormal {
		fmt.Fprintf(l.writer, "%s  🔀 Git: %s%s\n", l.colorCyan, operation, l.colorReset)
		if details != "" && l.level >= LogLevelVerbose {
			fmt.Fprintf(l.writer, "%s    %s%s\n", l.colorGray, details, l.colorReset)
		}
	}
}

// Progress shows a progress update (dots, spinner, etc.)
func (l *Logger) Progress(message string) {
	if l.level >= LogLevelNormal {
		fmt.Fprintf(l.writer, "%s  %s%s", l.colorGray, message, l.colorReset)
	}
}

// Summary prints a final execution summary
func (l *Logger) Summary(status string, summary *ExecutionSummary) {
	if l.level < LogLevelQuiet {
		return
	}

	l.printSummaryHeader()
	l.printStatus(status)
	l.printTaskAndDuration(summary)
	l.printMetrics(summary)
	l.printModifiedFiles(summary)
	l.printQualityGates(summary)
	l.printGitInfo(summary)
	l.printError(summary)
	l.printSummaryFooter()
}

func (l *Logger) printSummaryHeader() {
	fmt.Fprintln(l.writer)
	fmt.Fprintf(l.writer, "%s%s%s\n", l.colorBoldWhite, strings.Repeat("=", 70), l.colorReset)
	fmt.Fprintf(l.writer, "%s  EXECUTION SUMMARY%s\n", l.colorBoldWhite, l.colorReset)
	fmt.Fprintf(l.writer, "%s%s%s\n", l.colorBoldWhite, strings.Repeat("=", 70), l.colorReset)
}

func (l *Logger) printStatus(status string) {
	fmt.Fprint(l.writer, "  Status: ")
	switch status {
	case statusSuccess:
		fmt.Fprintf(l.writer, "%s✓ SUCCESS%s\n", l.colorBoldGreen, l.colorReset)
	case statusPartialSuccess:
		fmt.Fprintf(l.writer, "%s⚠ PARTIAL SUCCESS%s\n", l.colorYellow, l.colorReset)
	case statusFailed:
		fmt.Fprintf(l.writer, "%s✗ FAILED%s\n", l.colorBoldRed, l.colorReset)
	default:
		fmt.Fprintln(l.writer, status)
	}
}

func (l *Logger) printTaskAndDuration(summary *ExecutionSummary) {
	fmt.Fprintf(l.writer, "  Task: %s\n", summary.Task)
	fmt.Fprintf(l.writer, "  Duration: %s\n", summary.Duration.Round(time.Second))
}

func (l *Logger) printMetrics(summary *ExecutionSummary) {
	if summary.Metrics.FilesModified == 0 {
		return
	}

	fmt.Fprintf(l.writer, "\n  📊 Metrics:\n")
	fmt.Fprintf(l.writer, "    Files modified: %d\n", summary.Metrics.FilesModified)

	if summary.Metrics.TotalLinesAdded > 0 || summary.Metrics.TotalLinesRemoved > 0 {
		fmt.Fprintf(l.writer, "    Lines changed: +%d/-%d\n",
			summary.Metrics.TotalLinesAdded, summary.Metrics.TotalLinesRemoved)
	}

	fmt.Fprintf(l.writer, "    Tool calls: %d\n", summary.ToolCallCount)

	if summary.Metrics.TokensUsed > 0 {
		fmt.Fprintf(l.writer, "    Tokens used: %s\n", formatNumber(summary.Metrics.TokensUsed))
	}
}

func (l *Logger) printModifiedFiles(summary *ExecutionSummary) {
	if l.level < LogLevelVerbose || len(summary.FilesModified) == 0 {
		return
	}

	fmt.Fprintf(l.writer, "\n  📝 Modified Files:\n")
	for _, mod := range summary.FilesModified {
		change := ""
		if mod.LinesAdded > 0 || mod.LinesRemoved > 0 {
			change = fmt.Sprintf(" (+%d/-%d)", mod.LinesAdded, mod.LinesRemoved)
		}
		fmt.Fprintf(l.writer, "    • %s%s\n", mod.Path, change)
	}
}

func (l *Logger) printQualityGates(summary *ExecutionSummary) {
	if summary.QualityGateResults == nil || len(summary.QualityGateResults.Results) == 0 {
		return
	}

	fmt.Fprintf(l.writer, "\n  🎯 Quality Gates:\n")
	for _, result := range summary.QualityGateResults.Results {
		if result.Passed {
			fmt.Fprintf(l.writer, "%s    ✓ %s%s\n", l.colorBoldGreen, result.Name, l.colorReset)
		} else {
			fmt.Fprintf(l.writer, "%s    ✗ %s%s\n", l.colorBoldRed, result.Name, l.colorReset)
			if result.Error != "" && l.level >= LogLevelVerbose {
				fmt.Fprintf(l.writer, "%s      %s%s\n", l.colorGray, result.Error, l.colorReset)
			}
		}
	}
}

func (l *Logger) printGitInfo(summary *ExecutionSummary) {
	if summary.GitInfo == nil || summary.GitInfo.CommitHash == "" {
		return
	}

	fmt.Fprintf(l.writer, "\n  🔀 Git:\n")
	fmt.Fprintf(l.writer, "    Commit: %s\n", summary.GitInfo.CommitHash)
	if summary.PRURL != "" {
		fmt.Fprintf(l.writer, "    PR: %s\n", summary.PRURL)
	}
}

func (l *Logger) printError(summary *ExecutionSummary) {
	if summary.Error == "" {
		return
	}

	fmt.Fprintln(l.writer)
	fmt.Fprintf(l.writer, "%s  Error Details:%s\n", l.colorBoldRed, l.colorReset)
	fmt.Fprintf(l.writer, "%s    %s%s\n", l.colorRed, summary.Error, l.colorReset)
}

func (l *Logger) printSummaryFooter() {
	fmt.Fprintf(l.writer, "%s%s%s\n", l.colorBoldWhite, strings.Repeat("=", 70), l.colorReset)
	fmt.Fprintln(l.writer)
}

// Newline adds a blank line (respects log level)
func (l *Logger) Newline() {
	if l.level >= LogLevelNormal {
		fmt.Fprintln(l.writer)
	}
}

// parseLogLevel converts a string log level to LogLevel type
func parseLogLevel(level string) LogLevel {
	switch level {
	case "quiet":
		return LogLevelQuiet
	case "normal":
		return LogLevelNormal
	case "verbose":
		return LogLevelVerbose
	case "debug":
		return LogLevelDebug
	default:
		return LogLevelNormal
	}
}

// formatNumber formats large numbers with commas for readability
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}
