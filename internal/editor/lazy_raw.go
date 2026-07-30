package editor

import (
	"github.com/carvalhosauro/mdit/internal/blockedit"
	"github.com/carvalhosauro/mdit/internal/mdparse"
)

// openTableFn is the table factory; tests may stub it to force fallback.
var openTableFn = blockedit.OpenTable

// isStructural reports kinds that stay rendered under the cursor until the
// user signals edit intent (Enter or the first typed character). Textual kinds
// (paragraph, heading, list, quote, …) remain eager-raw.
func isStructural(k mdparse.Kind) bool {
	switch k {
	case mdparse.Table, mdparse.CodeFence, mdparse.IndentedCode:
		return true
	default:
		return false
	}
}

// cursorBlockRaw reports whether the cursor's block is currently shown raw.
func (m Model) cursorBlockRaw() bool {
	if m.zen || len(m.layouts) == 0 || m.cursorBlock < 0 || m.cursorBlock >= len(m.layouts) {
		return false
	}
	return m.layouts[m.cursorBlock].raw
}

// shouldLazyActivate is true when the cursor sits on a structural block that
// is still in the rendered (non-editing) state.
func (m Model) shouldLazyActivate() bool {
	if m.zen || m.editing || m.active != nil || len(m.blocks) == 0 {
		return false
	}
	if m.cursorBlock < 0 || m.cursorBlock >= len(m.blocks) {
		return false
	}
	return isStructural(m.blocks[m.cursorBlock].Kind)
}

// shouldOpenTableWidget is true when edit intent on the cursor block should
// open the structured table widget instead of lazy-raw.
func (m Model) shouldOpenTableWidget() bool {
	return m.shouldLazyActivate() && m.blocks[m.cursorBlock].Kind == mdparse.Table
}

// activateEditing flips the lazy-raw gate on so the structural cursor block
// re-layouts as raw source.
func (m *Model) activateEditing() {
	m.active = nil
	m.editing = true
}

// openTableWidget tries to open a table widget for the cursor block.
// Returns false when the table is malformed (caller falls back to lazy-raw).
func (m *Model) openTableWidget() bool {
	if m.cursorBlock < 0 || m.cursorBlock >= len(m.blocks) {
		return false
	}
	b := m.blocks[m.cursorBlock]
	raw := m.rawRange(b.Start, b.End)
	w, ok := openTableFn(raw, m.cursor, b.Start)
	if !ok {
		return false
	}
	m.clearSelection()
	m.editing = false
	m.active = w
	m.widgetBlockStart = b.Start
	m.widgetBlockEnd = b.End
	m.cursorBeforeOpen = m.cursor
	return true
}

// finishBlockEdit applies Commit/Cancel and clears the active widget.
// Cancel discards; Commit writes via ReplaceLines.
func (m *Model) finishBlockEdit(sig blockedit.Signal) {
	if m.active == nil {
		return
	}
	switch sig {
	case blockedit.Commit:
		start, end := m.widgetBlockStart, m.widgetBlockEnd
		lines := m.active.CommitLines()
		exit := m.active.ExitCursor(blockedit.Commit)
		m.active = nil
		m.editing = false
		pos := m.doc.ReplaceLines(start, end, lines)
		if exit.Line >= start {
			m.cursor = exit
		} else {
			m.cursor = pos
		}
		m.goalCol = m.cursor.Col
	case blockedit.Cancel:
		exit := m.active.ExitCursor(blockedit.Cancel)
		m.active = nil
		m.editing = false
		m.cursor = exit
		m.goalCol = m.cursor.Col
	default:
		return
	}
	m.recompute()
}
