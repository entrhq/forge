package types

// OverlayMode represents the current overlay state
type OverlayMode int

const (
	// OverlayModeNone indicates no overlay is active
	OverlayModeNone OverlayMode = iota
	// OverlayModeDiffViewer shows the diff approval overlay
	OverlayModeDiffViewer
	// OverlayModeFileTree shows the file tree overlay
	OverlayModeFileTree
	// OverlayModeCommandOutput shows command output overlay
	OverlayModeCommandOutput
	// OverlayModeHelp shows the help overlay
	OverlayModeHelp
	// OverlayModeApproval shows generic approval overlay
	OverlayModeApproval
	// OverlayModeSettings shows the settings overlay
	OverlayModeSettings
	// OverlayModeContext shows the context information overlay
	OverlayModeContext
	// OverlayModeToolResult shows full tool result overlay
	OverlayModeToolResult
	// OverlayModeNotes shows the scratchpad notes overlay
	OverlayModeNotes
)
