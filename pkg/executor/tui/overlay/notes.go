package overlay

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/executor/tui/types"
	pkgtypes "github.com/entrhq/forge/pkg/types"
)

// convertNoteData converts pkg/types.NoteData to tui/types.NoteData
func convertNoteData(note *pkgtypes.NoteData) *types.NoteData {
	if note == nil {
		return nil
	}
	return &types.NoteData{
		ID:        note.ID,
		Content:   note.Content,
		Tags:      note.Tags,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
		Scratched: note.Scratched,
	}
}

// noteListItem represents a single note in the list
type noteListItem struct {
	note        *pkgtypes.NoteData
	tuiNoteData *types.NoteData // Pre-converted for ViewNoteMsg
}

func (i noteListItem) FilterValue() string {
	return i.note.Content
}

func (i noteListItem) Title() string {
	tags := strings.Join(i.note.Tags, ", ")
	return fmt.Sprintf("[%s]", tags)
}

func (i noteListItem) Description() string {
	// Truncate content if too long (77 chars + 3 for "..." = 80 total)
	content := strings.ReplaceAll(i.note.Content, "\n", " ")
	if len(content) > 77 {
		content = content[:77] + "..."
	}
	return content
}

// noteListDelegate is a custom delegate for rendering note list items
type noteListDelegate struct {
	list.DefaultDelegate
}

func newNoteListDelegate() noteListDelegate {
	d := list.NewDefaultDelegate()

	// Customize styles to match Forge theme
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(types.SalmonPink).
		BorderForeground(types.SalmonPink)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(types.MutedGray).
		BorderForeground(types.SalmonPink)

	return noteListDelegate{DefaultDelegate: d}
}

// NotesOverlay displays scratchpad notes in a modal dialog
type NotesOverlay struct {
	list     list.Model
	notes    []pkgtypes.NoteData
	width    int
	height   int
	active   bool
	quitting bool
}

// NewNotesOverlay creates a new notes overlay
func NewNotesOverlay(notes []pkgtypes.NoteData, width, height int) *NotesOverlay {
	delegate := newNoteListDelegate()

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "📝 Scratchpad Notes"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)

	l.Styles.Title = lipgloss.NewStyle().
		Foreground(types.SalmonPink).
		Bold(true).
		Padding(0, 1)

	// Add custom key bindings
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "view note"),
			),
			key.NewBinding(
				key.WithKeys("esc", "q"),
				key.WithHelp("esc/q", "close"),
			),
		}
	}

	// Convert notes to list items
	items := make([]list.Item, len(notes))
	for i, note := range notes {
		noteCopy := note
		items[i] = noteListItem{
			note:        &noteCopy,
			tuiNoteData: convertNoteData(&noteCopy),
		}
	}

	l.SetItems(items)
	l.SetSize(width-4, height-4)

	return &NotesOverlay{
		list:   l,
		notes:  notes,
		width:  width,
		height: height,
		active: true,
	}
}

// Update handles messages for the notes overlay
func (o *NotesOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	if !o.active {
		return o, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", keyEsc:
			o.active = false
			// Return nil to signal close - caller will handle ClearOverlay()
			return nil, nil
		case keyEnter:
			// Show full note content
			if item, ok := o.list.SelectedItem().(noteListItem); ok {
				o.quitting = true
				return o, func() tea.Msg {
					return types.ViewNoteMsg{Note: item.tuiNoteData}
				}
			}
		}
	case tea.WindowSizeMsg:
		o.width = msg.Width
		o.height = msg.Height
		o.list.SetSize(msg.Width-4, msg.Height-4)
	}

	var cmd tea.Cmd
	o.list, cmd = o.list.Update(msg)
	return o, cmd
}

// View renders the notes overlay
func (o *NotesOverlay) View() string {
	if !o.active {
		return ""
	}

	// Create a bordered container for the list
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(types.SalmonPink).
		Padding(1, 2).
		Width(o.width - 4).
		Height(o.height - 4)

	return boxStyle.Render(o.list.View())
}

// Focused returns whether the notes overlay should handle input
func (o *NotesOverlay) Focused() bool {
	return o.active
}

// Width returns the width of the notes overlay
func (o *NotesOverlay) Width() int {
	return o.width
}

// Height returns the height of the notes overlay
func (o *NotesOverlay) Height() int {
	return o.height
}
