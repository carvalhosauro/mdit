// Package blockedit provides structured block editors (widgets) used by the
// mdit editor. Widgets are pure: they hold in-memory state, handle keys via
// Update, and never mutate the document — the editor applies CommitLines via
// doc.ReplaceLines on Commit.
package blockedit

import (
	"github.com/carvalhosauro/mdit/internal/doc"
	tea "github.com/charmbracelet/bubbletea"
)

// Signal tells the editor whether to keep the widget active or exit it.
type Signal int

const (
	// Continue keeps the widget active.
	Continue Signal = iota
	// Commit asks the editor to write CommitLines() via ReplaceLines, then clear active.
	Commit
	// Cancel discards the widget; the document is untouched.
	Cancel
)

// Widget is a structured editor for one block. Testable without a TUI loop.
type Widget interface {
	// Update handles input. On Commit/Cancel the editor clears active after applying.
	Update(msg tea.Msg) (Widget, tea.Cmd, Signal)

	// Lines returns screen rows for the block viewport.
	Lines(width int) []string

	// CommitLines returns the raw markdown lines that replace the block range.
	CommitLines() []string

	// ExitCursor returns a best-effort raw doc.Position after leaving the widget.
	ExitCursor(sig Signal) doc.Position
}
