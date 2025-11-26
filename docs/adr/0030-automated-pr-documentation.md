# ADR-0030: Automated PR Documentation Workflow

**Status:** Proposed  
**Date:** 2024-11-26  
**Authors:** Forge Team  
**Related:** [ADR-0029: Headless Git Integration](0029-headless-git-integration.md), [ADR-0031: Headless Git PR Creation](0031-headless-git-pr-creation.md)

## Context

Documentation tends to lag behind code changes in fast-moving projects. The PRD for this feature ([Automated PR Documentation](../product/features/automated-pr-documentation.md)) identifies the core problem:

**When developers create PRs:**
- Code changes are made without corresponding documentation updates
- Reviewers must request documentation updates, causing review delays
- Documentation often gets merged incomplete or outdated
- Knowledge about the code changes is fresh but documentation is deferred

**The PRD Solution:** Trigger Forge in headless mode when a PR is opened to automatically:
1. Analyze the code changes in the PR
2. Determine what documentation needs updating
3. Generate appropriate documentation updates
4. Commit changes directly to the PR branch (or create separate docs PR)
5. Post summary comment on the PR with links to changes

This ensures documentation ships **with** the code, not after it.

## Requirements

Per the PRD, this feature has specific requirements that differ from a scheduled documentation workflow:

### Functional Requirements

1. **PR-Triggered Execution**: Runs when PR is opened (not scheduled, not on updates)
2. **PR Context Integration**: Extracts PR metadata (title, description, changed files, labels) and templates into task
3. **Documentation Analysis**: Analyzes code changes to determine documentation impact
4. **Intelligent Generation**: Uses LLM to generate contextually appropriate documentation
5. **Direct Commit**: Commits documentation changes to the PR branch itself (default behavior)
6. **PR Comment**: Posts comprehensive summary comment with links to all changes
7. **Safety Constraints**: ONLY modifies markdown files in docs/ directory - no code changes

### Non-Functional Requirements

1. **Safety**: Never auto-merge, never modify code files (*.go, *.py, etc.)
2. **Quality**: Generated docs should be accurate and follow project conventions
3. **Efficiency**: Complete analysis and commit in <5 minutes
4. **Non-Blocking**: Workflow never fails the PR (informational only)
5. **Cost Control**: Limit LLM token usage with file/line constraints

## Configuration Options

The PRD requires supporting BOTH direct commit and separate PR workflows. Teams choose based on their trust level and workflow preferences.

### Mode 1: Direct Commit to PR Branch (Default)

**Configuration:**
```yaml
git:
  auto_commit: true
  auto_push: true
  create_pr: false  # Don't create separate PR
```

**Behavior:**
- Forge commits documentation changes directly to the PR branch
- Developer sees docs in same diff as code changes
- Single review and merge workflow

**When to Use:**
- Team trusts AI-generated documentation
- Want simplest workflow (one PR to review)
- Prefer seeing code and docs together
- Comfortable with AI commits on feature branches

**User Journey:**
1. Developer opens PR
2. Forge analyzes and commits docs to PR branch
3. Developer reviews code + docs in single PR
4. Developer merges when satisfied

### Mode 2: Separate Documentation PR (Trust-Building)

**Configuration:**
```yaml
git:
  auto_commit: true
  auto_push: true
  create_pr: true       # Create separate PR
  pr_base: "{{ pr_branch }}"  # Target the original PR branch
  pr_title: "docs: Documentation for PR #{{ pr_number }}"
```

**Behavior:**
- Forge creates a NEW PR targeting the original PR branch
- Documentation changes isolated from code changes
- Two-step review process

**When to Use:**
- Team new to AI documentation
- Want to review docs separately before accepting
- Prefer clean separation of code vs. docs commits
- Need to reject docs without affecting code PR

**User Journey:**
1. Developer opens PR #123
2. Forge creates PR #124 targeting PR #123's branch
3. Developer reviews documentation PR independently
4. Developer merges docs PR into code PR when satisfied
5. Developer merges code PR (now including docs) into main

**Implementation:** See [ADR-0031: Headless Git PR Creation](0031-headless-git-pr-creation.md) for the technical implementation of PR creation in headless mode, including GitHub CLI (`gh`) integration, authentication requirements, and configuration schema.
3. Developer reviews docs PR #124 first
4. Developer merges docs PR #124 into PR #123
5. Developer reviews combined PR #123
6. Developer merges PR #123 to main

### Trust Progression Path

**Week 1-2: Separate PR Mode**
- Team reviews AI docs in isolation
- Builds confidence in quality
- Can reject docs without impacting code

**Week 3-4: Transition Phase**
- Team notices high acceptance rate (>80%)
- Separate PR becomes overhead
- Starts to trust AI judgment

**Week 5+: Direct Commit Mode**
- Team switches to direct commit
- Reviews code + docs together
- Faster workflow, less overhead

This two-mode approach is ESSENTIAL for adoption - teams won't trust direct commits initially.

## Decision

**Chosen Approach:** Support BOTH direct commit and separate PR modes with configuration

### Rationale

1. **Trust-Building Essential**: Teams won't adopt if forced to accept direct commits immediately. Separate PR mode allows teams to build trust gradually.

2. **PRD Requirement**: The PRD explicitly requires both modes:
   - "Git configuration controls behavior (commit to branch vs. create new PR)" (line 89)
   - Mode 1: "Direct to PR branch: auto_commit: true, auto_push: true" (line 260)
   - Mode 2: "Separate docs PR: auto_pr: true - creates new PR targeting original PR branch" (line 262)

3. **Adoption Path**: Teams need a safe starting point:
   - Start with separate PR mode (review docs in isolation)
   - Build confidence over 2-4 weeks
   - Switch to direct commit mode (faster workflow)

4. **Flexibility**: Different teams have different workflows:
   - Some want clean commit history (separate PR)
   - Others prefer simplicity (direct commit)
   - Configuration lets each team choose

5. **Risk Management**: Separate PR mode provides safety net:
   - Can reject bad docs without affecting code PR
   - Easy rollback if AI makes mistakes
   - Builds organizational trust in AI

**Default Configuration:** Direct commit mode (simpler for new users who trust it), but documentation clearly shows how to enable separate PR mode for teams that need it.

### Consequences

#### Positive

- **Zero Documentation Lag**: Documentation updated immediately when PR created
- **Complete Context**: PR description, changed files, and code diff all available to LLM
- **Single Review Cycle**: Code and docs reviewed together, no separate PR overhead
- **Fresh Knowledge**: Documentation written while developer's intent is fresh
- **Automated Safety**: Constraints ensure only markdown files modified
- **Non-Blocking**: Workflow never fails PR, only enhances it
- **Clear Visibility**: PR comment shows exactly what was documented

#### Negative

- **Commits to Feature Branch**: Adds commits to developer's PR branch (may be unexpected)
- **Single Trigger**: Only runs on PR open, not on subsequent code updates
- **LLM Cost Per PR**: Each PR incurs API cost (mitigated by constraints)
- **Potential Noise**: Could add commits to PRs where no docs needed (mitigated by smart detection)

#### Neutral

- **Manual Re-trigger**: Developer can request re-run via PR comment if needed
- **Configuration Optional**: Works with sensible defaults, customization available
- **Git Strategy Configurable**: Can switch to separate docs PR if preferred (via auto_pr config)

## Implementation Details

### GitHub Actions Workflow

Per the PRD, the workflow triggers on PR open and extracts PR context:

```yaml
# .github/workflows/docs-automation.yml
name: Automated Documentation Updates

on:
  pull_request:
    types: [opened]  # Only on PR creation, not updates
  
  workflow_dispatch:  # Allow manual re-trigger
    inputs:
      pr_number:
        description: 'PR number to document'
        required: true

jobs:
  update-docs:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    
    steps:
      - name: Checkout PR Branch
        uses: actions/checkout@v4
        with:
          ref: ${{ github.event.pull_request.head.ref }}
          fetch-depth: 0
      
      - name: Install Forge
        run: |
          # Install forge-headless binary
          curl -L https://github.com/entrhq/forge/releases/latest/download/forge-linux-amd64 -o /usr/local/bin/forge
          chmod +x /usr/local/bin/forge
      
      - name: Extract PR Context
        id: pr_context
        run: |
          # Get changed files
          CHANGED_FILES=$(git diff --name-only origin/${{ github.event.pull_request.base.ref }}...HEAD)
          
          # Save to environment for next step
          echo "CHANGED_FILES<<EOF" >> $GITHUB_ENV
          echo "$CHANGED_FILES" >> $GITHUB_ENV
          echo "EOF" >> $GITHUB_ENV
      
      - name: Generate Forge Config
        run: |
          cat > /tmp/forge-config.yml <<'EOF'
          task: |
            Review and update all documentation for this PR.
            
            PR #${{ github.event.pull_request.number }}: ${{ github.event.pull_request.title }}
            
            PR Description:
            ${{ github.event.pull_request.body }}
            
            Changed Files:
            ${{ env.CHANGED_FILES }}
            
            Instructions:
            - Analyze the code changes and PR description for context
            - Determine what documentation needs updating (ADR, API docs, README, guides)
            - Update existing documentation where appropriate
            - Create new documentation only when genuinely needed
            - Follow existing documentation style and conventions
            - Be thorough but concise
            
            Safety Rules:
            - ONLY modify markdown (.md) files in docs/ directory and root-level docs
            - NEVER modify code files (*.go, *.py, *.js, *.ts, etc.)
            - Preserve existing documentation structure
          
          workspace_dir: ${{ github.workspace }}
          
          mode: write
          
          constraints:
            max_files: 10
            max_lines_changed: 1000
            allowed_patterns:
              - "docs/**/*.md"
              - "README.md"
              - "CHANGELOG.md"
            denied_patterns:
              - "**/*.go"
              - "**/*.py"
              - "**/*.js"
              - "**/*.ts"
              - "**/*.java"
              - "**/*.c"
              - "**/*.cpp"
            allowed_tools:
              - read_file
              - write_file
              - apply_diff
              - search_files
              - list_files
            max_tokens: 100000
            timeout: 5m
          
          quality_gates:
            - name: markdown-lint
              command: markdownlint docs/ || true
              required: false
            - name: link-check
              command: markdown-link-check docs/**/*.md || true
              required: false
          
          git:
            auto_commit: true
            auto_push: true
            commit_message: "docs: auto-update documentation for PR #${{ github.event.pull_request.number }}"
            author_name: "Forge AI"
            author_email: "forge-ai@github-actions"
          
          artifacts:
            enabled: true
            output_dir: /tmp/forge-artifacts
            json: true
            markdown: true
            metrics: true
          EOF
      
      - name: Run Forge Documentation Update
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          forge -headless -headless-config /tmp/forge-config.yml
      
      - name: Post PR Comment
        if: always()
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            
            // Read artifacts if available
            let summary = 'Documentation analysis complete.';
            try {
              const artifact = JSON.parse(fs.readFileSync('/tmp/forge-artifacts/summary.json', 'utf8'));
              summary = artifact.summary || summary;
            } catch (e) {
              console.log('No artifacts found, using default summary');
            }
            
            // Post comment
            await github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              body: `## 📚 Documentation Updated by Forge\n\n${summary}\n\n---\n*Need changes? Edit the docs directly in this PR.*`
            });
      
      - name: Cleanup
        if: always()
        run: rm -f /tmp/forge-config.yml
```

### Documentation Intelligence Strategy

The task prompt is dynamically generated with PR context to guide intelligent documentation:

#### 1. **PR Context Integration**

The workflow extracts and templates PR metadata into the task:
- **PR Title & Number**: Provides high-level intent
- **PR Description**: Developer's explanation of changes (critical context)
- **Changed Files**: List of modified files from git diff
- **Labels**: PR labels (e.g., "feature", "bug", "breaking") guide doc type

This matches PRD requirement:
> "PR metadata integration: PR title and description content, PR labels and change type indicators, Changed file paths and diff information" (lines 98-101)

#### 2. **Documentation Impact Analysis**

The LLM determines impact based on:
- **File Patterns**: API files → API docs, config files → config docs
- **Change Type**: New files → create docs, modified files → update docs
- **PR Labels**: "breaking" → CHANGELOG + migration guide
- **Commit Messages**: Conventional commits hint at doc needs

#### 3. **Quality Standards**

Documentation should:
- Follow existing conventions (detected by reading current docs)
- Match project style (tone, format, detail level)
- Include examples for new APIs or complex changes
- Cross-reference related documentation
- Update indexes/TOCs when structure changes

#### 4. **Safety Constraints**

Enforced via constraints config:
- **ONLY .md files**: `allowed_patterns: ["docs/**/*.md", "README.md", "CHANGELOG.md"]`
- **NEVER code files**: `denied_patterns: ["**/*.go", "**/*.py", etc.]`
- **Limited scope**: `max_files: 10, max_lines_changed: 1000`
- **Tool restrictions**: Only read/write/diff tools, no execute_command

This matches PRD requirement:
> "CRITICAL: Only modify markdown (.md) files - no code changes, no inline documentation" (line 121)

### Example Generated PR Comment

After Forge commits documentation to the PR branch, it posts a comment:

```markdown
## 📚 Documentation Updated by Forge

I've analyzed your PR and updated the following documentation:

### ✅ Updated Files
- **docs/adr/0031-headless-git-pr-creation.md** - Updated to reflect new auto_pr config option  
  [View changes](https://github.com/owner/repo/pull/123/files#diff-abc123)
- **README.md** - Added example of automated documentation workflow  
  [View changes](https://github.com/owner/repo/pull/123/files#diff-def456)

### ✨ Created Files
- **docs/guides/pr-documentation.md** - New guide for setting up automated PR documentation  
  [View file](https://github.com/owner/repo/blob/feature-branch/docs/guides/pr-documentation.md)

### 📊 Summary
- Files updated: 2
- Files created: 1
- Lines added: 87
- Confidence: High

<details>
<summary>📝 Key Changes Summary</summary>

#### ADR Update
Updated ADR-0031 to document the new `auto_pr` configuration option that enables automatic PR creation from headless mode. Added examples showing both direct commit and separate PR workflows.

#### README Enhancement
Added a new section demonstrating the automated documentation workflow in action, including the GitHub Actions configuration and expected output.

</details>

### 🔗 All Changes
[View commit](https://github.com/owner/repo/commit/abc123) | [Compare changes](https://github.com/owner/repo/pull/123/files)

---
*Need changes? Edit the docs directly in this PR, or comment "/forge update-docs" to re-run.*
```

This matches the PRD requirement for PR comments (lines 129-136).

### Alternative: Scheduled Documentation Review

**Note:** This is an ALTERNATIVE workflow for periodic documentation audits, separate from the PR-triggered workflow described above.

```yaml
# Scheduled documentation review task template (NOT the primary PR-triggered workflow)
task: |
  Analyze codebase for documentation gaps and generate improvements.
  
  **Analysis Scope:**
  {{#if scope_godoc}}
  - Go documentation (godoc comments)
  {{/if}}
  {{#if scope_readme}}
  - README and guide documentation
  {{/if}}
  {{#if scope_examples}}
  - Example code and usage
  {{/if}}
  
  **Focus Areas:**
  - Files changed since: {{ since }}
  - Exported symbols without documentation
  - Outdated documentation (code changed but docs didn't)
  
  **Quality Standards:**
  - Follow project's existing documentation style
  - Use clear, concise language
  - Include examples for complex APIs
  - Link to related documentation
  
  **Safety Rules:**
  - Only modify documentation (comments, README, examples)
  - Never change code logic
  - Don't document internal/private symbols
  - Preserve existing correct documentation

workspace_dir: "."

git:
  enabled: true
  branch: "docs/scheduled-review-{{ timestamp }}"
  create_pr: true
  pr_base: "main"
  pr_draft: false
```

This scheduled approach is different from the primary PR-triggered workflow and would be used for periodic documentation health checks rather than immediate PR documentation.

## Future Enhancements

These are potential future features beyond the initial PRD scope:

### 1. PR Update Trigger

Currently runs only on PR open. Future version could:
- Re-run on specific PR comment (e.g., `/forge update-docs`)
- Run on label addition (e.g., `needs-documentation`)
- Run on push to PR branch with `[docs]` commit message

**Trade-off:** More frequent runs = higher cost, but more comprehensive coverage.

### 2. Documentation Quality Metrics

Add structured metrics to PR comment showing documentation impact:

```markdown
## 📊 Documentation Metrics

**Coverage Impact:**
- ADR coverage: 12/15 decisions documented (80% → 100%)
- API endpoints: 45/50 documented (90% → 100%)

**Changes:**
- Files updated: 3
- New documentation: 2 files
- Lines added: 147

**Quality Gates:**
- ✅ Markdown lint passed
- ✅ All links valid
- ⚠️  1 TODO comment added (needs follow-up)
```

### 3. Multi-Language Documentation Support

Expand beyond markdown to support inline code comments and schemas:

**Languages:**
- **Go**: godoc comments (as shown below)
- **Python**: Docstrings
- **TypeScript**: TSDoc comments
- **Rust**: Rustdoc comments

**Schemas:**
- API schema documentation (OpenAPI, GraphQL)
- Database schema documentation

**Requires:** Enhanced safety constraints to prevent code modification.

```go
// Before (in codebase)
// Execute runs the task
func Execute() error { ... }

// After (AI-generated)
// Execute runs the configured task in headless mode and returns an error if execution fails.
// It performs the following steps:
// 1. Loads configuration from the specified config file
// 2. Initializes the LLM client with the provided API key
// 3. Creates a headless executor and runs the task
// 4. Optionally commits changes to git if configured
//
// Returns an error if any step fails.
func Execute() error { ... }
```

### 4. Documentation Templates

Pre-defined templates for common documentation patterns:
- **API endpoint documentation**: Request/response examples, error codes
- **CLI command documentation**: Usage, flags, examples
- **Configuration option documentation**: Valid values, defaults, examples

### 5. Interactive Review Mode

Allow reviewers to accept/reject suggestions individually:

```markdown
## Suggested Documentation Changes

### Change 1: Add godoc for `NewExecutor`
```diff
+// NewExecutor creates a new headless executor with the given configuration.
+// It validates the config and initializes the LLM client.
 func NewExecutor(cfg *Config) (*Executor, error) {
```

☐ Accept this change
☐ Reject this change
☐ Suggest improvement

[Comment on this suggestion]
```

### 6. Changelog Generation

Auto-generate CHANGELOG.md entries from git history and PR descriptions:

```markdown
## [Unreleased]

### Added
- Headless mode PR creation capability (#123)
- Documentation automation workflow (#124)

### Changed
- Improved git integration error handling (#125)
```

### 7. Documentation Search Index

Build searchable index of all documentation:
- Auto-generate documentation site
- Improve discoverability across projects
- Track documentation usage and gaps

### 8. AI Documentation Review

Have LLM review documentation PRs for quality:
- **Completeness**: All public APIs documented
- **Accuracy**: Docs match implementation
- **Clarity**: Easy to understand for target audience
- **Consistency**: Follows project style guide

## Cost Control

### Token Usage Limits

```yaml
# Cost control configuration
cost_control:
  max_tokens_per_run: 100000      # ~$0.10 at GPT-4 Turbo pricing
  max_files_per_analysis: 50      # Limit scope
  skip_large_files: true          # Skip files >1000 lines
  use_cheaper_model: true         # Use GPT-3.5-turbo for simple docs
```

### Incremental Analysis

Only analyze changed files:

```bash
# In GitHub Actions
git diff --name-only HEAD~7 HEAD | grep '\.go$' > changed_files.txt
```

Then configure Forge to only analyze those files.

### Estimated Costs

| Repository Activity | PRs/Month | Tokens/PR | Monthly Cost |
|---------------------|-----------|-----------|--------------|
| Low (5 PRs/month)   | 5         | 100,000   | ~$1.50       |
| Medium (20 PRs/month)| 20       | 100,000   | ~$6.00       |
| High (50 PRs/month) | 50        | 100,000   | ~$15.00      |

*Assumes GPT-4 Turbo pricing: $0.01/1K input, $0.03/1K output*
*Cost per PR based on max_tokens: 100,000 constraint*

## Security Considerations

### 1. Token Security

```yaml
env:
  OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}  # Stored in repo secrets
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}      # Auto-generated by Actions
```

- **Least Privilege**: Workflow only needs `contents: write` and `pull-requests: write`
- **Token Rotation**: Use short-lived tokens
- **Audit Logging**: GitHub Actions logs all runs

### 2. Code Safety

- **Read-Only Analysis**: Forge only reads code for context
- **Documentation Only**: Changes limited to comments and docs
- **PR Review Required**: Never auto-merge documentation changes
- **Branch Protection**: Documentation PRs must pass CI checks

### 3. Content Security

- **No Secrets in Docs**: LLM instructed not to document environment variables or keys
- **Safe Examples**: Generated examples use placeholder values
- **Link Validation**: Check that generated links don't leak internal URLs

## Testing Strategy

### Unit Tests

Test documentation analysis logic:

```go
func TestDocumentationAnalyzer(t *testing.T) {
    analyzer := NewDocAnalyzer()
    
    code := `
    package example
    
    // Add returns the sum of a and b
    func Add(a, b int) int { return a + b }
    
    // Missing godoc!
    func Subtract(a, b int) int { return a - b }
    `
    
    gaps := analyzer.FindGaps(code)
    assert.Len(t, gaps, 1)
    assert.Equal(t, "Subtract", gaps[0].Function)
}
```

### Integration Tests

Test full workflow in test repository:

```yaml
# .github/workflows/test-docs-automation.yml
name: Test Documentation Automation

on:
  pull_request:
    paths:
      - '.github/workflows/docs-automation.yml'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run Documentation Workflow (Dry Run)
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          # Test configuration but don't create PR
          cat > test-config.yaml <<EOF
          task: "Analyze pkg/executor/headless for documentation gaps"
          workspace_dir: "."
          git:
            enabled: false  # Dry run
          EOF
          
          forge -headless -headless-config test-config.yaml
      
      - name: Verify No Changes
        run: |
          if [[ -n $(git status --porcelain) ]]; then
            echo "Error: Dry run should not modify files"
            exit 1
          fi
```

## Migration and Rollout

### Phase 1: Pilot (Week 1-2)

- **Scope:** Single repository, manual trigger only
- **Goal:** Validate PR quality and review burden
- **Metrics:** PR review time, documentation accuracy, false positive rate

### Phase 2: Broader Adoption (Week 3-4)

- **Scope:** Enable for additional repositories, monitor multiple PRs
- **Goal:** Test automation across different codebases and PR patterns
- **Metrics:** Token usage per PR, documentation accuracy, team feedback

### Phase 3: Rollout (Week 5+)

- **Scope:** Deploy to all active repositories
- **Goal:** Organization-wide documentation improvement
- **Metrics:** Documentation coverage increase, maintenance cost

## Success Metrics

Per the PRD (Feature Metrics & Success Criteria section), we track:

### Quantitative Metrics

1. **Adoption**: 60%+ of active Forge teams within 6 months
2. **PR Coverage**: 80%+ of feature PRs get automated documentation
3. **Acceptance Rate**: >75% of auto-generated docs accepted without edits
4. **Accuracy**: 90%+ developers say docs are accurate (survey)
5. **Time Saved**: 15+ minutes saved per PR (developer survey)
6. **Cost**: Token usage <$5/month per repository

### Qualitative Metrics

1. **Review Cycles**: -50% reduction in "add docs" review comments
2. **Documentation Quality**: Improved ratings (4.0 → 4.5+/5)
3. **Onboarding Time**: Faster new developer ramp-up
4. **Support Questions**: Reduction in "how does X work?" questions

### Success Thresholds

**Launch Success (3 months):**
- 40%+ teams using the workflow
- 70%+ auto-generated docs accepted without edits
- 4.0+ satisfaction rating
- Zero incidents of incorrect/harmful documentation

**Product-Market Fit (6 months):**
- 60%+ teams using workflow on most PRs
- 80%+ documentation coverage for feature PRs
- 85%+ docs accepted without edits
- "Must-have" feedback from users

## Related Decisions

- **ADR-0029**: Headless Git Integration - Provides git commit capability
- **ADR-0031**: Headless Git PR Creation - Enables PR creation from headless mode

## References

- [Go Documentation Conventions](https://go.dev/doc/effective_go#commentary)
- [GitHub Actions Workflows](https://docs.github.com/en/actions/using-workflows)
- [Effective Documentation Patterns](https://documentation.divio.com/)



## Design Considerations

### Why PR-Triggered Instead of Scheduled?

The PRD is explicit about this choice:
- **Fresh Context**: Documentation written while developer intent is fresh
- **Complete Information**: PR description provides critical context
- **Immediate Feedback**: Developer sees docs while still working on PR
- **Zero Delay**: Documentation ships with code, not days later

Per PRD line 86: "Runs only on PR creation, not on subsequent pushes"

### Why Support BOTH Direct Commit and Separate PR?

This dual-mode support is critical for adoption:

**Trust Building (Separate PR Mode):**
- Teams can't be forced to trust AI immediately
- Separate PR lets them review docs in isolation first
- Easy to reject bad docs without affecting code PR
- Builds confidence over time with low risk

**Efficiency (Direct Commit Mode):**
- Once trust is established, separate PR becomes overhead
- Teams with >80% acceptance rate waste time merging docs PRs
- Direct commit streamlines workflow for trusted AI

**Configuration-Driven:**
- Same workflow file supports both modes
- Teams switch modes via config change, not code change
- Can start conservative, become aggressive as trust builds

This is why PRD line 89 says "Git configuration controls behavior" - the choice MUST be configurable.

### Why Only Markdown Files?

Critical PRD requirement (line 121):
- **Safety**: Prevents AI from accidentally modifying code logic
- **Clear Scope**: Documentation is markdown files, code is code files
- **Lower Risk**: Wrong docs can be fixed, wrong code breaks systems
- **Enforcement**: Safety constraints make this impossible to violate

Future version may support inline docs (comments/docstrings) with additional safety measures.

### Why Non-Blocking Workflow?

Per PRD line 1015:
- **Never Block Development**: Documentation enhances, doesn't gate
- **Graceful Degradation**: If Forge fails, PR still proceeds
- **Developer Control**: Can merge without docs if needed
- **Informational**: Workflow provides information, not enforcement

## Conclusion

This ADR establishes an automated documentation workflow that:
- ✅ Triggers when PRs are opened to document code changes
- ✅ Uses AI to generate contextually appropriate documentation
- ✅ Commits directly to PR branch or creates separate docs PR (configurable)
- ✅ Posts comprehensive PR comments with links to changes
- ✅ Maintains cost efficiency (<$5/month per repository)
- ✅ Ensures documentation ships with code, not after

The implementation leverages ADR-0031's PR creation capability and ADR-0029's Git integration to provide a complete automated documentation solution that eliminates documentation lag by generating docs when developer context is fresh.
