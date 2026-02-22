# 0044. Long-Term Memory — Storage Format & Data Model

**Status:** Accepted
**Date:** 2025-02-22
**Deciders:** Core Team
**Technical Story:** [Long-Term Persistent Memory System PRD](../product/features/long-term-memory.md)

---

## Context

### Background

Forge currently has two in-session memory systems: `ConversationMemory` (ADR-0007), which maintains the message history for context management, and the agent scratchpad (ADR-0032), which gives the agent working notes within a session. Neither persists across sessions.

This ADR defines the foundational data layer for cross-session long-term memory — the file format, directory layout, Go types, and storage interface that the capture pipeline (ADR-0046) and retrieval engine (ADR-0047) both depend on. It does not implement any LLM interaction.

### Problem Statement

Cross-session memories need a storage representation that is:
- **Human-readable and editable** — users must be able to inspect and modify memories without special tooling
- **Machine-parseable** — the Go runtime must be able to read, write, and query memory files efficiently
- **Graph-aware** — memories reference other memories via typed relationship edges; the format must carry these edges explicitly
- **Versionable** — old memories are never deleted; updated memories form a linear version chain via `supersedes` links
- **Dual-scoped** — memories are stored at either repo scope (`.forge/memory/`) or user scope (`~/.forge/memory/`) depending on their category

### Goals

- Define the canonical memory file format (YAML front-matter + markdown body)
- Define all schema fields, category constants, and relationship edge types
- Define the `MemoryFile` Go struct and its serialization/parsing
- Define the `MemoryStore` interface and a `FileStore` implementation
- Establish version chain semantics (supersedes, never-delete)
- Implement UUID-based file naming and directory initialisation

### Non-Goals

- LLM classifier integration (ADR-0046)
- Embedding or retrieval (ADR-0045, ADR-0047)
- TUI management commands (P2 / future)
- Memory consolidation (stretch / future)

---

## Decision Drivers

* Human auditability — users must be able to read, edit, and delete memories using only a text editor and their filesystem
* Agent reasoning — the agent must be able to traverse version chains and understand relationship types programmatically
* Simplicity of implementation — the format must be parseable with standard Go libraries, no bespoke serialization
* Durability — memories must survive agent crashes, restarts, and concurrent writes without corruption
* Extensibility — the schema must accommodate future fields without breaking existing files

---

## Considered Options

### Option 1: YAML front-matter + Markdown body

Each memory is a single `.md` file. Structured metadata lives in a fenced YAML block at the top of the file. Free-form memory content is written as plain markdown below.

**Pros:**
- Directly editable by users with any text editor
- Metadata queryable without parsing markdown
- Markdown body allows rich content (code blocks, lists, rationale sections) for complex memories
- Standard format familiar to developers; rendered nicely in GitHub, VS Code, etc.
- Front-matter pattern is established in the codebase (AGENTS.md context injection)

**Cons:**
- Parser must handle both YAML and markdown; two parse steps
- Front-matter boundary (`---`) must be robustly detected

### Option 2: Pure YAML files

Each memory is a `.yaml` file with all content in YAML fields.

**Pros:**
- Single parse step
- Strongly typed

**Cons:**
- Multi-line prose in YAML is awkward to read and edit (block scalars)
- No markdown rendering; rich content (code examples, rationale) is unpleasant
- Users cannot edit naturally

### Option 3: JSON files

**Pros:**
- Native Go marshal/unmarshal
- Tooling support

**Cons:**
- Not human-friendly to edit
- Multi-line content requires escaping
- No clear advantage over YAML for this use case

---

## Decision

**Chosen Option:** Option 1 — YAML front-matter + Markdown body

### Rationale

User auditability is a first-class requirement. The YAML + markdown pattern is the most natural format for content that must be both machine-queryable (front-matter) and human-readable (body). It is already familiar to the target audience, and the hybrid approach gives the best of both worlds: structured metadata for the agent, readable prose for the user.

---

## Consequences

### Positive

- Users can inspect, edit, and delete memories with standard tools
- Front-matter fields are directly queryable without full file parse for index operations
- Markdown body can capture rich rationale, code snippets, and context alongside atomic facts
- Version chains are trivially auditable by following `supersedes` links in a file explorer

### Negative

- Two-step parsing (YAML front-matter + markdown body) adds minor implementation complexity
- Front-matter parsing must be robust against files edited by humans (trailing whitespace, missing fields)

### Neutral

- Memory files are append-only at the store level (new file per version); disk usage grows over time, bounded by future consolidation feature

---

## Implementation

### File Naming

Memory files are named by their UUID with a `.md` extension:

```
mem_<uuid>.md
```

UUID generation uses `crypto/rand` to produce a standard UUID v4. The `mem_` prefix ensures the filename is unambiguous when listed alongside other files in `.forge/`.

```go
// pkg/agent/longtermmemory/id.go
package longtermmemory

import (
    "crypto/rand"
    "fmt"
)

// NewMemoryID generates a new unique memory identifier.
func NewMemoryID() string {
    b := make([]byte, 16)
    _, _ = rand.Read(b)
    b[6] = (b[6] & 0x0f) | 0x40 // version 4
    b[8] = (b[8] & 0x3f) | 0x80 // variant bits
    return fmt.Sprintf("mem_%08x-%04x-%04x-%04x-%012x",
        b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
```

### Directory Layout

```
# Repo-scoped memories
.forge/memory/
    mem_abc12345-...md
    mem_def67890-...md

# User-scoped memories
~/.forge/memory/
    mem_xyz11111-...md
    mem_uvw22222-...md
```

Both directories are created on first write if they do not exist (`os.MkdirAll`).

### YAML Front-matter Schema

```yaml
---
id: mem_<uuid>
created_at: 2025-01-15T10:30:00Z
updated_at: 2025-01-15T14:22:00Z
version: 1
scope: repo
category: coding-preferences
supersedes: null
related:
  - id: mem_<uuid>
    relationship: refines
session_id: <session-uuid>
trigger: cadence
---
```

All timestamps use RFC 3339 UTC. `supersedes` is `null` for the first version of any memory, and the ID of the memory it replaces for all subsequent versions.

### Go Types

```go
// pkg/agent/longtermmemory/types.go
package longtermmemory

import "time"

// Scope determines where a memory file is stored.
type Scope string

const (
    ScopeRepo Scope = "repo"
    ScopeUser Scope = "user"
)

// Category classifies the kind of information a memory encodes.
type Category string

const (
    CategoryCodingPreferences     Category = "coding-preferences"
    CategoryProjectConventions    Category = "project-conventions"
    CategoryArchitecturalDecisions Category = "architectural-decisions"
    CategoryUserFacts             Category = "user-facts"
    CategoryCorrections           Category = "corrections"
    CategoryPatterns              Category = "patterns"
)

// Relationship describes the directed edge type between two memories.
type Relationship string

const (
    RelationshipSupersedes  Relationship = "supersedes"
    RelationshipRefines     Relationship = "refines"
    RelationshipContradicts Relationship = "contradicts"
    RelationshipRelatesTo   Relationship = "relates-to"
)

// Trigger records what caused the capture pass that created this memory.
type Trigger string

const (
    TriggerCadence    Trigger = "cadence"
    TriggerCompaction Trigger = "compaction"
)

// RelatedMemory is a typed edge to another memory node.
type RelatedMemory struct {
    ID           string       `yaml:"id"`
    Relationship Relationship `yaml:"relationship"`
}

// MemoryMeta holds all YAML front-matter fields.
type MemoryMeta struct {
    ID         string          `yaml:"id"`
    CreatedAt  time.Time       `yaml:"created_at"`
    UpdatedAt  time.Time       `yaml:"updated_at"`
    Version    int             `yaml:"version"`
    Scope      Scope           `yaml:"scope"`
    Category   Category        `yaml:"category"`
    // Supersedes is nil for the first version of a memory and non-nil for all
    // subsequent versions. Using *string with omitempty serialises cleanly:
    // the key is absent when nil rather than `supersedes: ""` or `supersedes: null`.
    // Callers check `m.Meta.Supersedes != nil` rather than `!= ""`.
    Supersedes *string         `yaml:"supersedes,omitempty"`
    Related    []RelatedMemory `yaml:"related"`
    SessionID  string          `yaml:"session_id"`
    Trigger    Trigger         `yaml:"trigger"`
}

// MemoryFile is the fully parsed in-memory representation of a memory file.
type MemoryFile struct {
    Meta    MemoryMeta
    Content string // raw markdown body
}
```

### Serialization

The parser splits on the `---` front-matter delimiter, unmarshals the YAML block, and returns the remaining content as the markdown body. The serializer writes YAML front-matter followed by the markdown body.

```go
// pkg/agent/longtermmemory/parse.go
package longtermmemory

import (
    "fmt"
    "strings"

    "gopkg.in/yaml.v3"
)

const frontMatterDelimiter = "---"

// Parse deserializes a raw memory file byte slice into a MemoryFile.
// Returns an error if the front-matter block is missing or malformed.
func Parse(raw []byte) (*MemoryFile, error) {
    s := string(raw)
    if !strings.HasPrefix(s, frontMatterDelimiter) {
        return nil, fmt.Errorf("longtermmemory: missing front-matter delimiter")
    }
    // Find the closing delimiter
    rest := s[len(frontMatterDelimiter):]
    idx := strings.Index(rest, "\n"+frontMatterDelimiter)
    if idx == -1 {
        return nil, fmt.Errorf("longtermmemory: unclosed front-matter block")
    }
    yamlBlock := rest[:idx]
    
    // Support parsing bodies with single or double newline spacing after the YAML block
    bodyRaw := rest[idx+len("\n"+frontMatterDelimiter):]
    body := strings.TrimPrefix(bodyRaw, "\n")
    if strings.HasPrefix(body, "\n") {
        body = body[1:]
    }

    var meta MemoryMeta
    if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
        return nil, fmt.Errorf("longtermmemory: front-matter parse error: %w", err)
    }
    return &MemoryFile{Meta: meta, Content: body}, nil
}

// Serialize renders a MemoryFile back to its on-disk byte representation.
func Serialize(m *MemoryFile) ([]byte, error) {
    yamlBytes, err := yaml.Marshal(&m.Meta)
    if err != nil {
        return nil, fmt.Errorf("longtermmemory: serialize error: %w", err)
    }
    var sb strings.Builder
    sb.WriteString(frontMatterDelimiter + "\n")
    sb.Write(yamlBytes)
    sb.WriteString(frontMatterDelimiter + "\n\n")
    sb.WriteString(m.Content)
    return []byte(sb.String()), nil
}
```

### MemoryStore Interface

```go
// pkg/agent/longtermmemory/store.go
package longtermmemory

import "context"

// MemoryStore is the read/write interface for persisted memory files.
// Implementations must be safe for concurrent use.
type MemoryStore interface {
    // Write persists a new MemoryFile to the appropriate scope directory.
    // The caller is responsible for setting all Meta fields before calling Write.
    Write(ctx context.Context, m *MemoryFile) error

    // Read returns the MemoryFile for the given ID, or ErrNotFound if absent.
    Read(ctx context.Context, id string) (*MemoryFile, error)

    // List returns all MemoryFiles from both scope directories.
    // The order of results is not guaranteed.
    List(ctx context.Context) ([]*MemoryFile, error)

    // ListByScope returns all MemoryFiles for a specific scope.
    ListByScope(ctx context.Context, scope Scope) ([]*MemoryFile, error)
}
```

### FileStore Implementation

`FileStore` implements `MemoryStore` backed by the filesystem.

```go
// pkg/agent/longtermmemory/filestore.go
package longtermmemory

import (
    "context"
    "errors"
    "fmt"
    "os"
    "path/filepath"
)

// ErrNotFound is returned when a requested memory ID does not exist.
var ErrNotFound = errors.New("longtermmemory: memory not found")

// ErrAlreadyExists is returned when trying to overwrite an existing memory ID.
var ErrAlreadyExists = errors.New("longtermmemory: memory already exists")

// FileStore is a MemoryStore backed by the local filesystem.
type FileStore struct {
    repoDir string // absolute path to .forge/memory/
    userDir string // absolute path to ~/.forge/memory/
}

// NewFileStore creates a FileStore rooted at the given directories,
// creating them if they do not exist.
func NewFileStore(repoDir, userDir string) (*FileStore, error) {
    for _, dir := range []string{repoDir, userDir} {
        if err := os.MkdirAll(dir, 0o750); err != nil {
            return nil, fmt.Errorf("longtermmemory: init directory %s: %w", dir, err)
        }
    }
    return &FileStore{repoDir: repoDir, userDir: userDir}, nil
}

func (fs *FileStore) dirForScope(scope Scope) (string, error) {
    switch scope {
    case ScopeRepo:
        return fs.repoDir, nil
    case ScopeUser:
        return fs.userDir, nil
    default:
        return "", fmt.Errorf("longtermmemory: unknown scope %q", scope)
    }
}

// pathForID returns the absolute path for the given memory ID within the
// appropriate scope directory. It validates that the resolved path actually
// falls inside the expected directory, guarding against path traversal from
// user-edited or classifier-generated IDs containing `../` sequences.
func (fs *FileStore) pathForID(id string, scope Scope) (string, error) {
    if id == "" {
        return "", fmt.Errorf("longtermmemory: invalid memory id (empty)")
    }
    scopeDir, err := fs.dirForScope(scope)
    if err != nil {
        return "", err
    }
    dir, err := filepath.Abs(scopeDir)
    if err != nil {
        return "", fmt.Errorf("longtermmemory: abs dir: %w", err)
    }
    // Reject ids that embed a path separator before Abs can canonicalise them.
    if strings.ContainsAny(id, "/\\") {
        return "", fmt.Errorf("longtermmemory: invalid memory id %q (contains path separator)", id)
    }
    resolved := filepath.Join(dir, id+".md")
    if !strings.HasPrefix(resolved, dir+string(filepath.Separator)) {
        return "", fmt.Errorf("longtermmemory: path traversal detected for id %q", id)
    }
    return resolved, nil
}

// Write persists a new MemoryFile atomically if one does not already exist.
func (fs *FileStore) Write(_ context.Context, m *MemoryFile) error {
    b, err := Serialize(m)
    if err != nil {
        return err
    }
    path, err := fs.pathForID(m.Meta.ID, m.Meta.Scope)
    if err != nil {
        return err
    }
    if _, err := os.Stat(path); err == nil {
        return ErrAlreadyExists // append-only safety guard
    }
    // Write atomically via temp file + rename
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, b, 0o600); err != nil {
        return fmt.Errorf("longtermmemory: write temp file: %w", err)
    }
    if err := os.Rename(tmp, path); err != nil {
        _ = os.Remove(tmp) // best-effort cleanup
        return fmt.Errorf("longtermmemory: atomic rename %s: %w", path, err)
    }
    return nil
}

func (fs *FileStore) Read(_ context.Context, id string) (*MemoryFile, error) {
    // Try repo scope first, then user scope
    for _, scope := range []Scope{ScopeRepo, ScopeUser} {
        path, err := fs.pathForID(id, scope)
        if err != nil {
            return nil, err
        }
        b, err := os.ReadFile(path)
        if errors.Is(err, os.ErrNotExist) {
            continue
        }
        if err != nil {
            return nil, fmt.Errorf("longtermmemory: read %s: %w", path, err)
        }
        return Parse(b)
    }
    return nil, ErrNotFound
}

func (fs *FileStore) List(ctx context.Context) ([]*MemoryFile, error) {
    repo, err := fs.ListByScope(ctx, ScopeRepo)
    if err != nil {
        return nil, err
    }
    user, err := fs.ListByScope(ctx, ScopeUser)
    if err != nil {
        return nil, err
    }
    return append(repo, user...), nil
}

func (fs *FileStore) ListByScope(_ context.Context, scope Scope) ([]*MemoryFile, error) {
    dir, err := fs.dirForScope(scope)
    if err != nil {
        return nil, err
    }
    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, fmt.Errorf("longtermmemory: list %s: %w", dir, err)
    }
    var out []*MemoryFile
    for _, e := range entries {
        if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
            continue
        }
        filePath := filepath.Join(dir, e.Name())
        b, err := os.ReadFile(filePath)
        if err != nil {
            slog.Debug("longtermmemory: skipping unreadable memory file", "path", filePath, "err", err)
            continue
        }
        m, err := Parse(b)
        if err != nil {
            slog.Debug("longtermmemory: skipping corrupt memory file", "path", filePath, "err", err)
            continue
        }
        out = append(out, m)
    }
    return out, nil
}
```

### Version Chain Helpers

```go
// pkg/agent/longtermmemory/version.go
package longtermmemory

import (
    "context"
    "fmt"
)

// NewVersion creates a new MemoryFile that supersedes the given predecessor.
// The caller sets the Content field on the returned file before writing.
func NewVersion(predecessor *MemoryFile, sessionID string, trigger Trigger) *MemoryFile {
    now := timeNow() // injected for testability
    predecessorID := predecessor.Meta.ID
    return &MemoryFile{
        Meta: MemoryMeta{
            ID:         NewMemoryID(),
            CreatedAt:  now,
            UpdatedAt:  now,
            Version:    predecessor.Meta.Version + 1,
            Scope:      predecessor.Meta.Scope,
            Category:   predecessor.Meta.Category,
            Supersedes: &predecessorID, // *string; nil means "first version"
            Related:    predecessor.Meta.Related, // carry forward existing edges
            SessionID:  sessionID,
            Trigger:    trigger,
        },
    }
}

// VersionChain follows the supersedes chain BACKWARD from the given ID,
// following Supersedes links to predecessors, and returns the full ordered
// history oldest-first.
//
// IMPORTANT — direction: callers must pass the LATEST known version ID to
// obtain the full history. VersionChain cannot discover newer versions from
// an older one; it only walks toward the root. In practice the retrieval
// engine already holds the latest version (it was found by embedding search),
// so this is not a problem. If you need to locate the head of an arbitrary
// chain, call LatestVersion first.
//
// Stops at maxDepth hops to guard against cycles in user-edited files.
func VersionChain(ctx context.Context, store MemoryStore, id string, maxDepth int) ([]*MemoryFile, error) {
    var chain []*MemoryFile
    current := id
    for i := 0; i < maxDepth && current != ""; i++ {
        m, err := store.Read(ctx, current)
        if err != nil {
            return chain, fmt.Errorf("longtermmemory: version chain read %s: %w", current, err)
        }
        chain = append([]*MemoryFile{m}, chain...) // prepend so oldest is first
        if m.Meta.Supersedes == nil {
            break // reached the root of the chain
        }
        current = *m.Meta.Supersedes
    }
    return chain, nil
}

// LatestVersion resolves the head of a version chain given any member ID.
// It scans all memories and follows Supersedes links forward. O(n) over the
// full store — use sparingly (e.g., user-facing audit commands, not hot paths).
// Returns the input id unchanged if no newer version exists.
func LatestVersion(ctx context.Context, store MemoryStore, id string) (string, error) {
    all, err := store.List(ctx)
    if err != nil {
        return id, err
    }
    // Build reverse map: predecessorID -> successorID
    successor := make(map[string]string, len(all))
    for _, m := range all {
        if m.Meta.Supersedes != nil {
            successor[*m.Meta.Supersedes] = m.Meta.ID
        }
    }
    current := id
    visited := make(map[string]bool)
    for !visited[current] {
        visited[current] = true
        next, ok := successor[current]
        if !ok {
            return current, nil
        }
        current = next
    }
    return current, nil
}
```

### Package Layout

```
pkg/agent/longtermmemory/
    id.go          — NewMemoryID()
    types.go       — all types and constants
    parse.go       — Parse() and Serialise()
    store.go       — MemoryStore interface
    filestore.go   — FileStore implementation
    version.go     — NewVersion(), VersionChain()
    longtermmemory.go — package doc and init helpers
```

### Migration Path

No migration is required. The `.forge/memory/` and `~/.forge/memory/` directories are new; they do not conflict with any existing Forge state.

### Timeline

This ADR is the prerequisite for ADR-0045 (embedding provider), ADR-0046 (capture), and ADR-0047 (retrieval). It should be implemented and merged first.

---

## Validation

### Success Metrics

- `Parse(Serialize(m))` round-trip is lossless for all field types
- `FileStore.Write` is atomic (temp file + rename, no partial writes visible to readers)
- `FileStore.Read` returns `ErrNotFound` (not a generic error) for missing IDs
- `FileStore.ListByScope` skips corrupt files rather than failing the whole list
- `VersionChain` terminates within `maxDepth` hops even if a user creates a cycle in front-matter

### Monitoring

- Debug-level log on corrupt file skip in `ListByScope`
- Unit tests cover: round-trip serialization, missing delimiter, unclosed front-matter, atomic write, all scope combinations, version chain traversal, cycle guard

---

## Related Decisions

- [ADR-0007](0007-memory-system-design.md) — in-session conversation memory (not replaced; complementary)
- [ADR-0032](0032-agent-scratchpad-notes-system.md) — in-session scratchpad notes (not replaced; complementary)
- [ADR-0045](0045-long-term-memory-embedding-provider.md) — embedding provider extension (reads MemoryStore.List)
- [ADR-0046](0046-long-term-memory-capture.md) — capture pipeline (writes via MemoryStore)
- [ADR-0047](0047-long-term-memory-retrieval.md) — retrieval engine (reads via MemoryStore)

---

## References

- [Long-Term Memory PRD](../product/features/long-term-memory.md)
- [YAML front-matter spec](https://yaml.org/spec/1.2.2/)
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)

---

## Notes

The `timeNow` variable in `version.go` is a package-level `var` defaulting to `time.Now`, overridden in tests for deterministic timestamps. This is an established pattern in the codebase.

**Last Updated:** 2025-02-22
