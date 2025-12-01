// Package scratchpad provides tools for managing agent scratchpad notes.
//
// The scratchpad is a lightweight note-taking system that allows agents to:
//   - Record important information during task execution
//   - Organize notes with tags for easy retrieval
//   - Search and filter notes by content and tags
//   - Mark notes as addressed (scratched) without losing context
//
// Tool Overview:
//
// add_note: Create a new note with content (max 800 chars) and 1-5 tags
//
// search_notes: Search notes by content query and/or tags with relevance ranking
//
// list_notes: List all notes with optional tag filtering and scratched status
//
// list_tags: List all unique tags currently in use across active notes
//
// update_note: Update a note's content and/or tags by ID
//
// scratch_note: Mark a note as addressed/scratched to indicate it's been handled
//
// delete_note: Permanently remove a note (prefer scratch_note to preserve context)
//
// Usage Example:
//
//	// Create a notes manager
//	manager := notes.NewManager()
//
//	// Initialize scratchpad tools
//	addTool := scratchpad.NewAddNoteTool(manager)
//	searchTool := scratchpad.NewSearchNotesTool(manager)
//	listTool := scratchpad.NewListNotesTool(manager)
//
//	// Register with agent's tool registry
//	registry.Register(addTool)
//	registry.Register(searchTool)
//	registry.Register(listTool)
//
// Design Principles:
//
//   - Simple, focused operations aligned with note management primitives
//   - Consistent error handling and validation across all tools
//   - Rich metadata in responses for observability and debugging
//   - Non-blocking operations that don't interrupt the agent loop
//   - Support for both structured (tags) and unstructured (content) search
package scratchpad
