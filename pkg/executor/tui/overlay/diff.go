package overlay

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/executor/tui/syntax"
	"github.com/entrhq/forge/pkg/executor/tui/types"
	pkgtypes "github.com/entrhq/forge/pkg/types"
)

type ApprovalChoice int

const (
	ApprovalChoiceAccept ApprovalChoice = iota
	ApprovalChoiceReject
)

type DiffViewer struct {
	*ApprovalOverlayBase
	approvalID   string
	toolName     string
	preview      *tools.ToolPreview
	responseFunc func(*pkgtypes.ApprovalResponse)
}

func NewDiffViewer(approvalID, toolName string, preview *tools.ToolPreview, width, height int, responseFunc func(*pkgtypes.ApprovalResponse)) *DiffViewer {
	// Make overlay wide - 90% of screen width
	overlayWidth := max(int(float64(width)*0.9), 80)

	// Fixed viewport height: max 10 lines for diff content
	const maxViewportHeight = 10
	viewportHeight := maxViewportHeight

	// Calculate total overlay height
	// Title (2) + subtitle (1) + spacing (1) + border (2) + buttons (2) + hints (1) = 9 lines
	// Plus viewport height
	overlayHeight := viewportHeight + 9

	viewer := &DiffViewer{
		approvalID:   approvalID,
		toolName:     toolName,
		preview:      preview,
		responseFunc: responseFunc,
	}

	// Apply syntax highlighting to the diff content
	content := ""
	if preview != nil {
		// Extract language from metadata
		language := ""
		if preview.Metadata != nil {
			if lang, ok := preview.Metadata["language"].(string); ok {
				language = lang
			}
		}

		// Apply syntax highlighting
		highlightedContent, err := syntax.HighlightDiff(preview.Content, language)
		if err != nil {
			// Fall back to original content if highlighting fails
			content = preview.Content
		} else {
			content = highlightedContent
		}
	}

	// Configure approval overlay
	approvalConfig := ApprovalOverlayConfig{
		BaseConfig: BaseOverlayConfig{
			Width:                 overlayWidth,
			Height:                overlayHeight,
			ViewportWidth:         overlayWidth - 4,
			ViewportHeight:        viewportHeight,
			Content:               content,
			RenderHeader:          viewer.renderHeader,
			RenderFooter:          viewer.renderFooter,
			FooterRendersViewport: true, // Footer renders viewport with custom styling
		},
		OnApprove: viewer.handleApprove,
		OnReject:  viewer.handleReject,
		ShowHints: true,
	}

	viewer.ApprovalOverlayBase = NewApprovalOverlayBase(approvalConfig)
	return viewer
}

func (d *DiffViewer) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	// Check if this is a close key (ESC or Ctrl+C) before delegating
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		keyStr := keyMsg.String()
		if keyStr == "esc" || keyStr == "ctrl+c" {
			// Close keys should close the overlay
			return nil, nil
		}
	}

	updatedApproval, cmd := d.ApprovalOverlayBase.Update(msg, state, actions)
	d.ApprovalOverlayBase = updatedApproval
	return d, cmd
}

// handleApprove sends an approval response
func (d *DiffViewer) handleApprove() tea.Cmd {
	if d.responseFunc != nil {
		d.responseFunc(pkgtypes.NewApprovalResponse(d.approvalID, pkgtypes.ApprovalGranted))
	}
	return nil
}

// handleReject sends a rejection response
func (d *DiffViewer) handleReject() tea.Cmd {
	if d.responseFunc != nil {
		d.responseFunc(pkgtypes.NewApprovalResponse(d.approvalID, pkgtypes.ApprovalRejected))
	}
	return nil
}

// renderHeader renders the diff viewer header
func (d *DiffViewer) renderHeader() string {
	contentWidth := d.Width() - 6

	title := "Tool Approval Required"
	subtitle := fmt.Sprintf("%s: %s", d.toolName, d.preview.Title)

	// Manually center by calculating padding
	titleLen := len(title)
	subtitleLen := len(subtitle)
	titlePadding := max(0, (contentWidth-titleLen)/2)
	subtitlePadding := max(0, (contentWidth-subtitleLen)/2)

	var header strings.Builder
	header.WriteString(strings.Repeat(" ", titlePadding) + types.OverlayTitleStyle.Render(title))
	header.WriteString("\n")
	header.WriteString(strings.Repeat(" ", subtitlePadding) + types.OverlaySubtitleStyle.Render(subtitle))

	return header.String()
}

// renderFooter renders the diff viewer footer with buttons and hints
func (d *DiffViewer) renderFooter() string {
	contentWidth := d.Width() - 6

	var footer strings.Builder

	// Diff box has its own border (2) + padding (2), so reduce width further
	diffStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(types.SalmonPink).
		Padding(0, 1).
		Width(contentWidth - 4)

	footer.WriteString(diffStyle.Render(d.Viewport().View()))
	footer.WriteString("\n\n")

	// Render buttons
	buttonsRow := d.RenderButtons()
	buttonsLen := lipgloss.Width(buttonsRow)
	buttonsPadding := max(0, (contentWidth-buttonsLen)/2)
	footer.WriteString(strings.Repeat(" ", buttonsPadding) + buttonsRow)
	footer.WriteString("\n")

	// Render hints
	hints := d.RenderHints()
	hintsLen := lipgloss.Width(hints)
	hintsPadding := max(0, (contentWidth-hintsLen)/2)
	footer.WriteString(strings.Repeat(" ", hintsPadding) + hints)

	return footer.String()
}

func (d *DiffViewer) View() string {
	// BaseOverlay.View() already wraps in CreateOverlayContainerStyle, so just call it directly
	return d.BaseOverlay.View(d.Width())
}
