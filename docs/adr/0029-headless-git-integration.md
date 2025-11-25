# 29. Headless Git Integration

**Status:** Proposed
**Date:** 2025-01-21
**Deciders:** Forge Core Team
**Technical Story:** Enable automatic git operations (branch creation, commits, attribution) for headless CI/CD workflows

---

## Context

Headless mode (ADR-0026) enables autonomous code modifications with safety constraints (ADR-0027) and quality gates (ADR-0028). However, the final step—committing changes to version control—requires careful design to ensure proper attribution, traceability, and integration with team workflows.

### Background

In interactive mode, users manually commit changes through their IDE or git CLI. In headless CI/CD workflows, git operations must be automated while maintaining:

- **Attribution**: Clear indication that changes came from Forge, not a human
- **Traceability**: Link commits to the CI/CD run, task description, and execution context
- **Team Workflow Integration**: Support branch-first workflow for PR creation
- **Safety**: Prevent accidental pushes to protected branches
- **Flexibility**: Work with various git hosting platforms (GitHub, GitLab, Bitbucket)

Current challenges:
- CI/CD environments may have shallow clones or detached HEAD states
- Authentication varies by platform (SSH keys, tokens, bot accounts)
- Commit messages need to be informative yet automated
- Changes should be isolated on feature branches for review
- Must handle dirty workspaces and merge conflicts gracefully

### Problem Statement

We need a git integration system that:

1. **Creates feature branches automatically**: Isolate autonomous changes for PR workflows
2. **Commits with proper attribution**: Identify Forge as author with useful metadata
3. **Generates meaningful commit messages**: Describe what changed and why
4. **Handles workspace state**: Detect dirty workspaces, conflicts, and edge cases
5. **Supports branch-first workflow**: Create branch, commit, but don't push (v1.0 safety measure)
6. **Enables future auto-push/PR creation**: Architecture supports future enhancements

### Goals

- Implement robust git branch and commit operations for headless mode
- Generate informative commit messages automatically
- Support proper git author/committer attribution
- Handle common git workspace issues gracefully
- Enable branch-first workflow (create branch + commit, manual push)
- Provide foundation for future auto-push and PR creation

### Non-Goals

- Automatic push to remote (P2 feature - safety concern for v1.0)
- Automatic PR creation (P2 feature)
- Git conflict resolution (fail if conflicts detected)
- Multi-repo operations (future enhancement)
- Git hook execution (use native git hooks)

---

## Decision Drivers

* **Safety First**: No automatic pushes in v1.0, branch isolation for review
* **Attribution**: Clear identification of automated changes
* **Traceability**: Link commits to execution context and task
* **Developer Experience**: Works seamlessly with team git workflows
* **Platform Independence**: Not tied to specific git hosting platform
* **CI/CD Compatibility**: Handles shallow clones, detached HEAD, auth tokens

---

## Considered Options

### Option 1: Direct Git CLI Execution

**Description:** Execute git commands directly via shell (git commit, git branch, etc.).

**Pros:**
- Simple implementation
- No external dependencies
- Behavior matches developer experience
- Easy to debug (commands visible in logs)

**Cons:**
- Shell command construction is error-prone
- Harder to test in isolation
- Less structured error handling
- Platform-specific git binary paths

### Option 2: Go Git Library (go-git)

**Description:** Use a pure Go git implementation library like go-git.

**Pros:**
- Type-safe git operations
- No git binary dependency
- Easier to test (mock git operations)
- Structured error handling
- Cross-platform consistency

**Cons:**
- External dependency (large library)
- May not support all git features
- Potential compatibility issues with git servers
- Performance overhead vs. native git
- Learning curve for library API

### Option 3: Hybrid Approach (Wrap Git CLI)

**Description:** Create a GitManager abstraction that wraps git CLI commands with structured error handling and validation.

**Pros:**
- Simple implementation (shell commands)
- Uses native git (full compatibility)
- Structured Go interface for git operations
- Easy to test (can mock GitManager)
- No external dependencies beyond git binary
- Clear error messages and validation

**Cons:**
- Requires git binary in PATH
- Shell command construction (mitigated by abstraction)
- Platform-specific considerations

---

## Decision

**Chosen Option:** Option 3 - Hybrid Approach (Wrap Git CLI)

### Rationale

1. **Compatibility**: Native git ensures 100% compatibility with all git servers and features. No library will match git's complete feature set.

2. **Simplicity**: Wrapping shell commands is straightforward. We already have patterns for command execution (ExecuteCommandTool, quality gates).

3. **Testability**: The GitManager abstraction can be mocked for testing, providing same testability as a library.

4. **Debugging**: Developers can see exact git commands in logs, making troubleshooting easier.

5. **Zero Dependencies**: No need to vendor a large git library. Git binary is already a prerequisite for development.

6. **Proven Pattern**: The existing codebase uses shell commands successfully (see execute_command tool, quality gates).

The key insight is that we can get library-like benefits (structured interface, testability) while keeping implementation-level simplicity (shell commands).

---

## Consequences

### Positive

- Simple implementation using familiar git commands
- Full git compatibility (all features available)
- Easy debugging (commands visible in logs)
- No external library dependencies
- Testable via interface mocking
- Developers understand git command behavior

### Negative

- Requires git binary in CI/CD environment (reasonable assumption)
- Shell command construction requires care
- Platform-specific git binary paths (mitigated by $PATH)
- Error message parsing less structured than library

### Neutral

- GitManager abstraction layer (adds indirection but improves testability)
- Git operations logged for observability
- Workspace state validation required before operations

---

## Implementation

### Core Components

#### 1. GitManager Interface

```go
// GitManager handles git operations for headless mode
type GitManager interface {
    // Workspace state
    IsClean() (bool, error)
    GetCurrentBranch() (string, error)
    GetStatus() (*GitStatus, error)
    
    // Branch operations
    CreateBranch(name string) error
    CheckoutBranch(name string) error
    DeleteBranch(name string) error
    
    // Commit operations
    StageFiles(files []string) error
    Commit(message string, author *GitAuthor) error
    
    // Snapshot operations (for rollback)
    CreateSnapshot() (*GitSnapshot, error)
    RestoreSnapshot(snapshot *GitSnapshot) error
    
    // Information
    GetWorkspaceDir() string
}

// GitStatus represents workspace state
type GitStatus struct {
    CurrentBranch string
    IsClean       bool
    Staged        []string
    Unstaged      []string
    Untracked     []string
}

// GitAuthor represents commit author/committer
type GitAuthor struct {
    Name  string
    Email string
}

// GitSnapshot represents workspace state for rollback
type GitSnapshot struct {
    Branch        string
    CommitHash    string
    StagedFiles   []string
    UnstagedFiles []string
}
```

#### 2. GitManager Implementation

```go
type gitManager struct {
    workspaceDir string
    logger       *log.Logger
}

func NewGitManager(workspaceDir string) GitManager {
    return &gitManager{
        workspaceDir: workspaceDir,
        logger:       log.New(os.Stdout, "[GIT] ", log.LstdFlags),
    }
}

func (gm *gitManager) IsClean() (bool, error) {
    cmd := exec.Command("git", "status", "--porcelain")
    cmd.Dir = gm.workspaceDir
    
    output, err := cmd.Output()
    if err != nil {
        return false, fmt.Errorf("git status failed: %w", err)
    }
    
    // Empty output = clean workspace
    return len(strings.TrimSpace(string(output))) == 0, nil
}

func (gm *gitManager) GetCurrentBranch() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    cmd.Dir = gm.workspaceDir
    
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to get current branch: %w", err)
    }
    
    branch := strings.TrimSpace(string(output))
    if branch == "HEAD" {
        return "", fmt.Errorf("detached HEAD state detected")
    }
    
    return branch, nil
}

func (gm *gitManager) CreateBranch(name string) error {
    // Validate branch name
    if err := validateBranchName(name); err != nil {
        return fmt.Errorf("invalid branch name: %w", err)
    }
    
    // Create branch
    cmd := exec.Command("git", "checkout", "-b", name)
    cmd.Dir = gm.workspaceDir
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to create branch %s: %w\nOutput: %s", name, err, output)
    }
    
    gm.logger.Printf("Created and checked out branch: %s", name)
    return nil
}

func (gm *gitManager) StageFiles(files []string) error {
    if len(files) == 0 {
        return fmt.Errorf("no files to stage")
    }
    
    args := append([]string{"add"}, files...)
    cmd := exec.Command("git", args...)
    cmd.Dir = gm.workspaceDir
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to stage files: %w\nOutput: %s", err, output)
    }
    
    gm.logger.Printf("Staged %d files", len(files))
    return nil
}

func (gm *gitManager) Commit(message string, author *GitAuthor) error {
    args := []string{"commit", "-m", message}
    
    // Set author if provided
    if author != nil {
        authorStr := fmt.Sprintf("%s <%s>", author.Name, author.Email)
        args = append(args, "--author", authorStr)
    }
    
    cmd := exec.Command("git", args...)
    cmd.Dir = gm.workspaceDir
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("commit failed: %w\nOutput: %s", err, output)
    }
    
    gm.logger.Printf("Created commit: %s", message)
    return nil
}
```

#### 3. Snapshot/Rollback Implementation

```go
func (gm *gitManager) CreateSnapshot() (*GitSnapshot, error) {
    // Get current state
    branch, err := gm.GetCurrentBranch()
    if err != nil {
        return nil, fmt.Errorf("failed to get current branch: %w", err)
    }
    
    commitHash, err := gm.getCurrentCommit()
    if err != nil {
        return nil, fmt.Errorf("failed to get current commit: %w", err)
    }
    
    status, err := gm.GetStatus()
    if err != nil {
        return nil, fmt.Errorf("failed to get status: %w", err)
    }
    
    snapshot := &GitSnapshot{
        Branch:        branch,
        CommitHash:    commitHash,
        StagedFiles:   status.Staged,
        UnstagedFiles: status.Unstaged,
    }
    
    gm.logger.Printf("Created snapshot: branch=%s, commit=%s", branch, commitHash[:7])
    return snapshot, nil
}

func (gm *gitManager) RestoreSnapshot(snapshot *GitSnapshot) error {
    gm.logger.Printf("Restoring snapshot: branch=%s, commit=%s", 
        snapshot.Branch, snapshot.CommitHash[:7])
    
    // Reset to original commit (discards any commits made during execution)
    cmd := exec.Command("git", "reset", "--hard", snapshot.CommitHash)
    cmd.Dir = gm.workspaceDir
    
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to reset to commit %s: %w\nOutput: %s", 
            snapshot.CommitHash, err, output)
    }
    
    // Clean untracked files
    cmd = exec.Command("git", "clean", "-fd")
    cmd.Dir = gm.workspaceDir
    
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to clean workspace: %w\nOutput: %s", err, output)
    }
    
    // Checkout original branch if different
    currentBranch, err := gm.GetCurrentBranch()
    if err == nil && currentBranch != snapshot.Branch {
        cmd = exec.Command("git", "checkout", snapshot.Branch)
        cmd.Dir = gm.workspaceDir
        
        if output, err := cmd.CombinedOutput(); err != nil {
            return fmt.Errorf("failed to checkout branch %s: %w\nOutput: %s", 
                snapshot.Branch, err, output)
        }
    }
    
    gm.logger.Printf("Snapshot restored successfully")
    return nil
}

func (gm *gitManager) getCurrentCommit() (string, error) {
    cmd := exec.Command("git", "rev-parse", "HEAD")
    cmd.Dir = gm.workspaceDir
    
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to get current commit: %w", err)
    }
    
    return strings.TrimSpace(string(output)), nil
}
```

#### 4. Commit Message Generation

```go
// CommitMessageGenerator creates structured commit messages
type CommitMessageGenerator struct {
    llmClient LLMClient // Reuse existing git package client
}

// GenerateMessage creates a conventional commit message
func (g *CommitMessageGenerator) GenerateMessage(
    ctx context.Context, 
    task string,
    files []string,
    workspaceDir string,
) (string, error) {
    
    // Get git diff
    diff, err := getDiff(workspaceDir, files)
    if err != nil {
        return "", fmt.Errorf("failed to get diff: %w", err)
    }
    
    // If LLM client available, generate smart message
    if g.llmClient != nil {
        prompt := buildCommitPrompt(task, diff, files)
        message, err := g.llmClient.Generate(ctx, prompt)
        if err != nil {
            // Fall back to simple message on LLM error
            return g.generateSimpleMessage(task, files), nil
        }
        return strings.TrimSpace(message), nil
    }
    
    // Simple message without LLM
    return g.generateSimpleMessage(task, files), nil
}

func (g *CommitMessageGenerator) generateSimpleMessage(task string, files []string) string {
    return fmt.Sprintf("chore: %s\n\nAutomated by Forge AI\nFiles modified: %d", 
        task, len(files))
}

func buildCommitPrompt(task string, diff string, files []string) string {
    var sb strings.Builder
    
    sb.WriteString("Generate a conventional commit message for these autonomous changes.\n\n")
    sb.WriteString("Format: <type>(<scope>): <description>\n")
    sb.WriteString("Types: feat, fix, docs, style, refactor, test, chore\n\n")
    
    sb.WriteString(fmt.Sprintf("Task: %s\n\n", task))
    
    sb.WriteString("Files changed:\n")
    for _, file := range files {
        sb.WriteString(fmt.Sprintf("- %s\n", file))
    }
    
    sb.WriteString("\nDiff:\n")
    sb.WriteString(truncateDiff(diff, 2000))
    
    sb.WriteString("\n\nGenerate ONLY the commit message (one line), nothing else.")
    
    return sb.String()
}
```

### Configuration

#### YAML Configuration

```yaml
git:
  # Auto-commit configuration
  auto_commit: true
  
  # Commit message
  commit_message: "" # Empty = auto-generate from task
  
  # Branch configuration
  branch: "forge/auto-fixes-{{.RunID}}" # Template syntax
  create_branch: true # Create branch before committing
  
  # Attribution
  author_name: "Forge AI"
  author_email: "forge-bot@example.com"
  
  # Push configuration (v1.0: always false)
  auto_push: false # Not implemented in v1.0
```

#### Template Variables

Branch name templates support variables:
- `{{.RunID}}`: CI/CD run ID (from env: GITHUB_RUN_ID, CI_JOB_ID, etc.)
- `{{.Date}}`: Current date (YYYY-MM-DD)
- `{{.Timestamp}}`: Unix timestamp
- `{{.Task}}`: Slugified task name

Example branch names:
- `forge/auto-fixes-{{.RunID}}` → `forge/auto-fixes-1234567`
- `forge/{{.Date}}-cleanup` → `forge/2025-01-21-cleanup`
- `automated/{{.Task}}` → `automated/fix-linting-errors`

### Integration with HeadlessExecutor

```go
func (e *HeadlessExecutor) Run(ctx context.Context) error {
    // 1. Validate workspace state
    if err := e.validateWorkspace(); err != nil {
        return err
    }
    
    // 2. Create snapshot for rollback
    snapshot, err := e.gitManager.CreateSnapshot()
    if err != nil {
        return fmt.Errorf("failed to create snapshot: %w", err)
    }
    e.snapshot = snapshot
    
    // 3. Create branch if configured
    if e.config.Git.CreateBranch && e.config.Git.Branch != "" {
        branchName := e.resolveBranchTemplate(e.config.Git.Branch)
        if err := e.gitManager.CreateBranch(branchName); err != nil {
            return fmt.Errorf("failed to create branch: %w", err)
        }
    }
    
    // 4. Execute task
    if err := e.executeTask(ctx); err != nil {
        e.gitManager.RestoreSnapshot(snapshot)
        return err
    }
    
    // 5. Run quality gates
    if !e.runQualityGates(ctx) {
        e.gitManager.RestoreSnapshot(snapshot)
        return fmt.Errorf("quality gates failed")
    }
    
    // 6. Auto-commit if configured
    if e.config.Git.AutoCommit {
        if err := e.performAutoCommit(ctx); err != nil {
            e.gitManager.RestoreSnapshot(snapshot)
            return fmt.Errorf("auto-commit failed: %w", err)
        }
    }
    
    return nil
}

func (e *HeadlessExecutor) performAutoCommit(ctx context.Context) error {
    // Get modified files
    status, err := e.gitManager.GetStatus()
    if err != nil {
        return err
    }
    
    modifiedFiles := append(status.Unstaged, status.Untracked...)
    if len(modifiedFiles) == 0 {
        e.logger.Info("No files modified, skipping commit")
        return nil
    }
    
    // Stage files
    if err := e.gitManager.StageFiles(modifiedFiles); err != nil {
        return err
    }
    
    // Generate commit message
    var message string
    if e.config.Git.CommitMessage != "" {
        message = e.config.Git.CommitMessage
    } else {
        generator := git.NewCommitMessageGenerator(e.llmClient)
        message, err = generator.GenerateMessage(ctx, e.config.Task, modifiedFiles, e.workspaceDir)
        if err != nil {
            return fmt.Errorf("failed to generate commit message: %w", err)
        }
    }
    
    // Add execution metadata to commit message
    message = e.enhanceCommitMessage(message)
    
    // Create commit
    author := &git.GitAuthor{
        Name:  e.config.Git.AuthorName,
        Email: e.config.Git.AuthorEmail,
    }
    
    if err := e.gitManager.Commit(message, author); err != nil {
        return err
    }
    
    e.logger.Info("Changes committed successfully", "branch", e.currentBranch)
    return nil
}

func (e *HeadlessExecutor) enhanceCommitMessage(message string) string {
    var sb strings.Builder
    sb.WriteString(message)
    sb.WriteString("\n\n")
    sb.WriteString(fmt.Sprintf("Execution ID: %s\n", e.executionID))
    sb.WriteString(fmt.Sprintf("Task: %s\n", e.config.Task))
    sb.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339)))
    
    // Add CI/CD metadata if available
    if runID := os.Getenv("GITHUB_RUN_ID"); runID != "" {
        sb.WriteString(fmt.Sprintf("GitHub Run: %s\n", runID))
    }
    if jobID := os.Getenv("CI_JOB_ID"); jobID != "" {
        sb.WriteString(fmt.Sprintf("GitLab Job: %s\n", jobID))
    }
    
    return sb.String()
}
```

### Workspace Validation

```go
func (e *HeadlessExecutor) validateWorkspace() error {
    // Check if workspace is a git repository
    if !e.gitManager.IsGitRepository() {
        return fmt.Errorf("workspace is not a git repository")
    }
    
    // Check workspace cleanliness
    clean, err := e.gitManager.IsClean()
    if err != nil {
        return fmt.Errorf("failed to check workspace status: %w", err)
    }
    
    if !clean && !e.config.AllowDirtyWorkspace {
        return fmt.Errorf("workspace has uncommitted changes (use --allow-dirty to override)")
    }
    
    // Check for detached HEAD
    branch, err := e.gitManager.GetCurrentBranch()
    if err != nil {
        return fmt.Errorf("detached HEAD state not supported: %w", err)
    }
    
    e.currentBranch = branch
    return nil
}
```

---

## Validation

### Success Metrics

- Branch creation successful in 100% of runs
- Commits properly attributed to Forge author
- Commit messages informative and follow conventions
- Rollback successful after any git operation failure
- Works in shallow clones (common in CI/CD)
- Works with various authentication methods

### Test Scenarios

1. **Clean Workspace**: Create branch, modify files, commit
2. **Dirty Workspace**: Detect uncommitted changes, fail or warn
3. **Detached HEAD**: Detect detached HEAD, fail with clear error
4. **Shallow Clone**: Works with shallow clone (depth=1)
5. **Branch Exists**: Handle existing branch name gracefully
6. **Commit Message Generation**: LLM generates conventional commit message
7. **Rollback After Failure**: Restore snapshot on quality gate failure

---

## Related Decisions

- [ADR-0026](0026-headless-mode-architecture.md) - Headless mode architecture
- [ADR-0027](0027-safety-constraint-system.md) - Safety constraints (rollback uses git)
- [ADR-0028](0028-quality-gate-architecture.md) - Quality gates (gates run before commit)

---

## References

- [Headless CI/CD Mode PRD](../product/features/headless-ci-mode.md)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GitHub Actions: Checkout](https://github.com/actions/checkout)
- [GitLab CI: Git Strategy](https://docs.gitlab.com/ee/ci/runners/configure_runners.html#git-strategy)

---

## Notes

Git integration is designed with safety as the primary concern. In v1.0:
- **No automatic push**: Changes stay local for manual review
- **Branch isolation**: Changes on feature branch, not main/master
- **Clear attribution**: Forge identified as author

Future enhancements (P2):
- Automatic push to remote
- PR creation via GitHub/GitLab APIs
- Conflict detection and resolution strategies
- Multi-repo support

**Last Updated:** 2025-01-21
