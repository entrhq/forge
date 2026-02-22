# Long-Term Memory Package

This package implements the foundational data layer for the agent's cross-session long-term memory. It provides a filesystem-backed storage engine for creating, retrieving, and versioning memories. The full architectural design can be found in [ADR 0044](../../../docs/adr/0044-long-term-memory-storage.md).

## Overview

The system is designed as a content-addressable store where each memory is an immutable file, and new versions are created by creating new files that supersede previous ones.

## File Format

Memories are stored as files with YAML front-matter for metadata and a Markdown body for the content. This hybrid format is human-readable and easy to parse.

```yaml
---
id: mem_1H8Z7j3fR9x...
supersedes:
  - mem_1H8YpLm9w4q...
references:
  - doc_1H8XoEa7bCv...
scope: default
timestamp: "2024-05-20T12:00:00Z"
---

This is the Markdown body of the memory, containing the core content.
```

- **`MemoryMeta` (YAML)**: Contains `id`, versioning (`supersedes`), cross-references (`references`), `scope`, and `timestamp`.
- **Body (Markdown)**: The unstructured text of the memory itself.

## Core Components

### Storage Engine (`filestore.go`)

The `FileStore` implements the `MemoryStore` interface and provides the core CRUD operations for memories.

- **Atomic Writes**: Operations are atomic, using temporary files and atomic renames to prevent data corruption.
- **Security**: Files are created with `0o600` permissions and directories with `0o750` to ensure memory privacy.
- **Append-Only Design**: The store is effectively append-only. Once a memory is written, it is immutable. Updates are handled by creating new, versioned memories.

### Versioning (`version.go`)

- **`VersionChain`**: Provides a mechanism to traverse the history of a memory from the latest version back to the oldest.
- **Cycle Detection**: The versioning logic includes safeguards to detect and prevent infinite loops in case of a corrupted version history (e.g., `A -> B -> A`).