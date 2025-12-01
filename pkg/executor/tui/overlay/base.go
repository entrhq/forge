package overlay

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// BaseOverlay provides common functionality for all overlays.
// It handles viewport management, dimensions, focus state, and common key bindings.
type BaseOverlay struct {
	viewport viewport.Model
	width    int
	height   int
	focused  bool

	// Handlers for custom behavior
	onClose               func(actions types.ActionHandler) tea.Cmd
	onCustomKey           func(msg tea.KeyMsg, actions types.ActionHandler) (bool, tea.Cmd) // Returns (handled, cmd)
	renderHeader          func() string
	renderFooter          func() string
	footerRendersViewport bool // If true, footer is responsible for rendering viewport
}

// BaseOverlayConfig configures a base overlay
type BaseOverlayConfig struct {
	Width                 int
	Height                int
	ViewportWidth         int
	ViewportHeight        int
	Content               string
	OnClose               func(actions types.ActionHandler) tea.Cmd
	OnCustomKey           func(msg tea.KeyMsg, actions types.ActionHandler) (bool, tea.Cmd)
	RenderHeader          func() string
	RenderFooter          func() string
	FooterRendersViewport bool // If true, footer is responsible for rendering viewport
}

// NewBaseOverlay creates a new base overlay with the given configuration
func NewBaseOverlay(config BaseOverlayConfig) *BaseOverlay {
	vp := viewport.New(config.ViewportWidth, config.ViewportHeight)
	vp.Style = lipgloss.NewStyle()
	if config.Content != "" {
		vp.SetContent(config.Content)
	}

	return &BaseOverlay{
		viewport:              vp,
		width:                 config.Width,
		height:                config.Height,
		focused:               true,
		onClose:               config.OnClose,
		onCustomKey:           config.OnCustomKey,
		renderHeader:          config.RenderHeader,
		renderFooter:          config.RenderFooter,
		footerRendersViewport: config.FooterRendersViewport,
	}
}

// Update handles common overlay messages (window resize, viewport scrolling, close keys)
// Returns (handled, overlay, cmd) where handled indicates if the message was processed
func (b *BaseOverlay) Update(msg tea.Msg, actions types.ActionHandler) (bool, *BaseOverlay, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return b.handleKeyMsg(msg, actions)
	case tea.WindowSizeMsg:
		return b.handleWindowResize(msg)
	}
	return false, b, nil
}

// handleKeyMsg processes keyboard input
func (b *BaseOverlay) handleKeyMsg(msg tea.KeyMsg, actions types.ActionHandler) (bool, *BaseOverlay, tea.Cmd) {
	// Check for close keys (Esc, Ctrl+C)
	if b.isCloseKey(msg) {
		return true, b, b.close(actions)
	}

	// Give custom handler first priority
	if b.onCustomKey != nil {
		if handled, cmd := b.onCustomKey(msg, actions); handled {
			return true, b, cmd
		}
	}

	// Handle viewport scrolling
	if b.isScrollKey(msg) {
		var cmd tea.Cmd
		b.viewport, cmd = b.viewport.Update(msg)
		return true, b, cmd
	}

	return false, b, nil
}

// handleWindowResize updates dimensions on window resize
func (b *BaseOverlay) handleWindowResize(msg tea.WindowSizeMsg) (bool, *BaseOverlay, tea.Cmd) {
	b.width = msg.Width
	b.height = msg.Height

	var cmd tea.Cmd
	b.viewport, cmd = b.viewport.Update(msg)
	return true, b, cmd
}

// isCloseKey checks if the key should close the overlay
func (b *BaseOverlay) isCloseKey(msg tea.KeyMsg) bool {
	return msg.String() == keyEsc || msg.String() == keyCtrlC
}

// isScrollKey checks if the key is for scrolling
func (b *BaseOverlay) isScrollKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
		return true
	}
	return false
}

// close closes the overlay using the configured close handler or default
func (b *BaseOverlay) close(actions types.ActionHandler) tea.Cmd {
	if b.onClose != nil {
		return b.onClose(actions)
	}

	// Default close behavior - return nil to signal close
	// The caller (handleKeyPress) will call ClearOverlay()
	return nil
}

// View renders the overlay with header, viewport content, and footer
func (b *BaseOverlay) View(contentWidth int) string {
	var sections []string

	// Add header if provided
	if b.renderHeader != nil {
		sections = append(sections, b.renderHeader())
	}

	// Add viewport content unless footer is rendering it
	if !b.footerRendersViewport {
		sections = append(sections, b.viewport.View())
	}

	// Add footer if provided
	if b.renderFooter != nil {
		sections = append(sections, b.renderFooter())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return types.CreateOverlayContainerStyle(contentWidth).Render(content)
}

// SetContent updates the viewport content
func (b *BaseOverlay) SetContent(content string) {
	b.viewport.SetContent(content)
}

// Viewport returns the underlying viewport for advanced manipulation
func (b *BaseOverlay) Viewport() *viewport.Model {
	return &b.viewport
}

// Focused returns whether this overlay should handle input
func (b *BaseOverlay) Focused() bool {
	return b.focused
}

// SetFocused sets the focus state
func (b *BaseOverlay) SetFocused(focused bool) {
	b.focused = focused
}

// Width returns the overlay width
func (b *BaseOverlay) Width() int {
	return b.width
}

// Height returns the overlay height
func (b *BaseOverlay) Height() int {
	return b.height
}

// SetDimensions updates the overlay dimensions
func (b *BaseOverlay) SetDimensions(width, height int) {
	b.width = width
	b.height = height
}
