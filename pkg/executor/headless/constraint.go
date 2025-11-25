package headless

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/gobwas/glob"
)

// ConstraintManager enforces safety limits during headless execution
type ConstraintManager struct {
	config *ConstraintConfig
	mode   ExecutionMode // Track execution mode for validation

	// Runtime state tracking
	filesModified map[string]*FileModification
	tokensUsed    int
	startTime     time.Time

	// Pattern matching
	patternMatcher *PatternMatcher

	mu sync.RWMutex
}

// FileModification tracks modifications to a single file
type FileModification struct {
	Path         string
	LinesAdded   int
	LinesRemoved int
}

// ConstraintViolation represents a constraint violation error
type ConstraintViolation struct {
	Type    ViolationType
	Message string
	Details map[string]interface{}
}

func (e *ConstraintViolation) Error() string {
	return fmt.Sprintf("constraint violation (%s): %s", e.Type, e.Message)
}

// ViolationType identifies the type of constraint that was violated
type ViolationType string

const (
	ViolationFileCount       ViolationType = "file_count"
	ViolationLineCount       ViolationType = "line_count"
	ViolationFilePattern     ViolationType = "file_pattern"
	ViolationToolRestriction ViolationType = "tool_restriction"
	ViolationTokenLimit      ViolationType = "token_limit"
	ViolationTimeout         ViolationType = "timeout"
	ViolationReadOnlyMode    ViolationType = "read_only_mode"
)

// NewConstraintManager creates a new constraint manager
func NewConstraintManager(config ConstraintConfig, mode ExecutionMode) (*ConstraintManager, error) {
	// Create pattern matcher
	patternMatcher, err := NewPatternMatcher(config.AllowedPatterns, config.DeniedPatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to create pattern matcher: %w", err)
	}

	return &ConstraintManager{
		config:         &config,
		mode:           mode,
		filesModified:  make(map[string]*FileModification),
		startTime:      time.Now(),
		patternMatcher: patternMatcher,
	}, nil
}

// ValidateToolCall validates a tool call against constraints
func (cm *ConstraintManager) ValidateToolCall(toolName string, args interface{}) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Loop-breaking tools are always allowed and should not be restricted
	if isLoopBreakingTool(toolName) {
		return nil
	}

	// Enforce read-only mode: reject all file-modifying tools
	if cm.mode == ModeReadOnly && isFileModifyingTool(toolName) {
		return &ConstraintViolation{
			Type:    ViolationReadOnlyMode,
			Message: fmt.Sprintf("tool '%s' is not allowed in read-only mode", toolName),
			Details: map[string]interface{}{
				"tool": toolName,
				"mode": string(cm.mode),
			},
		}
	}

	// Check if tool is allowed
	if len(cm.config.AllowedTools) > 0 {
		allowed := false
		for _, allowedTool := range cm.config.AllowedTools {
			if allowedTool == toolName {
				allowed = true
				break
			}
		}
		if !allowed {
			return &ConstraintViolation{
				Type:    ViolationToolRestriction,
				Message: fmt.Sprintf("tool '%s' is not in allowed tools list", toolName),
				Details: map[string]interface{}{
					"tool":          toolName,
					"allowed_tools": cm.config.AllowedTools,
				},
			}
		}
	}

	// For file-modifying tools, check file patterns
	if isFileModifyingTool(toolName) {
		filePath, err := extractFilePath(args)
		if err == nil && filePath != "" {
			if !cm.patternMatcher.IsAllowed(filePath) {
				return &ConstraintViolation{
					Type:    ViolationFilePattern,
					Message: fmt.Sprintf("file '%s' does not match allowed patterns", filePath),
					Details: map[string]interface{}{
						"file":             filePath,
						"allowed_patterns": cm.config.AllowedPatterns,
						"denied_patterns":  cm.config.DeniedPatterns,
					},
				}
			}
		}
	}

	return nil
}

// RecordFileModification records a file modification and validates against limits
func (cm *ConstraintManager) RecordFileModification(path string, linesAdded, linesRemoved int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Normalize path
	path = filepath.Clean(path)

	// Check if this is a new file modification
	if _, exists := cm.filesModified[path]; !exists {
		// Check file count limit
		if cm.config.MaxFiles > 0 && len(cm.filesModified) >= cm.config.MaxFiles {
			return &ConstraintViolation{
				Type:    ViolationFileCount,
				Message: fmt.Sprintf("maximum file count exceeded (%d)", cm.config.MaxFiles),
				Details: map[string]interface{}{
					"max_files":      cm.config.MaxFiles,
					"current_count":  len(cm.filesModified),
					"attempted_file": path,
				},
			}
		}
	}

	// Record modification
	if mod, exists := cm.filesModified[path]; exists {
		mod.LinesAdded += linesAdded
		mod.LinesRemoved += linesRemoved
	} else {
		cm.filesModified[path] = &FileModification{
			Path:         path,
			LinesAdded:   linesAdded,
			LinesRemoved: linesRemoved,
		}
	}

	// Check total lines changed limit
	if cm.config.MaxLinesChanged > 0 {
		totalLinesChanged := cm.calculateTotalLinesAdded() + cm.calculateTotalLinesRemoved()
		if totalLinesChanged > cm.config.MaxLinesChanged {
			return &ConstraintViolation{
				Type:    ViolationLineCount,
				Message: fmt.Sprintf("maximum lines changed exceeded (%d)", cm.config.MaxLinesChanged),
				Details: map[string]interface{}{
					"max_lines_changed": cm.config.MaxLinesChanged,
					"current_total":     totalLinesChanged,
					"file":              path,
				},
			}
		}
	}

	return nil
}

// RecordTokenUsage records token usage and validates against limit
func (cm *ConstraintManager) RecordTokenUsage(tokens int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.tokensUsed += tokens

	if cm.config.MaxTokens > 0 && cm.tokensUsed > cm.config.MaxTokens {
		return &ConstraintViolation{
			Type:    ViolationTokenLimit,
			Message: fmt.Sprintf("maximum token usage exceeded (%d)", cm.config.MaxTokens),
			Details: map[string]interface{}{
				"max_tokens":  cm.config.MaxTokens,
				"tokens_used": cm.tokensUsed,
			},
		}
	}

	return nil
}

// CheckTimeout checks if execution has exceeded the timeout
func (cm *ConstraintManager) CheckTimeout() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.config.Timeout <= 0 {
		return nil // No timeout configured
	}

	elapsed := time.Since(cm.startTime)
	if elapsed > cm.config.Timeout {
		return &ConstraintViolation{
			Type:    ViolationTimeout,
			Message: fmt.Sprintf("execution timeout exceeded (%v)", cm.config.Timeout),
			Details: map[string]interface{}{
				"timeout": cm.config.Timeout,
				"elapsed": elapsed,
			},
		}
	}

	return nil
}

// GetCurrentState returns the current constraint state
func (cm *ConstraintManager) GetCurrentState() *ConstraintState {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	files := make([]FileModification, 0, len(cm.filesModified))
	for _, mod := range cm.filesModified {
		files = append(files, *mod)
	}

	return &ConstraintState{
		FilesModified:     files,
		TotalFiles:        len(cm.filesModified),
		TotalLinesAdded:   cm.calculateTotalLinesAdded(),
		TotalLinesRemoved: cm.calculateTotalLinesRemoved(),
		TokensUsed:        cm.tokensUsed,
		Elapsed:           time.Since(cm.startTime),
	}
}

// ConstraintState represents the current state of constraint tracking
type ConstraintState struct {
	FilesModified     []FileModification
	TotalFiles        int
	TotalLinesAdded   int
	TotalLinesRemoved int
	TokensUsed        int
	Elapsed           time.Duration
}

// calculateTotalLinesAdded calculates total lines added across all files
// Must be called with lock held
func (cm *ConstraintManager) calculateTotalLinesAdded() int {
	total := 0
	for _, mod := range cm.filesModified {
		total += mod.LinesAdded
	}
	return total
}

// calculateTotalLinesRemoved calculates total lines removed across all files
// Must be called with lock held
func (cm *ConstraintManager) calculateTotalLinesRemoved() int {
	total := 0
	for _, mod := range cm.filesModified {
		total += mod.LinesRemoved
	}
	return total
}

// isFileModifyingTool returns true if the tool modifies files or executes commands
func isFileModifyingTool(toolName string) bool {
	switch toolName {
	case "write_file", "apply_diff", "execute_command":
		return true
	default:
		return false
	}
}

// isLoopBreakingTool returns true if the tool is a loop-breaking tool
// Loop-breaking tools (task_completion, ask_question, converse) should always
// be allowed as they are essential for agent communication and control flow
func isLoopBreakingTool(toolName string) bool {
	switch toolName {
	case "task_completion", "ask_question", "converse":
		return true
	default:
		return false
	}
}

// extractFilePath extracts the file path from tool arguments.
// Returns the path and an error if extraction fails.
func extractFilePath(args interface{}) (string, error) {
	if args == nil {
		return "", fmt.Errorf("tool arguments are nil")
	}

	argsMap, ok := args.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("tool arguments are not a map[string]interface{}, got type %T", args)
	}

	pathValue, exists := argsMap["path"]
	if !exists {
		return "", fmt.Errorf("'path' field not found in tool arguments")
	}

	path, ok := pathValue.(string)
	if !ok {
		return "", fmt.Errorf("'path' field is not a string, got type %T", pathValue)
	}

	if path == "" {
		return "", fmt.Errorf("'path' field is empty")
	}

	return path, nil
}

// PatternMatcher handles glob pattern matching for file access control
type PatternMatcher struct {
	allowedPatterns []glob.Glob
	deniedPatterns  []glob.Glob
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(allowed, denied []string) (*PatternMatcher, error) {
	pm := &PatternMatcher{}

	// Compile allowed patterns
	for _, pattern := range allowed {
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed pattern '%s': %w", pattern, err)
		}
		pm.allowedPatterns = append(pm.allowedPatterns, g)
	}

	// Compile denied patterns
	for _, pattern := range denied {
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid denied pattern '%s': %w", pattern, err)
		}
		pm.deniedPatterns = append(pm.deniedPatterns, g)
	}

	return pm, nil
}

// IsAllowed returns true if the path is allowed by the pattern rules
func (pm *PatternMatcher) IsAllowed(path string) bool {
	// Normalize path
	path = filepath.Clean(path)

	// Denied patterns take precedence
	for _, pattern := range pm.deniedPatterns {
		if pattern.Match(path) {
			return false
		}
	}

	// If no allowed patterns specified, allow all (except denied)
	if len(pm.allowedPatterns) == 0 {
		return true
	}

	// Check if path matches any allowed pattern
	for _, pattern := range pm.allowedPatterns {
		if pattern.Match(path) {
			return true
		}
	}

	return false
}
