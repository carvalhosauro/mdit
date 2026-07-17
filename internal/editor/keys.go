package editor

import (
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/mdparse"
)

// handleKey dispatches a key message, mutating the model and returning it along
// with any command (follow-link or autocomplete) the key produced.
func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyRunes:
		return m.insertAndMaybeAutocomplete(string(msg.Runes))
	case tea.KeySpace:
		return m.insertAndMaybeAutocomplete(" ")
	case tea.KeyEnter:
		m.cursor = m.doc.Insert(m.cursor, "\n")
		m.goalCol = m.cursor.Col
		m.recompute()
		return m, nil
	case tea.KeyBackspace:
		m.cursor = m.doc.DeleteBackward(m.cursor)
		m.goalCol = m.cursor.Col
		m.recompute()
		return m, nil
	case tea.KeyDelete:
		m.cursor = m.doc.DeleteForward(m.cursor)
		m.goalCol = m.cursor.Col
		m.recompute()
		return m, nil

	case tea.KeyUp:
		m.moveVertical(-1)
	case tea.KeyDown:
		m.moveVertical(1)
	case tea.KeyLeft:
		m.moveLeft()
	case tea.KeyRight:
		m.moveRight()
	case tea.KeyCtrlLeft:
		m.moveWordLeft()
	case tea.KeyCtrlRight:
		m.moveWordRight()
	case tea.KeyHome:
		m.cursor.Col = 0
		m.goalCol = 0
	case tea.KeyEnd:
		m.cursor.Col = runeLen(m.doc.Line(m.cursor.Line))
		m.goalCol = m.cursor.Col
	case tea.KeyPgUp:
		m.movePage(-1)
	case tea.KeyPgDown:
		m.movePage(1)

	case tea.KeyCtrlZ:
		if pos, ok := m.doc.Undo(); ok {
			m.cursor = pos
			m.goalCol = pos.Col
		}
	case tea.KeyCtrlY:
		if pos, ok := m.doc.Redo(); ok {
			m.cursor = pos
			m.goalCol = pos.Col
		}

	case tea.KeyCtrlCloseBracket:
		if target, ok := mdparse.WikiLinkAt(m.doc.Line(m.cursor.Line), m.cursor.Col); ok {
			t := target
			m.recompute()
			return m, func() tea.Msg { return FollowLinkMsg{Target: t} }
		}
		return m, nil

	default:
		return m, nil // ignored; higher layers (Task 7) handle the rest
	}

	m.recompute()
	return m, nil
}

// insertAndMaybeAutocomplete inserts s at the cursor and, if that just completed
// a "[[" trigger, returns an AutocompleteMsg command.
func (m Model) insertAndMaybeAutocomplete(s string) (Model, tea.Cmd) {
	m.cursor = m.doc.Insert(m.cursor, s)
	m.goalCol = m.cursor.Col
	m.recompute()
	if cmd := m.autocompleteCmd(); cmd != nil {
		return m, cmd
	}
	return m, nil
}

// autocompleteCmd returns an AutocompleteMsg command when the two runes ending
// at the cursor are "[[".
func (m Model) autocompleteCmd() tea.Cmd {
	line := []rune(m.doc.Line(m.cursor.Line))
	c := m.cursor.Col
	if c >= 2 && line[c-1] == '[' && line[c-2] == '[' {
		return func() tea.Msg { return AutocompleteMsg{Query: ""} }
	}
	return nil
}

// moveVertical moves the cursor by delta raw lines, clamping the column to the
// target line length while remembering the goal column for later moves.
func (m *Model) moveVertical(delta int) {
	target := m.cursor.Line + delta
	if target < 0 || target > m.doc.LineCount()-1 {
		return
	}
	m.cursor.Line = target
	n := runeLen(m.doc.Line(target))
	if m.goalCol < n {
		m.cursor.Col = m.goalCol
	} else {
		m.cursor.Col = n
	}
}

// movePage moves the cursor by dir*height raw lines (page up/down); scroll
// follows in recompute.
func (m *Model) movePage(dir int) {
	step := m.height
	if step < 1 {
		step = 1
	}
	target := m.cursor.Line + dir*step
	if target < 0 {
		target = 0
	}
	if target > m.doc.LineCount()-1 {
		target = m.doc.LineCount() - 1
	}
	m.cursor.Line = target
	n := runeLen(m.doc.Line(target))
	if m.goalCol < n {
		m.cursor.Col = m.goalCol
	} else {
		m.cursor.Col = n
	}
}

func (m *Model) moveLeft() {
	if m.cursor.Col > 0 {
		m.cursor.Col--
	} else if m.cursor.Line > 0 {
		m.cursor.Line--
		m.cursor.Col = runeLen(m.doc.Line(m.cursor.Line))
	}
	m.goalCol = m.cursor.Col
}

func (m *Model) moveRight() {
	n := runeLen(m.doc.Line(m.cursor.Line))
	if m.cursor.Col < n {
		m.cursor.Col++
	} else if m.cursor.Line < m.doc.LineCount()-1 {
		m.cursor.Line++
		m.cursor.Col = 0
	}
	m.goalCol = m.cursor.Col
}

func (m *Model) moveWordLeft() {
	if m.cursor.Col == 0 {
		if m.cursor.Line > 0 {
			m.cursor.Line--
			m.cursor.Col = runeLen(m.doc.Line(m.cursor.Line))
		}
		m.goalCol = m.cursor.Col
		return
	}
	runes := []rune(m.doc.Line(m.cursor.Line))
	c := m.cursor.Col
	for c > 0 && unicode.IsSpace(runes[c-1]) {
		c--
	}
	for c > 0 && !unicode.IsSpace(runes[c-1]) {
		c--
	}
	m.cursor.Col = c
	m.goalCol = c
}

func (m *Model) moveWordRight() {
	runes := []rune(m.doc.Line(m.cursor.Line))
	c := m.cursor.Col
	if c >= len(runes) {
		if m.cursor.Line < m.doc.LineCount()-1 {
			m.cursor.Line++
			m.cursor.Col = 0
		}
		m.goalCol = m.cursor.Col
		return
	}
	for c < len(runes) && unicode.IsSpace(runes[c]) {
		c++
	}
	for c < len(runes) && !unicode.IsSpace(runes[c]) {
		c++
	}
	m.cursor.Col = c
	m.goalCol = c
}

// --- white-box test helpers (used by tests to drive state without KeyMsgs) ---

// cursorTo moves the cursor to p, resetting the goal column, and rebuilds layout.
func (m *Model) cursorTo(p doc.Position) {
	m.cursor = p
	m.goalCol = p.Col
	m.recompute()
}

// insertText inserts s at the cursor and rebuilds layout.
func (m *Model) insertText(s string) {
	m.cursor = m.doc.Insert(m.cursor, s)
	m.goalCol = m.cursor.Col
	m.recompute()
}

// undo applies one undo step, restoring the cursor from the patch.
func (m *Model) undo() {
	if pos, ok := m.doc.Undo(); ok {
		m.cursor = pos
		m.goalCol = pos.Col
	}
	m.recompute()
}
