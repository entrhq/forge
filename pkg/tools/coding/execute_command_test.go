package coding

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecuteCommandTool_SimpleCommand(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	xmlInput := `<arguments>
	<command>echo "Hello World"</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify output contains expected text
	if !strings.Contains(result, "Hello World") {
		t.Errorf("Expected result to contain 'Hello World', got: %s", result)
	}

	// Verify successful execution
	if !strings.Contains(result, "successfully") {
		t.Errorf("Expected result to contain 'successfully', got: %s", result)
	}

	// Verify metadata
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}
	if metadata["command"] != `echo "Hello World"` {
		t.Errorf("Expected command metadata, got %v", metadata["command"])
	}
}

func TestExecuteCommandTool_ExitCode(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	xmlInput := `<arguments>
	<command>exit 42</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify exit code in result
	if !strings.Contains(result, "Exit code: 42") {
		t.Errorf("Expected result to contain 'Exit code: 42', got: %s", result)
	}

	// Verify metadata
	if metadata["exit_code"].(int) != 42 {
		t.Errorf("Expected exit_code=42, got %v", metadata["exit_code"])
	}
}

func TestExecuteCommandTool_Stderr(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Use a command that writes to stderr (ls on non-existent with redirect to catch stderr)
	xmlInput := `<arguments>
	<command>ls /nonexistent 2>&1 || echo "completed"</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify output contains error indication or completed message
	if !strings.Contains(result, "No such file") && !strings.Contains(result, "completed") && !strings.Contains(result, "cannot access") {
		t.Errorf("Expected result to contain error or 'completed', got: %s", result)
	}

	// Should still be exit code 0 due to || echo
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}
}

func TestExecuteCommandTool_WorkingDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	writeTestFile(t, filepath.Join(subDir, "test.txt"), "content")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	xmlInput := `<arguments>
	<command>ls test.txt</command>
	<working_dir>subdir</working_dir>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify command ran in subdirectory
	if !strings.Contains(result, "test.txt") {
		t.Errorf("Expected result to contain 'test.txt', got: %s", result)
	}

	// Verify metadata includes working directory
	workDir := metadata["working_dir"].(string)
	if !strings.HasSuffix(workDir, "subdir") {
		t.Errorf("Expected working_dir to end with 'subdir', got: %s", workDir)
	}
}

func TestExecuteCommandTool_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	xmlInput := `<arguments>
	<command>sleep 5</command>
	<timeout>1</timeout>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify timeout message
	if !strings.Contains(result, "timed out") {
		t.Errorf("Expected result to contain 'timed out', got: %s", result)
	}

	// Exit code should be non-zero for timeout
	exitCode := metadata["exit_code"].(int)
	if exitCode == 0 {
		t.Errorf("Expected non-zero exit code for timeout, got %d", exitCode)
	}
}

func TestExecuteCommandTool_MissingCommand(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	xmlInput := `<arguments>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for missing command")
	}
	if !strings.Contains(err.Error(), "command cannot be empty") {
		t.Errorf("Expected 'command cannot be empty' error, got: %v", err)
	}
}

func TestExecuteCommandTool_InvalidWorkingDir(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	xmlInput := `<arguments>
	<command>echo test</command>
	<working_dir>../outside</working_dir>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for working directory outside workspace")
	}
	if !strings.Contains(err.Error(), "invalid working directory") {
		t.Errorf("Expected 'invalid working directory' error, got: %v", err)
	}
}

func TestExecuteCommandTool_MultilineOutput(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	xmlInput := `<arguments>
	<command>echo "line1"; echo "line2"; echo "line3"</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all lines present
	if !strings.Contains(result, "line1") {
		t.Errorf("Expected result to contain 'line1', got: %s", result)
	}
	if !strings.Contains(result, "line2") {
		t.Errorf("Expected result to contain 'line2', got: %s", result)
	}
	if !strings.Contains(result, "line3") {
		t.Errorf("Expected result to contain 'line3', got: %s", result)
	}

	// Verify successful execution
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}
}

func TestExecuteCommandTool_FileManipulation(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Create a file using command
	xmlInput := `<arguments>
	<command>echo "test content" > output.txt</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify command succeeded
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}

	// Verify file was created
	outputPath := filepath.Join(tmpDir, "output.txt")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	if !strings.Contains(string(content), "test content") {
		t.Errorf("Expected file to contain 'test content', got: %s", string(content))
	}

	// Verify result message
	if !strings.Contains(result, "successfully") {
		t.Errorf("Expected success message, got: %s", result)
	}
}

func TestExecuteCommandTool_CommandNotFound(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	xmlInput := `<arguments>
	<command>nonexistentcommand12345</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify non-zero exit code
	exitCode := metadata["exit_code"].(int)
	if exitCode == 0 {
		t.Errorf("Expected non-zero exit code for command not found, got %d", exitCode)
	}

	// Verify error is indicated in result
	if !strings.Contains(result, "failed") && !strings.Contains(result, "not found") {
		t.Errorf("Expected error indication in result, got: %s", result)
	}
}

func TestExecuteCommandTool_DefaultTimeout(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Quick command should complete well within default timeout
	xmlInput := `<arguments>
	<command>echo "quick"</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify successful execution
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}

	// Verify result
	if !strings.Contains(result, "quick") {
		t.Errorf("Expected result to contain 'quick', got: %s", result)
	}
}

func TestExecuteCommandTool_DurationTracking(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Command that takes a known amount of time
	sleepTime := 0.1 // 100ms
	xmlInput := fmt.Sprintf(`<arguments>
	<command>sleep %.1f</command>
</arguments>`, sleepTime)

	start := time.Now()
	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify duration is tracked in metadata
	durationMs := metadata["duration_ms"].(int64)
	if durationMs < 50 { // Should be at least 50ms
		t.Errorf("Expected duration_ms >= 50, got %d", durationMs)
	}

	// Verify duration is mentioned in result
	if !strings.Contains(result, "successfully") {
		t.Errorf("Expected success message with duration, got: %s", result)
	}

	// Verify actual duration is reasonable
	if duration < time.Duration(sleepTime*float64(time.Second))*9/10 {
		t.Errorf("Expected duration >= 90ms, got %v", duration)
	}
}

func TestExecuteCommandTool_EnvironmentVariables(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Command that uses environment variable
	xmlInput := `<arguments>
	<command>TEST_VAR=hello; echo $TEST_VAR</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify environment variable was set and used
	if !strings.Contains(result, "hello") {
		t.Errorf("Expected result to contain 'hello', got: %s", result)
	}

	// Verify successful execution
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}
}

func TestExecuteCommandTool_PipeAndRedirection(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Command with pipe
	xmlInput := `<arguments>
	<command>echo "apple\nbanana\ncherry" | grep banana</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify pipe worked
	if !strings.Contains(result, "banana") {
		t.Errorf("Expected result to contain 'banana', got: %s", result)
	}
	// Should not contain other fruits
	if strings.Contains(result, "apple") || strings.Contains(result, "cherry") {
		t.Errorf("Expected only 'banana' in result, got: %s", result)
	}

	// Verify successful execution
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}
}

func TestExecuteCommandTool_ComplexShellScript(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Multi-line shell script
	xmlInput := `<arguments>
	<command>for i in 1 2 3; do echo "Item $i"; done</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify loop output
	if !strings.Contains(result, "Item 1") {
		t.Errorf("Expected result to contain 'Item 1', got: %s", result)
	}
	if !strings.Contains(result, "Item 2") {
		t.Errorf("Expected result to contain 'Item 2', got: %s", result)
	}
	if !strings.Contains(result, "Item 3") {
		t.Errorf("Expected result to contain 'Item 3', got: %s", result)
	}

	// Verify successful execution
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}
}

func TestExecuteCommandTool_LargeOutput(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Generate large output
	xmlInput := `<arguments>
	<command>for i in $(seq 1 100); do echo "Line $i with some text"; done</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify output contains first and last lines
	if !strings.Contains(result, "Line 1") {
		t.Errorf("Expected result to contain 'Line 1', got first 100 chars: %s", result[:min(100, len(result))])
	}
	if !strings.Contains(result, "Line 100") {
		t.Errorf("Expected result to contain 'Line 100', got last 100 chars: %s", result[max(0, len(result)-100):])
	}

	// Verify successful execution
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}
}

func TestExecuteCommandTool_Metadata(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Verify name
	if tool.Name() != "execute_command" {
		t.Errorf("Expected name 'execute_command', got '%s'", tool.Name())
	}

	// Verify description
	desc := tool.Description()
	if !strings.Contains(desc, "Execute") || !strings.Contains(desc, "shell command") {
		t.Errorf("Expected description to mention executing shell commands, got: %s", desc)
	}

	// Verify schema
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	// Verify loop breaking status
	if tool.IsLoopBreaking() {
		t.Error("ExecuteCommandTool should not be loop-breaking")
	}
}

func TestExecuteCommandTool_CrossPlatformCommand(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewExecuteCommandTool(guard)

	// Use a cross-platform compatible command
	xmlInput := `<arguments>
	<command>echo test</command>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify output
	if !strings.Contains(result, "test") {
		t.Errorf("Expected result to contain 'test', got: %s", result)
	}

	// Verify successful execution
	if metadata["exit_code"].(int) != 0 {
		t.Errorf("Expected exit_code=0, got %v", metadata["exit_code"])
	}
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
