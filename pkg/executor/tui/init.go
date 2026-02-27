package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
)

// initialModel returns the initial state of the TUI.
// It creates and configures all components needed for the interactive interface.
func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = "" // ADR-0051: prompt glyph rendered externally in buildInputBox
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.MaxHeight = 10 // Allow up to 10 lines
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Disable default Enter behavior
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(salmonPink)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(brightWhite)

	// Viewport initialization: wait to receive tea.WindowSizeMsg to set actual dimensions
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().Padding(0, 2)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(salmonPink)

	// Initialize command palette with registered commands
	commands := getAllCommands()
	cmdItems := make([]overlay.CommandItem, len(commands))
	for i, cmd := range commands {
		cmdItems[i] = overlay.CommandItem{
			Name:        cmd.Name,
			Description: cmd.Description,
		}
	}

	return model{
		viewport:         vp,
		textarea:         ta,
		content:          &strings.Builder{},
		thinkingBuffer:   &strings.Builder{},
		messageBuffer:    &strings.Builder{},
		overlay:          newOverlayState(),
		commandPalette:   overlay.NewCommandPalette(cmdItems),
		summarization:    &summarizationStatus{},
		toast:            &toastNotification{},
		spinner:          s,
		agentBusy:        false,
		followScroll:     true,  // ADR-0048: auto-follow agent output by default
		hasNewContent:    false, // ADR-0048: no new content initially
		resultClassifier: NewToolResultClassifier(),
		resultSummarizer: NewToolResultSummarizer(),
		resultCache:      newResultCache(20),
		resultList:       overlay.NewResultListModel(),
	}
}

// Init is the first function that will be called by Bubble Tea.
// It returns commands to start the textarea blink animation and spinner,
// plus any startup warning toasts queued before the session began.
func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{textarea.Blink, m.spinner.Tick}
	for _, w := range m.startupWarnings {
		w := w // capture loop variable
		cmds = append(cmds, func() tea.Msg { return w })
	}
	return tea.Batch(cmds...)
}
