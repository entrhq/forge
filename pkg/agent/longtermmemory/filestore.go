package longtermmemory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var ErrNotFound = errors.New("longtermmemory: memory not found")
var ErrAlreadyExists = errors.New("longtermmemory: memory already exists")

// FileStore is a local file-system implementation of MemoryStore.
// It stores memory files as Markdown files with YAML front-matter
// separated across repository and user scopes.
type FileStore struct {
	repoDir string
	userDir string
}

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
	if strings.ContainsAny(id, "/\\") {
		return "", fmt.Errorf("longtermmemory: invalid memory id %q (contains path separator)", id)
	}
	resolved := filepath.Join(dir, id+".md")
	if !strings.HasPrefix(resolved, dir+string(filepath.Separator)) {
		return "", fmt.Errorf("longtermmemory: path traversal detected for id %q", id)
	}
	return resolved, nil
}

// Write persists a new memory file to disk. It writes atomically via a
// temporary file, ensuring append-only behavior by returning ErrAlreadyExists
// if the given ID is already present on disk. It must be safe for concurrent use.
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
		return ErrAlreadyExists
	}
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

// Read attempts to retrieve a memory file by ID, searching first in
// ScopeRepo, then ScopeUser. It returns ErrNotFound if it does not exist.
func (fs *FileStore) Read(_ context.Context, id string) (*MemoryFile, error) {
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

// List returns all valid memory files from all configured scopes.
// Corrupt or unreadable files are skipped automatically.
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

// ListByScope returns all valid memory files within a specific scope.
// Corrupt or unreadable files are skipped automatically.
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
