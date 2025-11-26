# ADR-0031: Headless Git PR Creation

**Status:** Proposed  
**Date:** 2024-01-XX  
**Authors:** Forge Team  
**Related:** [ADR-0029: Headless Git Integration](0029-headless-git-integration.md), ADR-0030 (Automated PR Documentation Workflow)

## Context

ADR-0029 established headless mode with git commit capabilities for CI/CD workflows. However, many automated workflows require creating pull requests rather than direct commits:

- **Automated Documentation Updates**: Generate and submit docs changes as PRs for review
- **Dependency Updates**: Create PRs for package upgrades that require human approval  
- **Code Migrations**: Submit refactoring changes as reviewable PRs
- **Security Patches**: Create PRs for vulnerability fixes with automated analysis

Direct commits bypass code review processes. We need to enable headless mode to create pull requests while maintaining the safety and review standards of collaborative development.

**Key Insight:** Forge already has PR creation working in TUI mode via the `/pr` command, which uses `gh` CLI. We should reuse this proven implementation rather than building new GitHub API integration.

## Requirements

### Functional Requirements

1. **PR Creation from Headless Mode**: Enable headless executor to create GitHub pull requests
2. **Configurable PR Metadata**: Support custom PR titles, descriptions, base/head branches
3. **Code Reuse**: Leverage existing `git.CreatePR()` implementation from TUI mode
4. **GitHub Actions Compatibility**: Work seamlessly in GitHub Actions (where `gh` CLI is pre-installed)
5. **Graceful Degradation**: Fall back to direct commits if PR creation fails

### Non-Functional Requirements

1. **Minimal Dependencies**: Reuse existing `gh` CLI, no new SDKs or HTTP clients
2. **Security**: Require `GITHUB_TOKEN` for authentication, fail safely without it
3. **Observability**: Clear logging of PR creation status and URLs
4. **Maintainability**: Simple configuration schema matching existing git config pattern

## Options Considered

### Option 1: GitHub CLI (gh) Wrapper ✅ CHOSEN

**Description:** Extend headless `GitConfig` to support PR creation using existing `git.CreatePR()` function that wraps `gh` CLI.

**Pros:**
- **Code Reuse**: Leverages existing, proven implementation from `/pr` command
- **Zero New Dependencies**: `gh` CLI already available in GitHub Actions
- **Simple Configuration**: Just add PR fields to existing GitConfig schema
- **Proven Solution**: Already works reliably in TUI mode
- **Easy Testing**: Can test using same mocks as TUI mode

**Cons:**
- **GitHub-Only**: Limited to GitHub (no GitLab/Bitbucket support initially)
- **CLI Dependency**: Requires `gh` CLI installed (but this is default in GitHub Actions)

### Option 2: Direct GitHub API via HTTP Client

**Description:** Implement GitHub REST API client for PR creation.

**Pros:**
- No CLI dependency
- Full control over API calls

**Cons:**
- **Reinventing the Wheel**: We already have working PR creation via `gh`
- **Maintenance Burden**: Must track GitHub API changes ourselves
- **More Code**: 200+ lines vs. configuration-only change
- **Testing Complexity**: Need to mock HTTP responses

### Option 3: Go GitHub SDK (google/go-github)

**Description:** Use official Go SDK for GitHub API operations.

**Pros:**
- Type-safe operations
- Battle-tested library

**Cons:**
- **Unnecessary Dependency**: We only need PR creation, which `gh` already does
- **Overkill**: Full SDK for one operation we already have working
- **Future Limitation**: Still GitHub-only, but with heavier dependency

## Decision

**Chosen Option:** Option 1 - GitHub CLI (gh) Wrapper

### Rationale

1. **Code Reuse Over Reinvention**: The `git.CreatePR()` function in `pkg/agent/git/pr.go` already implements PR creation via `gh` CLI. It's proven, tested, and working in production (TUI `/pr` command).

2. **Zero New Code**: This is primarily a **configuration change**. We just wire the existing PR creation logic into headless mode via `GitConfig` extensions.

3. **GitHub Actions Native**: GitHub Actions includes `gh` CLI by default. No installation needed, no binary dependencies to manage.

4. **Simplicity**: The entire implementation is ~50 lines of configuration parsing and function calls, vs. 200+ lines for API client implementations.

5. **Future Flexibility**: If we need GitLab/Bitbucket later, we can add provider-specific adapters. For now, GitHub coverage is sufficient.

6. **Proven Pattern**: We already use this pattern successfully for commits. Extending it to PRs is natural and consistent.

### Consequences

#### Positive

- **Minimal Implementation**: Just extend GitConfig schema and wire existing function
- **Zero New Dependencies**: Reuses `gh` CLI already in GitHub Actions
- **Proven Reliability**: Same code path as successful `/pr` command
- **Easy Testing**: Can reuse existing test infrastructure
- **Consistent UX**: Headless PR creation works exactly like TUI `/pr`

#### Negative

- **GitHub-Only**: Limited to GitHub initially (acceptable for MVP)
- **CLI Dependency**: Requires `gh` CLI (but this is standard in CI/CD)

#### Neutral

- **Configuration-Driven**: PR creation controlled by GitConfig, not code
- **Optional Feature**: Falls back to direct commits if not configured
- **Token Required**: Needs `GITHUB_TOKEN` for authentication

## Implementation Details

### Configuration Schema Extension

Extend the existing `GitConfig` from ADR-0029 with PR configuration:

```go
// GitConfig defines git operation configuration (from ADR-0029)
type GitConfig struct {
    Enabled       bool   `yaml:"enabled"`
    CommitMessage string `yaml:"commit_message"`
    Branch        string `yaml:"branch"`
    
    // NEW: PR creation configuration
    CreatePR      bool   `yaml:"create_pr"`       // If true, create PR instead of direct commit
    PRTitle       string `yaml:"pr_title"`        // PR title (optional, auto-generated if empty)
    PRBody        string `yaml:"pr_body"`         // PR description (optional, auto-generated if empty)
    PRBase        string `yaml:"pr_base"`         // Target branch (default: "main")
    PRDraft       bool   `yaml:"pr_draft"`        // Create as draft PR
}
```

**Example headless configuration:**

```yaml
# headless-config.yaml
task: "Update API documentation based on recent code changes"
workspace_dir: "/workspace"

git:
  enabled: true
  branch: "docs/api-update"
  
  # PR configuration (NEW)
  create_pr: true
  pr_base: "main"
  pr_draft: false
```

### Headless Executor Integration

The headless executor extends its existing git integration to support PR creation:

```go
// In pkg/executor/headless/executor.go

func (e *Executor) handleGitOperations(ctx context.Context) error {
    if !e.config.Git.Enabled {
        return nil
    }
    
    // 1. Create and checkout branch (existing logic from ADR-0029)
    if err := e.gitManager.CreateAndCheckoutBranch(e.config.Git.Branch); err != nil {
        return fmt.Errorf("failed to create branch: %w", err)
    }
    
    // 2. Stage and commit changes (existing logic from ADR-0029)
    if err := e.gitManager.StageAll(); err != nil {
        return fmt.Errorf("failed to stage changes: %w", err)
    }
    
    commitMsg := e.config.Git.CommitMessage
    if commitMsg == "" {
        commitMsg = "chore: automated changes from forge"
    }
    
    if err := e.gitManager.Commit(commitMsg); err != nil {
        return fmt.Errorf("failed to commit: %w", err)
    }
    
    // 3. Create PR or push directly (NEW)
    if e.config.Git.CreatePR {
        return e.createPullRequest(ctx)
    }
    
    // Fallback to direct push if PR creation not enabled
    return e.gitManager.Push()
}

func (e *Executor) createPullRequest(ctx context.Context) error {
    cfg := e.config.Git
    
    // Detect base branch if not specified
    base := cfg.PRBase
    if base == "" {
        detectedBase, err := git.DetectBaseBranch(e.config.WorkspaceDir)
        if err != nil {
            e.logger.Warn("Could not detect base branch, using 'main'")
            base = "main"
        } else {
            base = detectedBase
        }
    }
    
    head := cfg.Branch
    
    // Generate PR title and description using LLM if not provided
    title := cfg.PRTitle
    body := cfg.PRBody
    
    if title == "" || body == "" {
        prContent, err := e.generatePRContent(ctx, base, head)
        if err != nil {
            e.logger.Warn("Failed to generate PR content, using defaults: %v", err)
            if title == "" {
                title = fmt.Sprintf("chore: automated changes in %s", head)
            }
            if body == "" {
                body = "Automated changes generated by Forge."
            }
        } else {
            if title == "" {
                title = prContent.Title
            }
            if body == "" {
                body = prContent.Description
            }
        }
    }
    
    // Use existing git.CreatePR() function (from pkg/agent/git/pr.go)
    prURL, err := git.CreatePR(e.config.WorkspaceDir, title, body, base, head)
    if err != nil {
        return fmt.Errorf("failed to create pull request: %w", err)
    }
    
    e.logger.Info("✅ Pull request created: %s", prURL)
    return nil
}

func (e *Executor) generatePRContent(ctx context.Context, base, head string) (*git.PRContent, error) {
    // Get commits since base branch
    commits, err := git.GetCommitsSinceBase(e.config.WorkspaceDir, base, head)
    if err != nil {
        return nil, fmt.Errorf("failed to get commits: %w", err)
    }
    
    // Get diff summary
    diffSummary, err := git.GetDiffSummary(e.config.WorkspaceDir, base, head)
    if err != nil {
        return nil, fmt.Errorf("failed to get diff: %w", err)
    }
    
    // Use existing PRGenerator (from pkg/agent/git/pr.go)
    generator := git.NewPRGenerator(e.llmClient)
    return generator.Generate(ctx, commits, diffSummary, base, head, "")
}
```

### Workflow Examples

#### Example 1: Automated Documentation PR

```yaml
# .github/workflows/docs-update.yml
name: Update Documentation

on:
  schedule:
    - cron: '0 2 * * 1'  # Weekly on Monday 2 AM

jobs:
  update-docs:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Run Forge Documentation Update
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          forge -headless -headless-config <<EOF
          task: "Review all Go files and update documentation comments for any functions missing proper godoc"
          workspace_dir: "."
          git:
            enabled: true
            branch: "docs/weekly-update"
            create_pr: true
            pr_base: "main"
            pr_draft: false
          EOF
```

#### Example 2: Direct Commit (Existing Behavior)

```yaml
# Still works exactly as before from ADR-0029
task: "Fix linting errors"
workspace_dir: "."
git:
  enabled: true
  commit_message: "chore: fix linting errors"
  branch: "main"
  # create_pr: false (default, commits directly)
```

#### Example 3: Security Patch PR

```yaml
task: "Update all dependencies with known vulnerabilities"
workspace_dir: "."
git:
  enabled: true
  branch: "security/dep-updates"
  create_pr: true
  pr_base: "main"
  pr_title: "🔒 Security: Update vulnerable dependencies"
  pr_body: |
    ## Security Updates
    
    This PR updates dependencies with known security vulnerabilities.
    
    **Please review carefully before merging.**
  pr_draft: false
```

## Security Considerations

### Token Management

1. **GITHUB_TOKEN Required**: PR creation requires `GITHUB_TOKEN` environment variable
2. **Token Permissions**: Must have `contents: write` and `pull-requests: write` permissions
3. **Validation**: Executor validates token exists before attempting PR creation
4. **Error Handling**: Falls back to direct commit if PR creation fails (optional, configurable)

```go
func (e *Executor) createPullRequest(ctx context.Context) error {
    // Validate GITHUB_TOKEN exists
    if os.Getenv("GITHUB_TOKEN") == "" {
        if e.config.Git.RequirePR {
            return fmt.Errorf("GITHUB_TOKEN required for PR creation")
        }
        e.logger.Warn("GITHUB_TOKEN not found, falling back to direct push")
        return e.gitManager.Push()
    }
    
    // ... PR creation logic
}
```

### Safety Constraints

1. **Branch Protection**: Respects GitHub branch protection rules
2. **No Force Push**: PR creation never force-pushes
3. **Audit Trail**: All PR creations logged with metadata
4. **Rate Limiting**: `gh` CLI handles GitHub API rate limits automatically

## Testing Strategy

### Unit Tests

```go
func TestHeadlessExecutor_CreatePR(t *testing.T) {
    tests := []struct {
        name        string
        config      *Config
        setupMock   func(*git.MockManager, *git.MockPRGenerator)
        wantErr     bool
        wantPRURL   string
    }{
        {
            name: "successful PR creation",
            config: &Config{
                Git: GitConfig{
                    Enabled:   true,
                    Branch:    "test-branch",
                    CreatePR:  true,
                    PRBase:    "main",
                },
            },
            setupMock: func(gm *git.MockManager, pg *git.MockPRGenerator) {
                // Mock PR generation
                pg.On("Generate", mock.Anything, mock.Anything, mock.Anything, "main", "test-branch", "").
                    Return(&git.PRContent{
                        Title:       "Test PR",
                        Description: "Test description",
                    }, nil)
                
                // Mock PR creation (uses existing git.CreatePR)
                gm.On("CreatePR", mock.Anything, "Test PR", "Test description", "main", "test-branch").
                    Return("https://github.com/owner/repo/pull/1", nil)
            },
            wantErr:   false,
            wantPRURL: "https://github.com/owner/repo/pull/1",
        },
        {
            name: "fallback to direct commit when token missing",
            config: &Config{
                Git: GitConfig{
                    Enabled:  true,
                    Branch:   "test-branch",
                    CreatePR: true,
                },
            },
            setupMock: func(gm *git.MockManager, pg *git.MockPRGenerator) {
                // No GITHUB_TOKEN, should fall back to push
                gm.On("Push").Return(nil)
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation reuses existing git package mocks
        })
    }
}
```

### Integration Tests

1. **GitHub Actions Test Workflow**: Actual PR creation in test repository
2. **Token Validation**: Ensure proper error handling without token
3. **Branch Detection**: Verify base branch auto-detection logic
4. **LLM Generation**: Test PR title/description generation with mock LLM

## Migration Path

### For Existing Users

**No Breaking Changes**: This is purely additive. Existing headless configurations work unchanged.

```yaml
# Existing config (still works)
task: "Fix bugs"
git:
  enabled: true
  commit_message: "fix: bug fixes"
  branch: "main"

# New config (opt-in to PR creation)
task: "Fix bugs"
git:
  enabled: true
  branch: "bugfix/automated"
  create_pr: true  # <-- New field
```

### For GitHub Actions

**Minimal Changes**: Just add `pull-requests: write` permission if creating PRs.

```yaml
# Before (ADR-0029)
permissions:
  contents: write

# After (ADR-0031, if using create_pr)
permissions:
  contents: write
  pull-requests: write  # <-- Add this
```

## Success Metrics

1. **Reliability**: 99%+ PR creation success rate in GitHub Actions
2. **Adoption**: Used in at least 3 automated workflows within first month
3. **Code Reuse**: Zero new PR creation code (100% reuse of existing `git.CreatePR()`)
4. **Performance**: PR creation completes in <10 seconds
5. **Error Handling**: Graceful degradation when `gh` unavailable or token missing

## Related Decisions

- **ADR-0029**: Established headless git integration foundation
- **ADR-0030** (Next): Will use this PR creation capability for automated documentation workflow

## References

- [GitHub CLI Documentation](https://cli.github.com/manual/gh_pr_create)
- [GitHub Actions Permissions](https://docs.github.com/en/actions/security-guides/automatic-token-authentication#permissions-for-the-github_token)
- [Existing Implementation](../../pkg/agent/git/pr.go): `git.CreatePR()` function

## Future Enhancements

1. **GitLab Support**: Add adapter for GitLab MR creation via `glab` CLI
2. **PR Templates**: Support repository-specific PR templates
3. **Auto-Merge**: Conditionally auto-merge PRs based on CI status
4. **PR Comments**: Post additional context as PR comments (needed for ADR-0030)
5. **Multi-Platform**: Abstract PR creation behind interface for platform independence

## Design Considerations

### Why Not Build GitHub API Client?

We already have a working solution via `gh` CLI that:
- Is proven in production (`/pr` command)
- Requires zero new code
- Is available by default in GitHub Actions
- Handles authentication, retries, and errors

Building an API client would be **reinventing the wheel** for no practical benefit.

### Why Not Use Go GitHub SDK?

The SDK is excellent, but we only need PR creation. Using the SDK would:
- Add a dependency we don't need
- Require 200+ lines of integration code
- Duplicate functionality we already have working
- Still be GitHub-only (no multi-platform benefit)

The simpler approach is to reuse existing code.

### Extensibility for Other Platforms

If we need GitLab/Bitbucket later, we can:
1. Abstract `git.CreatePR()` behind an interface
2. Add platform-specific adapters (GitHubAdapter, GitLabAdapter)
3. Detect platform from git remote URL
4. Delegate to appropriate adapter

This can be added without changing the configuration schema or headless executor logic.
