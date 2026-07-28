package editor

import (
	"github.com/carvalhosauro/mdit/internal/doc"
)

// Selection returns the normalized half-open selection range [from, to).
// ok is false when there is no active selection (or the range is empty).
func (m Model) Selection() (from, to doc.Position, ok bool) {
	if !m.selOn {
		return doc.Position{}, doc.Position{}, false
	}
	from, to = m.selAnchor, m.cursor
	if posLess(to, from) {
		from, to = to, from
	}
	if from == to {
		return from, to, false
	}
	return from, to, true
}

// SelectedText returns the document text covered by the selection, or "".
func (m Model) SelectedText() string {
	from, to, ok := m.Selection()
	if !ok {
		return ""
	}
	return m.doc.TextRange(from, to)
}

// Register returns the internal yank/paste buffer.
func (m Model) Register() string { return m.register }

// HasSelection reports whether a non-empty selection is active.
func (m Model) HasSelection() bool {
	_, _, ok := m.Selection()
	return ok
}

func (m *Model) clearSelection() {
	m.selOn = false
	m.selAnchor = doc.Position{}
}

// beginExtend starts a selection at the current cursor if one is not already
// active. Call before moving the cursor with a shift-motion.
func (m *Model) beginExtend() {
	if !m.selOn {
		m.selOn = true
		m.selAnchor = m.cursor
	}
}

// deleteSelection removes the selected range, clears the selection, and
// returns true when a deletion happened.
func (m *Model) deleteSelection() bool {
	from, to, ok := m.Selection()
	if !ok {
		return false
	}
	m.cursor = m.doc.DeleteRange(from, to)
	m.goalCol = m.cursor.Col
	m.clearSelection()
	return true
}

// yankSelection copies the selection into the internal register. Returns the
// yanked text (empty when there is no selection).
func (m *Model) yankSelection() string {
	text := m.SelectedText()
	if text == "" {
		return ""
	}
	m.register = text
	return text
}

func posLess(a, b doc.Position) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Col < b.Col
}

// posInRange reports whether p is in the half-open range [from, to).
func posInRange(p, from, to doc.Position) bool {
	if posLess(p, from) {
		return false
	}
	return posLess(p, to)
}
