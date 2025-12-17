# 34. Live-Reloadable LLM Settings

**Status:** Accepted
**Date:** 2025-12-02
**Deciders:** Core Team
**Technical Story:** Implement persistent LLM configuration that applies immediately to current session without requiring restart

---

## Context

Forge currently requires users to specify LLM settings via CLI flags (`-model`, `-base-url`) or environment variables on every startup. This creates repetitive configuration overhead and breaks the modern development tool expectation that preferences should persist across sessions.

### Background

Users coming from modern IDEs (VSCode, JetBrains) expect configuration to be:
1. Persistent across sessions
2. Discoverable through a settings UI
3. Immediately effective without restart
4. Easily modifiable without command-line knowledge

Forge's current approach forces users to:
- Remember and type CLI flags repeatedly (`forge -model gpt-4-turbo`)
- Set shell-specific environment variables that don't persist
- Search documentation to find available models
- Restart Forge after changing configuration

### Problem Statement

**How can we make LLM configuration persistent, discoverable, and immediately effective while maintaining CLI override capabilities and security best practices?**

The challenge is balancing:
- User convenience (save once, works forever)
- Security (API keys shouldn't be in plaintext config by default)
- Flexibility (CLI/env vars still work for automation)
- Live reload (changes apply to current session immediately)

### Goals

- Eliminate repetitive CLI flag typing for common configurations
- Make LLM settings discoverable through `/settings` UI
- Apply configuration changes immediately to current session (no restart)
- Persist user preferences across Forge sessions
- Support advanced use cases (OpenRouter, local LLMs, custom base URLs)
- Maintain clear precedence: CLI > ENV > Config > Defaults

### Non-Goals

- Model dropdown with autocomplete (deferred to v1.1)
- Provider presets (OpenRouter, Ollama templates) (deferred to v1.1)
- Connection testing from settings UI (deferred to v1.1)
- Multiple saved profiles (deferred to v1.2)
- Encrypted API key storage via OS keyring (deferred to v1.3)

---

## Decision Drivers

* **Developer Experience**: Users expect modern tools to remember preferences
* **Discoverability**: Settings UI makes options visible without documentation hunting
* **Live Reload**: Changes should apply immediately without disrupting workflow
* **Security**: API keys need safe storage options (prefer ENV over config file)
* **Flexibility**: CLI/env var overrides must still work for automation
* **Simplicity**: MVP should be minimal but complete (no half-features)

---

## Considered Options

### Option 1: Config File Only (Traditional Approach)

**Description:** Store LLM settings in `~/.forge/config.json` as structured data. Read at startup, no live reload.

**Pros:**
- Simple implementation (already have config system)
- Standard approach (git, docker use config files)
- Easy to version control and share team configs

**Cons:**
- Requires Forge restart to apply changes
- Breaks user expectation of immediate effect
- Poor UX compared to modern tools
- Forces context loss when switching models mid-conversation

**Precedence:** CLI > ENV > Config > Defaults (standard)

### Option 2: Environment Variables Only

**Description:** Use only environment variables for configuration. No config file persistence.

**Pros:**
- Already works (current system)
- Good for automation/CI
- No file permissions to worry about

**Cons:**
- Not persistent (need to set every shell session)
- Shell-specific syntax (bash vs fish vs PowerShell)
- Poor discoverability (hidden from users)
- Can't edit through UI
- Doesn't solve the core problem

**Precedence:** CLI > ENV > Defaults

### Option 3: Live-Reloadable Config with Session Application

**Description:** Store settings in config file BUT apply changes immediately to current session when saved via `/settings` UI. Use Viper's WatchConfig for file changes or implement manual reload on save.

**Pros:**
- Best of both worlds: persistent AND immediate
- Settings UI changes apply instantly (no restart)
- Config file for long-term persistence
- CLI/ENV overrides still work
- Matches modern tool expectations (VSCode behavior)
- Enables mid-conversation model switching without context loss

**Cons:**
- More complex implementation (need config reload mechanism)
- File watcher overhead (can mitigate by reload-on-save only)
- Need to propagate config changes to active agent/executor

**Precedence:** CLI > ENV > Config > Defaults (standard + live reload)

**Implementation Approach - Explicit Reload on Save:**
```go
// In settings UI save handler
config.SaveLLMSettings(settings)
app.ApplyLLMSettings(settings)  // Immediate effect
```
- Simple: only reload when user explicitly saves via UI
- No file watcher overhead or complexity
- Clear user model: save action = apply action
- Sufficient for settings UI use case

### Option 4: Dual Storage (Config + ENV Preference)

**Description:** Store settings in config file but always prefer ENV vars. Show warnings if both exist.

**Pros:**
- Security-first (encourages ENV usage)
- Clear separation (config for convenience, ENV for security)

**Cons:**
- Confusing user mental model (why set config if ENV wins?)
- Doesn't solve live-reload problem
- Added complexity for minimal benefit
- Standard precedence already handles this

---

## Decision

**Chosen Option:** Option 3 - Live-Reloadable Config with Explicit Reload on Save

### Rationale

This option best balances all decision drivers:

1. **Developer Experience**: Settings persist across sessions AND apply immediately
2. **Live Reload**: No restart required - changes take effect on next message
3. **Discoverability**: Settings UI makes configuration visible and editable
4. **Security**: API key is optional in config (ENV var recommended, clearly documented)
5. **Flexibility**: CLI/ENV overrides continue to work as expected
6. **Simplicity**: Explicit reload on save is simpler than file watcher

**Why Not Option 1 (Config Only):**
Requiring restart violates modern UX expectations. Users switching models mid-conversation would lose context and interrupt flow.

**Why Not Option 2 (ENV Only):**
Doesn't solve persistence or discoverability. Power users already know env vars - this feature is for everyone else.

**Implementation: Explicit Reload Only**
We implement explicit reload on save rather than automatic file watching because:
- Simpler implementation (no file watcher dependencies)
- Better performance (no continuous monitoring overhead)
- Clearer user model (save action directly triggers apply)
- Matches settings UI workflow (user saves when ready to apply)
- No need to handle external file edits (settings modified via UI only)

**Security Considerations:**
- API key in config file is plaintext (documented risk)
- Users warned to use ENV var for production
- File permissions set to 0600 on save
- Future: OS keyring integration (v1.3)

---

## Consequences

### Positive

- Users configure model once, it persists forever
- Mid-conversation model switching without context loss
- Settings discoverable through `/settings` command
- Matches modern tool UX (VSCode, JetBrains)
- Reduces support burden ("how do I change model?")
- Enables experimentation (try Claude vs GPT-4 instantly)
- Team config sharing (export/import settings)
- OpenRouter/local LLM setup becomes discoverable

### Negative

- Config file may contain plaintext API key (mitigated by ENV preference)
- Need to propagate config changes through app layers
- Settings UI becomes more complex (3 fields instead of none)
- Need clear documentation on precedence rules

### Neutral

- Config file location standardized (`~/.forge/config.json`)
- JSON format for settings (already established)
- CLI flags remain primary for automation/CI

---

## Implementation

### Core Components

**1. Config Schema (pkg/config/llm.go):**
```go
type LLMSection struct {
    Model   string `json:"model"`    // e.g., "gpt-4-turbo"
    BaseURL string `json:"baseUrl"` // e.g., "https://openrouter.ai/api/v1"
    APIKey  string `json:"apiKey"`  // Optional, ENV preferred
}

func (s *LLMSection) Name() string { return "llm" }
func (s *LLMSection) Defaults() Section { ... }
func (s *LLMSection) Validate() error { ... }
```

**2. Settings UI (pkg/ui/settings/settings.go):**
- Add LLM Settings section
- Text input fields: Model, Base URL, API Key
- API Key masked with EchoPassword mode (bubbles/textinput)
- Save (Ctrl+S) triggers: SaveToFile() + ApplyToSession()

**3. Precedence Resolution (pkg/config/config.go):**
```go
func (m *Manager) GetLLMModel() string {
    if cliFlag != "" { return cliFlag }
    if envVar := os.Getenv("OPENAI_MODEL"); envVar != "" { return envVar }
    if cfg := m.Get("llm").(*LLMSection); cfg.Model != "" { return cfg.Model }
    return "gpt-4o" // Default
}
```

**4. Live Application (pkg/app/app.go):**
```go
func (a *App) ApplyLLMSettings(settings *config.LLMSection) {
    // Update executor's LLM provider
    a.executor.UpdateLLMConfig(settings.Model, settings.BaseURL, settings.APIKey)
    // Show confirmation
    a.ui.ShowMessage("✓ Settings saved and applied")
}
```

### Migration Path

**Phase 1 (v1.0):**
1. Implement LLMSection config schema
2. Add settings UI fields (Model, Base URL, API Key)
3. Implement explicit reload on save
4. Document precedence and security considerations

**Phase 2 (v1.1):**
- Model dropdown with popular choices
- Provider presets (OpenRouter, Ollama templates)
- Connection testing

**Phase 3 (v1.2+):**
- Model profiles (save multiple configs)
- Advanced parameters (temperature, max tokens)

### Timeline

- **Week 1-2**: Config schema + precedence logic
- **Week 3**: Settings UI integration
- **Week 4**: Testing, documentation, polish
- **Target Release**: v1.0 (Q1 2025)

---

## Validation

### Success Metrics

- **Adoption**: >80% of users configure settings within first week
- **Support Reduction**: "How to change model" questions drop by 90%
- **Time Savings**: Average 30s saved per Forge startup (no flag typing)
- **Feature Discovery**: 50% of users try non-default models within 30 days

### Monitoring

- Track config file creation/modification rates
- Measure time between Forge start and first message (should decrease)
- Survey users on configuration experience
- Monitor support tickets related to model configuration

### Acceptance Criteria

**Must Have:**
- [ ] Settings persist across Forge restarts
- [ ] Changes apply to current session immediately (no restart)
- [ ] CLI flags override config file
- [ ] ENV vars override config file
- [ ] API key masked in settings UI
- [ ] Config file permissions set to 0600
- [ ] Clear documentation on precedence

**Should Have:**
- [ ] Visual confirmation when settings saved
- [ ] Help text explaining each field
- [ ] Example values shown in placeholders

**Won't Have (v1.0):**
- [ ] Model dropdown (deferred to v1.1)
- [ ] Connection testing (deferred to v1.1)
- [ ] OS keyring integration (deferred to v1.3)

---

## Related Decisions

- [ADR-0017](0017-auto-approval-and-settings-system.md) - Settings System Architecture
- [ADR-0032](0032-agent-scratchpad-notes-system.md) - Config Section Interface Pattern

---

## References

- [Product Requirements](../product/features/llm-settings.md) - Full feature specification
- [VSCode Settings Sync](https://code.visualstudio.com/docs/editor/settings-sync) - UX inspiration
- [Viper Config Library](https://github.com/spf13/viper) - Go config management
- [Charm Bubbles](https://github.com/charmbracelet/bubbles) - TUI input components

---

## Notes

**API Key Storage Security:**
The decision to allow (but not encourage) API key storage in config file is pragmatic:
- Many users prefer convenience over perfect security for local dev
- ENV var option clearly documented and recommended
- Future OS keyring integration (v1.3) provides secure alternative
- File permissions (0600) provide basic protection
- Warning shown in UI when API key set in config

**Live Reload vs Restart:**
The choice to implement immediate session application (vs restart-only) is critical for UX:
- Enables mid-conversation model comparison
- Preserves context and conversation flow
- Matches user expectations from modern tools
- Minimal implementation complexity with explicit reload approach

**Last Updated:** 2025-12-02
