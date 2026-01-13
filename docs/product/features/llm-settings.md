# Product Requirements: LLM Settings

**Feature:** Persistent LLM Configuration  
**Version:** 1.0  
**Status:** Proposed  
**Owner:** Core Team  
**Last Updated:** January 2025

---

## Product Vision

Eliminate the friction of configuring LLM settings on every Forge startup. Users should configure their preferred model, base URL, and provider once through an intuitive settings interface, with these preferences persisting across sessions **and applying immediately to the current session**. No more remembering CLI flags, editing config files, or restarting Forge—just set it and it works instantly.

**Strategic Alignment:** Reducing configuration friction accelerates time-to-productivity for all users. By making LLM settings persistent, discoverable, and live-reloadable, we lower the barrier to entry for new users while respecting the needs of power users who experiment with different models and providers without interrupting their workflow.

---

## Problem Statement

Developers using Forge face repetitive configuration overhead that disrupts their workflow and creates unnecessary friction:

1. **Repetitive CLI Flags:** Users must specify `-model gpt-4-turbo` every single time they start Forge, even though their preference rarely changes
2. **Undiscoverable Options:** New users don't know what models are available or how to configure alternative providers (OpenRouter, local LLMs)
3. **Environment Variable Fatigue:** Setting up shell-specific env vars for `OPENAI_BASE_URL` is cumbersome and differs across shells/platforms
4. **No Persistence:** Unlike every other modern development tool (VSCode, IDEs), Forge doesn't remember user preferences between sessions
5. **Documentation Hunting:** Users search docs to find "how do I use Claude instead of GPT-4?"

**Current Workarounds (All Problematic):**
- **Shell aliases:** `alias forge-gpt4='forge -model gpt-4-turbo'` → Platform-specific, brittle, doesn't help new users
- **Environment variables:** `export FORGE_MODEL=gpt-4-turbo` → Must remember to set, shell-specific, not discoverable
- **CLI flags every time:** `forge -model gpt-4-turbo -base-url https://...` → Tedious, error-prone, breaks flow

**Real-World Impact:**
- **First-time user:** Tries Forge with default model (GPT-4o), wants Claude instead, searches docs for 10 minutes, gives up
- **Power user:** Switches between projects, forgets which model they prefer, tries default, realizes mid-conversation it's wrong model, opens settings to switch instantly
- **Team member:** Team standardizes on GPT-4 Turbo, but everyone has to remember the flag—leads to inconsistent usage
- **OpenRouter user:** Wants to use multiple models via OpenRouter, but configuring base URL as env var is clunky and not live-reloadable

**Cost of Lack of Persistence:**
- Average 30 seconds wasted per Forge startup (typing flags or checking aliases)
- 10+ support questions per month: "How do I change the model?"
- Reduced discoverability of alternative providers and models
- Users stick with defaults because changing is too hard

---

## Key Value Propositions

### For New Users (Getting Started)
- **Discoverability:** Browse available models and providers through visual interface, no documentation needed
- **One-Time Setup:** Configure model preference once, never think about it again
- **Guided Configuration:** Clear descriptions of each model's strengths and use cases
- **Safe Defaults:** Comes with sensible defaults (GPT-4o), easily changeable

### For Power Users (Efficiency)
- **Zero Overhead:** No CLI flags to remember or type repeatedly
- **Instant Switching:** Change models through `/settings` and see results immediately—no restart required
- **Live Experimentation:** Test different models mid-conversation to compare quality and speed
- **Multiple Profiles:** (Future) Save and switch between model configurations (GPT-4 for refactors, Claude for analysis)
- **Advanced Options:** Configure base URL for OpenRouter, local LLMs, or alternative endpoints with immediate effect

### For Teams (Standardization)
- **Consistent Setup:** Export LLM settings as part of team configuration template
- **Easy Onboarding:** New team members import settings, immediately have correct model
- **Visible Configuration:** Team leads can verify everyone's using the approved model/provider
- **No Secret Knowledge:** Settings visible and editable through UI, not buried in shell configs

---

## Target Users & Use Cases

### Primary: Frequent Forge User

**Profile:**
- Uses Forge daily across multiple projects
- Has a preferred LLM model for different tasks (GPT-4 for code, Claude for docs)
- Values efficiency and hates repetitive configuration
- Comfortable with CLI but prefers persistent settings

**Key Use Cases:**
- Setting default model once, having it remembered forever
- Switching models occasionally without command-line flags
- Configuring custom base URL for OpenRouter or local LLM
- Understanding what models are available without reading docs

**Pain Points Addressed:**
- Tired of typing `-model gpt-4-turbo` every startup
- Forgets exact model name (is it `gpt-4-turbo` or `gpt-4-turbo-preview`?)
- Wants to try Claude but doesn't know the model string
- Frustrated that preferences don't persist like every other tool

**Success Story:**
"I use GPT-4 Turbo for refactoring and Claude Sonnet for code reviews. I opened `/settings`, went to LLM Settings, set my default model to GPT-4 Turbo, and saved. Now every time I start Forge, it just uses GPT-4 Turbo automatically. When I want Claude mid-session, I open settings, switch to Claude Sonnet, hit Ctrl+S, and my next message uses Claude immediately—no restart needed. The whole switch takes 15 seconds, and I never lose my context or have to remember CLI flags."

**User Journey:**
```
Daily Forge usage, tired of typing -model flag
    ↓
Open Forge
    ↓
Type /settings
    ↓
Navigate to "LLM Settings" section
    ↓
See current model: gpt-4o (default)
    ↓
Navigate to "Model" field
    ↓
Enter to edit → See available models:
    - gpt-4o (current)
    - gpt-4-turbo
    - gpt-4
    - claude-3-5-sonnet-20241022
    - claude-3-opus-20240229
    ↓
Type "gpt-4-turbo" and confirm (Enter)
    ↓
Modified indicator (*) appears on Model field
    ↓
Save settings (Ctrl+S)
    ↓
Settings saved to ~/.forge/config.json AND applied to current session
    ↓
Close settings (Esc)
    ↓
Next message uses gpt-4-turbo immediately (no restart)
    ↓
Restart Forge tomorrow → Still uses gpt-4-turbo (persisted)
    ↓
Success - Never types -model flag again, changes take effect instantly
```

---

### Secondary: OpenRouter / Alternative Provider User

**Profile:**
- Uses OpenRouter for access to multiple models with one API key
- Or runs local LLMs (Ollama, LM Studio)
- Wants to configure custom base URL
- Technically savvy but prefers UI configuration over env vars

**Key Use Cases:**
- Configuring custom base URL (https://openrouter.ai/api/v1) with immediate effect
- Testing different models without API key juggling or restarts
- Using local LLM endpoints (http://localhost:1234/v1) and switching instantly
- Comparing provider responses mid-conversation (OpenRouter vs Anthropic vs OpenAI)

**Pain Points Addressed:**
- Setting OPENAI_BASE_URL as env var is platform-specific and clunky
- Wants to try multiple OpenRouter models without remembering base URL every time
- Needs to document base URL for team but it's hidden in shell config
- Accidentally forgets base URL, sends requests to wrong endpoint

**Success Story:**
"I use OpenRouter because I can try GPT-4, Claude, and Gemini with one API key. I configured the base URL once in settings (https://openrouter.ai/api/v1), saved it, and now I can switch between any model OpenRouter offers just by changing the model field. When I want to test Claude vs GPT-4 mid-conversation, I just open settings, change the model, save, and my next message uses the new model instantly. No more environment variables, no more CLI flags, no restarts. Just works."

---

### Tertiary: Team Lead (Standardization)

**Profile:**
- Manages team using Forge
- Wants everyone on same model for consistency
- Needs to document team standards
- Values reproducibility and ease of onboarding

**Key Use Cases:**
- Setting team-wide default model (GPT-4 Turbo)
- Exporting LLM settings as part of team config
- Onboarding new members with pre-configured settings
- Ensuring everyone uses approved provider/model

**Pain Points Addressed:**
- Team has inconsistent model usage → unpredictable results
- Hard to help team member when they're on different model
- Onboarding docs say "set these env vars" → platform-specific, error-prone
- No visibility into what model each person is using

**Success Story:**
"Our team standardized on Claude Sonnet for code reviews. I configured it in settings, exported our team config, and added it to the onboarding repo. New hires import the settings and immediately have the right model configured. No more 'why is this working differently for you?' questions."

---

## Product Requirements

### Priority 0 (Must Have - v1.0)

#### P0-1: LLM Configuration Section
**Description:** Add new "LLM Settings" section to settings system for model and provider configuration

**User Stories:**
- As a user, I want to configure my preferred LLM model so it persists across sessions
- As a user, I want to set a custom base URL so I can use alternative providers (OpenRouter, local LLMs)
- As a user, I want to see what models are available so I can make informed choices

**Acceptance Criteria:**
- New "LLM Settings" section visible in `/settings` overlay
- Settings section implements existing `config.Section` interface
- Data persists to `~/.forge/config.json` (or platform-equivalent)
- Settings load on application startup
- Validation prevents invalid model names or URLs
- Three text input fields: Model Name, Base URL, API Key
- API Key field displays masked value (••••••••) when not empty
- Each field shows appropriate help text when focused
- Changes tracked with modified indicator (*)
- Settings auto-save on Ctrl+S

**Configuration Fields:**
1. **Model** (string, required)
   - Default: `gpt-4o`
   - Item type: `itemTypeText`
   - Validation: Non-empty string
   - Help text: "The LLM model to use for code generation and chat. Changes apply immediately to the current session."
   - Examples: gpt-4o, gpt-4-turbo, claude-3-5-sonnet-20241022
   
2. **Base URL** (string, optional)
   - Default: empty (uses OpenAI default)
   - Item type: `itemTypeText`
   - Validation: Valid URL format if provided, or empty
   - Help text: "Custom API endpoint for OpenAI-compatible providers (OpenRouter, local LLMs). Changes apply immediately. Leave blank for default OpenAI endpoint."
   - Examples: https://openrouter.ai/api/v1, http://localhost:11434/v1
   
3. **API Key** (string, optional)
   - Default: empty (uses OPENAI_API_KEY env var)
   - Item type: `itemTypeText`
   - Display: Masked as ••••••••• when not empty
   - Validation: Optional - any string accepted
   - Help text: "API key for the LLM provider. Changes apply immediately to the current session. Leave blank to use OPENAI_API_KEY environment variable (recommended for security)."
   - Security note: Stored in config file - environment variable is more secure for production use

**Technical Implementation:**
```go
// pkg/config/llm.go
type LLMSection struct {
    mu      sync.RWMutex
    model   string // Default: "gpt-4o"
    baseURL string // Default: ""
}

func (s *LLMSection) ID() string { return "llm" }
func (s *LLMSection) Title() string { return "LLM Settings" }
func (s *LLMSection) Description() string {
    return "Configure your preferred language model and API provider"
}
```

**File Structure:**
```
~/.forge/config.json
{
  "version": "1.0",
  "sections": {
    "llm": {
      "model": "gpt-4-turbo",
      "base_url": "https://openrouter.ai/api/v1"
    },
    "auto_approval": { ... },
    "command_whitelist": { ... }
  }
}
```

---

#### P0-2: Settings UI Integration
**Description:** Add LLM settings to interactive settings overlay with intuitive editing

**User Stories:**
- As a user, I want to edit LLM settings through the TUI so I don't have to edit JSON files
- As a user, I want validation feedback immediately so I don't save invalid settings
- As a user, I want help text explaining each setting

**Acceptance Criteria:**
- "LLM Settings" section appears in settings overlay
- Text input field for "Model" with current value displayed
- Text input field for "Base URL" with current value displayed (can be empty)
- Real-time validation with inline error messages
- Help text displayed for focused field
- Changes mark settings as modified (visual indicator)
- Auto-save on change (or Ctrl+S explicit save)

**UI Layout:**
```
╭────────────────────────────────────────────────────────────╮
│                  Settings - LLM Configuration              │
├────────────────────────────────────────────────────────────┤
│                                                             │
│ Configure your preferred language model and API provider   │
│                                                             │
│ ➜ Model Name: gpt-4-turbo *                                │
│   Base URL: https://openrouter.ai/api/v1                   │
│   API Key: ••••••••                                         │
│                                                             │
├────────────────────────────────────────────────────────────┤
│ ↑↓/jk: Navigate • Enter: Edit • Ctrl+S: Save • Esc: Close  │
╰────────────────────────────────────────────────────────────╯
```

**UI Elements:**
- **Section Header:** "LLM Configuration" in mint green (focused) or default color
- **Section Description:** Muted gray italic text explaining the section's purpose
- **Item Labels:** "Model Name:", "Base URL:", "API Key:" in muted gray (default) or bright white + bold (focused)
- **Item Values:** Displayed after colon, in muted gray (default) or bright white (focused)
- **Focus Indicator:** ➜ prefix on currently focused item
- **Modified Indicator:** * asterisk in salmon pink when value changed but not saved
- **Masked Display:** API key shows ••••••••• instead of actual value for security
- **Help Bar:** Bottom row shows context-sensitive keyboard shortcuts

**Edit Dialog (when Enter pressed on text item):**
```
╭──────────────────────────────────────────╮
│          Edit Model Name                 │
├──────────────────────────────────────────┤
│                                          │
│ Model Name                               │
│ ┌──────────────────────────────────────┐ │
│ │ gpt-4-turbo▸                         │ │
│ └──────────────────────────────────────┘ │
│                                          │
│ The LLM model to use for code generation │
│ Examples: gpt-4o, gpt-4-turbo,           │
│ claude-3-5-sonnet-20241022               │
│                                          │
├──────────────────────────────────────────┤
│ [Enter] Confirm • [Esc] Cancel           │
╰──────────────────────────────────────────╯
```

**Dialog Features:**
- Rounded border in salmon pink
- Dark background
- Current value pre-filled with cursor (▸) at end
- Multi-line help text explaining the field
- Character count for limited fields (not shown for LLM fields)
- Validation error messages appear in salmon pink if validation fails

**Interaction Flow:**

**Navigating to LLM Settings:**
1. User presses `/settings` or uses settings command
2. Settings overlay opens showing all sections (tabs)
3. User presses Tab or → to navigate to "LLM Settings" section
4. Section header changes to salmon pink indicating focus
5. First item ("Model Name") is automatically selected with ➜ indicator

**Editing Model Name:**
1. User navigates to "Model Name" field (already focused, or use ↓ key)
2. Presses Enter to open edit dialog
3. `showTextEditDialog()` creates modal dialog:
   - Title: "Edit Model Name"
   - Current value: "gpt-4o" with cursor (▸) at end
   - Help text shows examples of valid model names
4. User types new model name (e.g., "gpt-4-turbo")
5. Presses Enter to confirm (or Esc to cancel)
6. Dialog validates: model name not empty
7. If valid:
   - Dialog closes
   - Main view updates: "Model Name: gpt-4-turbo *"
   - Asterisk (*) appears indicating unsaved change
   - Item marked as `modified: true`
8. Help text updates to show "Ctrl+S: Save" prominently

**Editing Base URL:**
1. User presses ↓ to navigate to "Base URL" field
2. Focus indicator (➜) moves to Base URL line
3. Presses Enter to open edit dialog
4. Dialog shows current value (may be empty)
5. User types new URL: "https://openrouter.ai/api/v1"
6. Validation runs: check valid URL format or empty string
7. If valid, dialog closes and value updates with * indicator

**Editing API Key (Masked Field):**
1. User navigates to "API Key" field
2. Current display shows: "API Key: ••••••••" (masked)
3. Presses Enter to open edit dialog
4. Dialog shows full unmasked value for editing
5. User can see and edit actual key
6. User types new API key or clears field
7. Presses Enter to confirm
8. Main view updates: "API Key: ••••••••" (masked again)
9. Modified indicator (*) appears

**Saving Changes:**
1. User has modified one or more fields (indicated by *)
2. Presses Ctrl+S to save
3. `saveChanges()` method:
   - Extracts all item values from LLM section
   - Creates data map: {"model": "gpt-4-turbo", "base_url": "https://...", "api_key": "sk-..."}
   - Calls config section's `SetData(data)` method
   - Persists to `~/.forge/config.json` via FileStore
4. **Live Update to Current Session:**
   - Agent's provider client is reconfigured with new settings
   - Next LLM call uses updated model/base URL/API key
   - No restart required - changes active immediately
5. Visual confirmation: Brief "✓ Settings saved and applied" message
6. Modified indicators (*) cleared from all items
7. File written with 600 permissions (owner read/write only)

**Discarding Changes:**
1. User modifies fields but decides not to save
2. Presses Esc to close settings
3. If changes exist, confirmation dialog appears:
   - "Unsaved Changes"
   - "You have unsaved changes"
   - Options: "Save: Ctrl+S" or "Discard: Press Esc again"
4. User presses Esc again to confirm discard
5. Settings overlay closes, changes lost
6. Original config file unchanged

---

#### P0-3: Configuration Precedence System
**Description:** Implement clear precedence for how LLM settings are resolved from multiple sources

**User Stories:**
- As a user, I want CLI flags to override my saved settings for one-off model changes
- As a developer, I want environment variables to take precedence over config file for CI/CD
- As a team member, I want to understand the order settings are applied

**Precedence Order (highest to lowest):**
1. **CLI Flags** (`-model`, `-base-url`)
2. **Environment Variables** (`FORGE_MODEL`, `OPENAI_BASE_URL`)
3. **Config File** (`~/.forge/config.json`)
4. **Hardcoded Defaults** (`gpt-4o`, empty base URL)

**Acceptance Criteria:**
- CLI flags always win over all other sources
- Environment variables override config file
- Config file used if no CLI flag or env var set
- Defaults used only if nothing else specified
- Precedence documented in help text and docs
- Log which source was used (debug mode)

**Implementation Example:**
```go
func parseFlags() *Config {
    config := &Config{}
    
    // Load from config file as defaults
    var defaultModel = "gpt-4o"
    var defaultBaseURL = ""
    
    if config.IsInitialized() {
        if llm := config.GetLLMSection(); llm != nil {
            defaultModel = llm.GetModel()
            defaultBaseURL = llm.GetBaseURL()
        }
    }
    
    // Check env vars (higher precedence than config)
    envModel := os.Getenv("FORGE_MODEL")
    if envModel != "" {
        defaultModel = envModel
    }
    
    envBaseURL := os.Getenv("OPENAI_BASE_URL")
    if envBaseURL != "" {
        defaultBaseURL = envBaseURL
    }
    
    // CLI flags (highest precedence)
    flag.StringVar(&config.Model, "model", defaultModel, "LLM model to use")
    flag.StringVar(&config.BaseURL, "base-url", defaultBaseURL, "API base URL")
    
    flag.Parse()
    return config
}
```

**Logging Example (Debug Mode):**
```
[DEBUG] Loading LLM configuration...
[DEBUG] Config file: model=gpt-4-turbo, base_url=
[DEBUG] Environment: FORGE_MODEL not set, OPENAI_BASE_URL not set
[DEBUG] CLI flags: -model not provided, -base-url not provided
[DEBUG] Final configuration: model=gpt-4-turbo (from config), base_url= (default)
```

---

#### P0-4: Documentation and Help Text
**Description:** Clear documentation of LLM settings feature and in-app help

**User Stories:**
- As a new user, I want to understand how to configure my preferred model
- As a power user, I want to know the precedence order for troubleshooting
- As a team lead, I want documentation to share with team

**Acceptance Criteria:**
- Settings overlay shows inline help for each field
- `/help` command mentions `/settings` for LLM configuration
- README updated with settings section
- Example configurations in docs (OpenRouter, local LLM)
- Troubleshooting guide for common issues

**Help Text Examples:**

**Model Field:**
```
The LLM model to use for code generation and chat.

Common models:
  - gpt-4o (fast, intelligent, recommended)
  - gpt-4-turbo (most capable, slower)
  - gpt-4 (reliable, widely tested)
  - claude-3-5-sonnet-20241022 (excellent for code)
  - claude-3-opus-20240229 (most capable Claude)

Changes apply immediately to current session.
```

**Base URL Field:**
```
Custom API endpoint for OpenAI-compatible providers.

Examples:
  - OpenRouter: https://openrouter.ai/api/v1
  - Local Ollama: http://localhost:11434/v1
  - Azure OpenAI: https://your-resource.openai.azure.com/
  
Leave blank to use default OpenAI endpoint (api.openai.com).
```

---

### Priority 1 (Should Have - v1.1)

#### P1-1: Model Dropdown with Popular Choices
**Description:** Replace text input with dropdown showing popular model options

**User Stories:**
- As a new user, I want to see what models are available without searching docs
- As any user, I want to avoid typos in model names
- As a user, I want to discover new models I didn't know existed

**Acceptance Criteria:**
- Dropdown shows curated list of popular models
- Models grouped by provider (OpenAI, Anthropic, etc.)
- Option to enter custom model name (for new/unlisted models)
- Search/filter within dropdown
- Visual indicator of currently selected model

**Model List (Initial):**
```
OpenAI:
  - gpt-4o (recommended)
  - gpt-4-turbo
  - gpt-4
  - gpt-3.5-turbo

Anthropic:
  - claude-3-5-sonnet-20241022
  - claude-3-opus-20240229
  - claude-3-sonnet-20240229

Custom:
  - [Enter custom model name...]
```

---

#### P1-2: Provider Preset Configurations
**Description:** One-click provider presets that configure both model and base URL

**User Stories:**
- As an OpenRouter user, I want to select "OpenRouter" and have base URL auto-filled
- As a local LLM user, I want "Ollama" preset to configure localhost endpoint
- As a new user, I want guided setup for popular providers

**Acceptance Criteria:**
- "Provider" dropdown with presets: OpenAI, Anthropic, OpenRouter, Ollama, Azure OpenAI
- Selecting preset auto-fills base URL and suggests compatible models
- Option for "Custom" provider with manual configuration
- Help text explains what each provider is and when to use it

**Provider Presets:**
```yaml
OpenAI:
  base_url: "" # Default
  suggested_models: [gpt-4o, gpt-4-turbo, gpt-4]
  
Anthropic:
  base_url: "https://api.anthropic.com"
  suggested_models: [claude-3-5-sonnet-20241022, claude-3-opus-20240229]
  
OpenRouter:
  base_url: "https://openrouter.ai/api/v1"
  suggested_models: [openai/gpt-4-turbo, anthropic/claude-3-5-sonnet]
  
Ollama (Local):
  base_url: "http://localhost:11434/v1"
  suggested_models: [llama2, codellama, mistral]
```

---

#### P1-3: In-Settings Model Test
**Description:** Test API connection and model availability directly from settings

**User Stories:**
- As a user, I want to verify my API key works before closing settings
- As a user configuring a new provider, I want to test the connection immediately
- As a troubleshooting user, I want to see what's wrong with my configuration

**Acceptance Criteria:**
- "Test Connection" button in LLM settings
- Button sends simple prompt to configured model/endpoint
- Shows success message with response time and model info
- Shows error message with actionable troubleshooting steps
- Doesn't interfere with ongoing conversations

**Test Flow:**
```
User clicks "Test Connection"
    ↓
Shows "Testing..." spinner
    ↓
Sends test prompt: "Reply with OK"
    ↓
Success:
  ✓ Connection successful
  Model: gpt-4-turbo
  Response time: 1.2s
  API: OpenAI (api.openai.com)
    
Failure:
  ✗ Connection failed
  Error: Invalid API key
  Fix: Check OPENAI_API_KEY environment variable
```

---

### Priority 2 (Nice to Have - v1.2+)

#### P2-1: Multiple Model Profiles
**Description:** Save and switch between named model configurations

**User Stories:**
- As a power user, I want different profiles for different tasks (refactoring vs analysis)
- As a team member, I want to save team config and personal config separately
- As an experimenter, I want to quickly switch between models to compare

**Example:**
```
Profiles:
  - Default (GPT-4 Turbo)
  - Code Review (Claude Sonnet) 
  - Quick Tasks (GPT-3.5 Turbo)
  - Experiments (Local Llama)
```

---

#### P2-2: Advanced LLM Parameters
**Description:** Configure temperature, max tokens, top_p in settings

**User Stories:**
- As a power user, I want to tune model behavior for my workflow
- As a conservative user, I want lower temperature for deterministic code
- As a creative user, I want higher temperature for brainstorming

**Parameters:**
- Temperature (0.0-2.0)
- Max tokens (limit response length)
- Top P (nucleus sampling)
- Frequency penalty
- Presence penalty

---

#### P2-3: Cost Tracking and Budget Alerts
**Description:** Track API usage and costs, alert when approaching budget

**User Stories:**
- As a cost-conscious user, I want to see how much I'm spending
- As a team lead, I want to set budget limits
- As any user, I want to avoid surprise API bills

**Features:**
- Real-time cost estimation
- Budget limit setting
- Usage alerts (50%, 75%, 90% of budget)
- Historical usage graphs

---

## Security Considerations

### API Key Storage

**Decision: Support optional API key storage with security warnings**

**Rationale:**
- **Primary Use Case:** Environment variables remain the recommended approach for production and security-conscious users
- **Secondary Use Case:** Config file storage provides convenience for local development and experimentation
- **User Choice:** Allow users to choose their preferred method based on their security requirements
- **Clear Warnings:** UI and documentation must emphasize security implications

**Implementation:**

1. **API Key Field in Settings:**
   - Optional text field in LLM Settings section
   - Displays masked value (••••••••) when not empty
   - Full value visible in edit dialog for editing
   - Clear help text warning about security implications

2. **Security Warning Display:**
   ```
   API Key: ••••••••
   
   ⚠️  WARNING: Storing API keys in config file is less secure than
   using environment variables. Config files may be accidentally
   committed to version control or exposed in backups.
   
   Recommended: Set OPENAI_API_KEY environment variable instead
   and leave this field blank.
   ```

3. **Precedence Order (Updated):**
   - **CLI Flag** (`-api-key`) - Highest priority
   - **Environment Variable** (`OPENAI_API_KEY`) - Recommended for security
   - **Config File** (`~/.forge/config.json`) - Convenience for local dev
   - **Empty** - User prompted for API key if none found

4. **File Permissions Enforcement:**
   - Config file created with 600 permissions (owner read/write only)
   - On load, verify permissions and warn if too permissive
   - Refuse to save API key if permissions > 600
   - Platform-specific permission checks

5. **Git Ignore Recommendation:**
   - Documentation includes note to add `~/.forge/config.json` to global gitignore
   - First-run wizard could prompt to configure this
   - Settings UI shows reminder if git detected in parent directories

**Security Best Practices Communicated:**
- Environment variables preferred for production
- Config file storage acceptable for local development only
- Never commit config file with API key to version control
- Use different API keys for development vs production
- Rotate keys regularly if stored in config file

**Future Enhancement:**
- Implement OS keyring integration (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- Encrypted storage option with master password
- Per-project API key isolation
- Automatic key rotation reminders

---

### File Permissions

**Requirement:** Config file must be readable only by owner

**Implementation:**
- Set file permissions to 600 (owner read/write only) on creation
- Verify permissions on load, warn if too permissive
- Documentation includes permission requirements

```bash
chmod 600 ~/.forge/config.json
```

---

## Success Metrics

**Adoption Metrics:**
- **Target:** 60% of users configure LLM settings within first week
- **Target:** 40% of startups use persisted model (not default)
- **Target:** 80% reduction in `-model` CLI flag usage

**Efficiency Metrics:**
- **Target:** Average time to configure model: <30 seconds
- **Target:** Zero config-related startup failures
- **Target:** 50% reduction in "how to change model" support questions

**Quality Metrics:**
- **Target:** <1% config file corruption rate
- **Target:** 100% of model names validated before save
- **Target:** Zero security issues with stored settings

---

## Implementation Details

### Settings UI Architecture

The LLM settings UI builds on the existing settings overlay system implemented in `pkg/executor/tui/overlay/settings.go`. The settings overlay uses a section-based architecture where each configuration area (auto-approval, command whitelist, LLM) is represented as a section with multiple items.

**Key Components:**

1. **Settings Section (`settingsSection`)**
   - Represents a logical grouping of related settings
   - Contains: ID, title, description, and list of items
   - Example: "LLM Settings" section contains model, base URL, and API key items

2. **Settings Items (`settingsItem`)**
   - Individual configurable values within a section
   - Three types supported:
     - `itemTypeToggle`: Boolean on/off settings (e.g., auto-approval flags)
     - `itemTypeText`: String input fields (e.g., model name, base URL)
     - `itemTypeList`: Complex list items (e.g., whitelist patterns)

3. **Input Dialogs (`inputDialog`)**
   - Modal dialogs for editing text and list items
   - Support validation, help text, and error messages
   - Can contain multiple fields (for complex edits like whitelist patterns)

**LLM Settings Section Structure:**

```go
// In buildSections() method
llmSection := settingsSection{
    id:          "llm",
    title:       "LLM Settings",
    description: "Configure your preferred language model and API provider",
    items: []settingsItem{
        {
            key:         "model",
            displayName: "Model Name",
            itemType:    itemTypeText,
            value:       cfg.GetModel(), // e.g., "gpt-4-turbo"
            modified:    false,
        },
        {
            key:         "base_url", 
            displayName: "Base URL",
            itemType:    itemTypeText,
            value:       cfg.GetBaseURL(), // e.g., "https://openrouter.ai/api/v1"
            modified:    false,
        },
        {
            key:         "api_key",
            displayName: "API Key",
            itemType:    itemTypeText,
            value:       cfg.GetAPIKey(), // Will be masked in display
            modified:    false,
        },
    },
}
```

**UI Interaction Flow:**

1. **Navigation:**
   - Tab/←→ keys switch between sections (Auto-Approval, Command Whitelist, LLM Settings)
   - ↑↓/jk keys navigate between items within a section
   - Visual indicator (➜) shows focused item

2. **Editing Text Items:**
   - User navigates to a text item (model, base URL, API key)
   - Presses Enter to open edit dialog
   - `showTextEditDialog()` creates modal with:
     - Current value pre-filled
     - Cursor indicator (▸)
     - Character count for fields with limits
     - Validation error messages (inline)
   - User types new value
   - Enter confirms, Esc cancels
   - Changes marked with asterisk (*) in main view

3. **Value Masking:**
   - API key field automatically detected (section.id == "llm" && item.key == "api_key")
   - Display value replaced with "••••••••" when not empty
   - Full value available for editing in dialog
   - Prevents shoulder surfing and accidental exposure

4. **Help Text Context:**
   - `buildHelpText()` shows different shortcuts based on focused item type:
     - Toggle items: "Space/Enter: Toggle"
     - Text items: "Enter: Edit"  
     - List items: "Enter/e: Edit"
   - Whitelist section adds: "a: Add, d: Delete"
   - Always shows: "Ctrl+S: Save, Esc/q: Close"

**Rendering Details:**

The settings overlay uses lipgloss for styling:
- **Section headers:** Mint green when not focused, salmon pink when focused
- **Item labels:** Muted gray normally, bright white + bold when focused
- **Toggle checkboxes:** `[ ]` unchecked, `[x]` checked (green)
- **Text values:** Muted gray value display, bright white when focused
- **Modified indicator:** Salmon pink asterisk (*)
- **Dialogs:** Rounded border, salmon pink border, dark background

**Data Persistence:**

When user saves settings (Ctrl+S):

1. `saveChanges()` method iterates through all sections
2. For LLM section, extracts item values by key:
   ```go
   data := make(map[string]interface{})
   for _, item := range section.items {
       data[item.key] = item.value
   }
   ```
3. Calls section's `SetData(data)` method
4. Config manager serializes to JSON via FileStore
5. Writes to `~/.forge/config.json` with 600 permissions
6. Visual confirmation: "✓ Settings saved" message

**Validation Integration:**

Each settings item can have validation logic:
- Model field: Non-empty string check
- Base URL field: Valid URL format or empty string allowed
- API key field: Optional validation (e.g., prefix check for sk-...)

Validation runs:
- On blur (when leaving field)
- Before save (all fields checked)
- Error messages shown inline in dialog

---

### File Locations (Platform-Specific)

**Unix/Linux:**
```
~/.config/forge/config.json  # XDG standard
~/.forge/config.json         # Fallback
```

**macOS:**
```
~/Library/Application Support/forge/config.json  # Preferred
~/.forge/config.json                              # Fallback
```

**Windows:**
```
%APPDATA%\forge\config.json  # Preferred
%USERPROFILE%\.forge\config.json  # Fallback
```

---

### Config File Schema

```json
{
  "version": "1.0",
  "sections": {
    "llm": {
      "model": "gpt-4-turbo",
      "base_url": "https://openrouter.ai/api/v1",
      "api_key": "sk-or-v1-..."
    },
    "auto_approval": {
      "read_file": true,
      "list_files": true
    },
    "command_whitelist": {
      "patterns": [
        "git status",
        "go test ./..."
      ]
    }
  }
}
```

**Field Descriptions:**
- `model` (string, required): LLM model identifier (e.g., "gpt-4-turbo", "claude-3-5-sonnet-20241022")
- `base_url` (string, optional): Custom API endpoint for OpenAI-compatible providers (empty string uses default)
- `api_key` (string, optional): API key for LLM provider (empty string uses OPENAI_API_KEY env var)

**Security Note:** The `api_key` field stores the API key in plaintext. File permissions should be 600 (owner read/write only) to protect this sensitive data.

---

### API Contract

```go
// pkg/config/llm.go
package config

type LLMSection struct {
    mu      sync.RWMutex
    model   string
    baseURL string
    apiKey  string
}

// Getters
func (s *LLMSection) GetModel() string
func (s *LLMSection) GetBaseURL() string
func (s *LLMSection) GetAPIKey() string

// Setters with validation
func (s *LLMSection) SetModel(model string) error
func (s *LLMSection) SetBaseURL(url string) error
func (s *LLMSection) SetAPIKey(key string) error

// Implements config.Section interface
func (s *LLMSection) ID() string
func (s *LLMSection) Title() string
func (s *LLMSection) Description() string
func (s *LLMSection) Data() map[string]interface{}
func (s *LLMSection) SetData(data map[string]interface{}) error
func (s *LLMSection) Validate() error
func (s *LLMSection) Reset()

// Settings UI integration
func (s *LLMSection) BuildItems() []settingsItem
```

**Implementation Details:**

```go
// Data() serializes section for JSON storage
func (s *LLMSection) Data() map[string]interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    return map[string]interface{}{
        "model":    s.model,
        "base_url": s.baseURL,
        "api_key":  s.apiKey,
    }
}

// SetData() deserializes from JSON storage
func (s *LLMSection) SetData(data map[string]interface{}) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if model, ok := data["model"].(string); ok {
        s.model = model
    }
    if baseURL, ok := data["base_url"].(string); ok {
        s.baseURL = baseURL
    }
    if apiKey, ok := data["api_key"].(string); ok {
        s.apiKey = apiKey
    }
    
    return s.Validate()
}

// Validate() checks field constraints
func (s *LLMSection) Validate() error {
    if s.model == "" {
        return fmt.Errorf("model cannot be empty")
    }
    
    // Base URL validation: must be valid URL or empty
    if s.baseURL != "" {
        if _, err := url.Parse(s.baseURL); err != nil {
            return fmt.Errorf("invalid base URL: %w", err)
        }
    }
    
    // API key: no validation (any string accepted)
    return nil
}

// BuildItems() creates settings UI items for this section
func (s *LLMSection) BuildItems() []settingsItem {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    return []settingsItem{
        {
            key:         "model",
            displayName: "Model Name",
            itemType:    itemTypeText,
            value:       s.model,
            modified:    false,
            helpText:    "The LLM model to use for code generation. Examples: gpt-4o, gpt-4-turbo, claude-3-5-sonnet-20241022",
        },
        {
            key:         "base_url",
            displayName: "Base URL",
            itemType:    itemTypeText,
            value:       s.baseURL,
            modified:    false,
            helpText:    "Custom API endpoint for OpenAI-compatible providers. Leave blank for default OpenAI endpoint.",
        },
        {
            key:         "api_key",
            displayName: "API Key",
            itemType:    itemTypeText,
            value:       s.apiKey,
            modified:    false,
            helpText:    "⚠️  WARNING: Storing API keys in config is less secure than environment variables. Recommended: Set OPENAI_API_KEY env var and leave this blank.",
        },
    }
}
```

---

## User Flows

### First-Time Setup Flow
```
User installs Forge
    ↓
Runs: forge
    ↓
Default model: gpt-4o
    ↓
Wants to use GPT-4 Turbo instead
    ↓
Types: /settings
    ↓
Settings overlay opens
    ↓
Navigates to "LLM Settings" section
    ↓
Sees:
  Model: gpt-4o
  Base URL: (empty)
  API Key: (empty)
    ↓
Navigates to "Model" field (arrow keys)
    ↓
Presses Enter to edit
    ↓
Types "gpt-4-turbo"
    ↓
Presses Enter to confirm
    ↓
Modified indicator (*) appears: Model: gpt-4-turbo *
    ↓
Presses Ctrl+S to save
    ↓
Settings saved to ~/.forge/config.json
    ↓
Provider client reconfigured with new model
    ↓
See: "✓ Settings saved and applied"
    ↓
Closes settings (Esc)
    ↓
Next message uses gpt-4-turbo immediately
    ↓
Restarts Forge tomorrow → Still uses gpt-4-turbo (persisted)
    ↓
Success - Changes applied instantly, persisted forever
```

---

### OpenRouter Setup Flow
```
User wants to use OpenRouter
    ↓
Opens /settings
    ↓
LLM Settings section
    ↓
Sets:
  Model: openai/gpt-4-turbo
  Base URL: https://openrouter.ai/api/v1
    ↓
Presses Ctrl+S to save
    ↓
Settings saved and applied to current session
    ↓
Sets env var: OPENAI_API_KEY=sk-or-v1-...
    ↓
Next message uses OpenRouter endpoint immediately (no restart)
    ↓
Can switch models via settings without changing base URL
    ↓
All model changes apply instantly to current session
```

---

### Mid-Session Model Switch Flow
```
User working with GPT-4 Turbo (saved in config)
    ↓
Wants to try Claude for current conversation
    ↓
Opens /settings (doesn't close Forge)
    ↓
LLM Settings section
    ↓
Changes Model: gpt-4-turbo → claude-3-5-sonnet-20241022
    ↓
Presses Ctrl+S to save
    ↓
Settings saved and applied immediately
    ↓
Closes settings (Esc)
    ↓
Next message uses Claude instantly (no restart, context preserved)
    ↓
Completes task with Claude
    ↓
Opens /settings again
    ↓
Switches back to GPT-4 Turbo
    ↓
Saves (Ctrl+S) - applied immediately
    ↓
Continues working with GPT-4 Turbo
    ↓
Entire workflow completed without restart or losing context
```

---

## Design Decisions

### 1. API Key Storage Approach

**Decision:** Support optional API key storage in config file with security warnings

**Options Considered:**
- **Option A:** Environment variable only (no config storage)
  - Pros: Most secure, industry standard
  - Cons: Inconvenient for local dev, requires shell config
- **Option B:** Config file only (deprecated env vars)
  - Pros: Simple, one place to configure
  - Cons: Security risk, files can be committed to git
- **Option C (Selected):** Both supported with clear precedence
  - Pros: Flexibility for different use cases, clear migration path
  - Cons: More complex to implement and document

**Rationale:** Users have different security requirements. Production deployments should use env vars, but local development benefits from convenience of config storage. Clear warnings and documentation help users make informed choices.

---

### 2. Value Masking for API Keys

**Decision:** Mask API key display in settings UI as ••••••••

**Implementation:**
- Detection logic: `section.id == "llm" && item.key == "api_key"`
- Display value replaced with bullet characters when not empty
- Full value visible in edit dialog for editing
- Applied in `settingsModel.View()` rendering logic

**Rationale:** Prevents shoulder surfing and accidental exposure when screen sharing. Common pattern in password/credential UIs. Users still need to see actual value for editing, so edit dialog shows full text.

---

### 3. Model Validation Strategy

**Decision:** Allow any string for model name, no validation against known list

**Rationale:**
- New models released frequently (can't maintain exhaustive list)
- Users may use custom fine-tuned models
- Provider-specific model naming varies (OpenRouter uses prefixes)
- Simple validation: non-empty string only
- Future enhancement: warn for unknown models, suggest similar names

---

### 4. Settings Section Architecture

**Decision:** Single "LLM Settings" section with three text fields

**Structure:**
```
LLM Settings (section)
  ├── Model Name (itemTypeText)
  ├── Base URL (itemTypeText)
  └── API Key (itemTypeText, masked)
```

**Rationale:** 
- Fits existing settings overlay architecture
- All related configuration in one place
- Text fields appropriate for all three values
- Future enhancement: dropdown for Model Name with popular choices

---

### 5. Configuration Precedence

**Decision:** CLI flags > Environment variables > Config file > Defaults

**Rationale:**
- CLI flags: Temporary overrides for one-off tasks (applies for that session only)
- Env vars: Deployment-specific configuration (CI/CD, containers)
- Config file: User preferences, persistent defaults (apply immediately when saved via /settings)
- Hardcoded defaults: Sensible fallback (gpt-4o)

This matches industry standards (Git, Docker, AWS CLI) and user expectations.

**Important Note:** Settings changed via `/settings` UI apply immediately to the current session AND persist to the config file. CLI flags remain temporary overrides that don't affect saved settings.

---

## Open Questions

1. **Provider Presets (P1 Feature):** Should we include provider presets in v1.0 or defer to v1.1?
   - **Recommendation:** Defer to v1.1 - adds complexity, not critical for MVP
   
2. **Connection Testing (P1 Feature):** Include "Test Connection" button in settings UI?
   - **Recommendation:** Defer to v1.1 - useful but not essential for basic configuration
   
3. **Model Dropdown (P1 Feature):** Text input or dropdown with popular models?
   - **Recommendation:** Text input for v1.0, dropdown for v1.1 when we have static model list
   
4. **Default Model:** Keep `gpt-4o` as default, or use `gpt-4-turbo`?
   - **Recommendation:** Keep `gpt-4o` - faster, cheaper, good balance for most users

5. **Config File Location:** Use XDG standard (`~/.config/forge/`) or simple (`~/.forge/`)?
   - **Recommendation:** Support both with precedence: XDG first, fallback to ~/.forge/

---

## Implementation Plan

### Phase 1: Core LLM Section (Week 1)

**Goal:** Create LLMSection backend with persistence

**Tasks:**
1. Create `pkg/config/llm.go` with LLMSection struct
2. Implement config.Section interface methods:
   - `ID()`, `Title()`, `Description()`
   - `Data()`, `SetData()`, `Validate()`, `Reset()`
3. Add getters/setters: `GetModel()`, `SetModel()`, `GetAPIKey()`, etc.
4. Register LLMSection with config Manager in initialization
5. Write unit tests for all methods
6. Test JSON serialization/deserialization

**Acceptance Criteria:**
- LLMSection implements config.Section interface
- Data persists to/from JSON correctly
- Validation prevents empty model names and invalid URLs
- Thread-safe with mutex protection
- 100% test coverage for section logic

---

### Phase 2: Settings UI Integration (Week 2)

**Goal:** Add LLM Settings section to TUI overlay

**Tasks:**
1. Update `buildSections()` in `pkg/executor/tui/overlay/settings.go`:
   - Create LLM section with three items (model, base_url, api_key)
   - Set item types to `itemTypeText` for all fields
2. Implement API key masking logic in `View()` rendering:
   - Detect LLM section + api_key field
   - Replace value with ••••••••• for display
   - Preserve full value for edit dialog
3. Update `buildHelpText()` to show appropriate shortcuts for text items
4. Test navigation, editing, and saving:
   - Tab between sections
   - Arrow keys between items
   - Enter to edit, Esc to cancel
   - Ctrl+S to save
5. Add inline help text for each field
6. Test unsaved changes dialog

**Acceptance Criteria:**
- LLM Settings section appears in settings overlay
- Three text fields editable via Enter key
- API key masked in main view but visible in edit dialog
- Help text updates based on focused field
- Modified indicator (*) appears on changes
- Ctrl+S saves to config file
- Settings persist across restarts

---

### Phase 3: Startup Integration (Week 3)

**Goal:** Load LLM settings at startup with proper precedence

**Tasks:**
1. Update `parseFlags()` in main.go:
   - Load config file to get defaults
   - Check environment variables (FORGE_MODEL, OPENAI_API_KEY, OPENAI_BASE_URL)
   - Define CLI flags with precedence-based defaults
   - Log which source was used (debug mode)
2. Update provider initialization to use resolved config:
   - Pass model, base URL, API key to provider
   - Handle empty API key (prompt user or error)
3. Add config file permission checks:
   - Verify 600 permissions on load
   - Warn if too permissive
   - Set 600 on save
4. Integration tests for precedence:
   - CLI flag overrides env var
   - Env var overrides config file
   - Config file overrides default
   - Each combination tested

**Acceptance Criteria:**
- Config file loads at startup if exists
- Precedence order enforced: CLI > ENV > Config > Default
- Empty config creates file with defaults on first save
- File permissions enforced (600)
- Debug logging shows configuration source
- Works on all platforms (Windows, macOS, Linux)

---

### Phase 4: Documentation & Polish (Week 4)

**Goal:** Complete documentation and user-facing polish

**Tasks:**
1. Update documentation:
   - README.md: Add LLM Settings section
   - docs/how-to/configure-provider.md: Settings UI instructions
   - In-app help text: Review and refine all field descriptions
   - Security warnings: Emphasize env var best practice
2. Add example configurations:
   - OpenRouter setup guide
   - Local LLM (Ollama) setup guide
   - Azure OpenAI setup guide
3. User acceptance testing:
   - First-time setup flow
   - Model switching flow
   - API key configuration
   - Precedence testing (CLI override)
4. Bug fixes and edge cases:
   - Invalid URL handling
   - Config file corruption recovery
   - Permission denied errors
   - Cross-platform path issues
5. Performance testing:
   - Large config files
   - Concurrent access
   - Startup time impact

**Acceptance Criteria:**
- All documentation updated and reviewed
- Example configs tested and working
- Known edge cases handled gracefully
- Zero critical bugs
- User flows documented with screenshots
- Performance benchmarks acceptable

---

### Total Timeline: 3-4 Weeks

**Week 1:** Backend implementation and unit tests  
**Week 2:** UI integration and manual testing  
**Week 3:** Startup integration and precedence logic  
**Week 4:** Documentation, polish, and release prep

---

## Dependencies

**Required:**
- Existing config system (pkg/config/)
- Settings TUI overlay (pkg/executor/tui/overlay/settings.go)
- FileStore persistence (pkg/config/store.go)

**Optional (Future):**
- Keyring library for encrypted API key storage
- LLM provider APIs for model discovery

---

## Testing Strategy

**Unit Tests:**
- LLMSection CRUD operations
- Validation logic (model names, URLs)
- Precedence resolution (CLI > ENV > Config > Default)

**Integration Tests:**
- End-to-end config loading at startup
- Settings UI save/load cycle
- CLI flag override behavior
- Config file corruption recovery

**Manual Testing:**
- First-time setup flow
- OpenRouter configuration
- Local LLM (Ollama) configuration
- Multiple environment scenarios
- Cross-platform file locations

---

## Future Enhancements

**Phase 2 (v1.2):**
- Model dropdown with popular choices (instant switching)
- Provider presets (OpenRouter, Ollama) with live preview
- Connection testing from settings (test without committing)
- Model profiles (save multiple configs, switch instantly between them)

**Phase 3 (v1.3+):**
- Advanced LLM parameters (temperature, max tokens) with live updates
- Cost tracking and budget alerts
- Multi-provider support with provider-specific settings (instant switching)
- Encrypted API key storage via OS keyring

**Long-term:**
- Model performance comparison
- Automatic model selection based on task type
- Team-wide model policies and governance
- Usage analytics and recommendations
