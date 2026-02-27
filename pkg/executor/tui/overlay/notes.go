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
	list   list.Model
	notes  []pkgtypes.NoteData
	width  int
	height int
	active bool
}

// NewNotesOverlay creates a new notes overlay
func NewNotesOverlay(notes []pkgtypes.NoteData, width, height int) *NotesOverlay {
	overlayWidth := types.ComputeOverlayWidth(width, 0.80, 56, 100)
	overlayHeight := types.ComputeViewportHeight(height, 2)

	delegate := newNoteListDelegate()

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
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
	l.SetSize(overlayWidth-4, overlayHeight-4)

	return &NotesOverlay{
		list:   l,
		notes:  notes,
		width:  overlayWidth,
		height: overlayHeight,
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
				return o, func() tea.Msg {
					return types.ViewNoteMsg{Note: item.tuiNoteData}
				}
			}
		}
	case tea.WindowSizeMsg:
		o.width = types.ComputeOverlayWidth(msg.Width, 0.80, 56, 100)
		o.height = types.ComputeViewportHeight(msg.Height, 2)
		o.list.SetSize(o.width-4, o.height-2)
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

	title := fmt.Sprintf("Scratchpad Notes (%d)", len(o.notes))

	listContent := o.list.View()

	// Title pad logic matches help/context
	headerStr := types.OverlayTitleStyle.Render(title)
	innerWidth := o.width - 4
	if innerWidth < 0 {
		innerWidth = 0
	}

	headerLen := lipgloss.Width(headerStr)
	titlePad := ""
	for i := 0; i < max(0, (innerWidth-headerLen)/2); i++ {
		titlePad += " "
	}

	separator := lipgloss.NewStyle().Foreground(types.MutedGray).Render(strings.Repeat(sepChar, innerWidth))

	content := titlePad + headerStr + "\n" + separator + "\n" + listContent

	return types.CreateOverlayContainerStyle(o.width).Height(o.height).Render(content)
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
