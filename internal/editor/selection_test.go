package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/carvalhosauro/mdit/internal/doc"
)

func TestSelection_ShiftRightMarksRange(t *testing.T) {
	m := newEditor(t, "hello world", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 0})

	m, _ = key(m, typeKey(tea.KeyShiftRight))
	m, _ = key(m, typeKey(tea.KeyShiftRight))
	m, _ = key(m, typeKey(tea.KeyShiftRight))
	m, _ = key(m, typeKey(tea.KeyShiftRight))
	m, _ = key(m, typeKey(tea.KeyShiftRight))

	from, to, ok := m.Selection()
	if !ok {
		t.Fatal("expected an active selection")
	}
	if from != (doc.Position{Line: 0, Col: 0}) || to != (doc.Position{Line: 0, Col: 5}) {
		t.Fatalf("selection = %v-%v, want 0:0-0:5", from, to)
	}
	if got := m.SelectedText(); got != "hello" {
		t.Fatalf("SelectedText() = %q, want %q", got, "hello")
	}
}

func TestSelection_PlainArrowClears(t *testing.T) {
	m := newEditor(t, "abcd", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 1})
	m, _ = key(m, typeKey(tea.KeyShiftRight))
	m, _ = key(m, typeKey(tea.KeyShiftRight))
	if _, _, ok := m.Selection(); !ok {
		t.Fatal("expected selection before clear")
	}
	m, _ = key(m, typeKey(tea.KeyRight))
	if _, _, ok := m.Selection(); ok {
		t.Fatal("plain arrow should clear selection")
	}
}

func TestSelection_DeleteRemovesRange(t *testing.T) {
	m := newEditor(t, "hello world", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 0})
	for i := 0; i < 5; i++ {
		m, _ = key(m, typeKey(tea.KeyShiftRight))
	}
	m, _ = key(m, typeKey(tea.KeyDelete))
	if m.Doc().Line(0) != " world" {
		t.Fatalf("after delete selection got %q, want %q", m.Doc().Line(0), " world")
	}
	if _, _, ok := m.Selection(); ok {
		t.Fatal("selection should clear after delete")
	}
	if m.Cursor() != (doc.Position{Line: 0, Col: 0}) {
		t.Fatalf("cursor = %v, want 0:0", m.Cursor())
	}
}

func TestSelection_BackspaceRemovesRange(t *testing.T) {
	m := newEditor(t, "abXYcd", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 2})
	m, _ = key(m, typeKey(tea.KeyShiftRight))
	m, _ = key(m, typeKey(tea.KeyShiftRight))
	m, _ = key(m, typeKey(tea.KeyBackspace))
	if m.Doc().Line(0) != "abcd" {
		t.Fatalf("got %q, want abcd", m.Doc().Line(0))
	}
}

func TestSelection_TypingReplacesRange(t *testing.T) {
	m := newEditor(t, "hello world", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 0})
	for i := 0; i < 5; i++ {
		m, _ = key(m, typeKey(tea.KeyShiftRight))
	}
	m, _ = key(m, runeKey('X'))
	if m.Doc().Line(0) != "X world" {
		t.Fatalf("got %q, want %q", m.Doc().Line(0), "X world")
	}
}

func TestSelection_CopyPasteRoundTrip(t *testing.T) {
	m := newEditor(t, "hello world", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 0})
	for i := 0; i < 5; i++ {
		m, _ = key(m, typeKey(tea.KeyShiftRight))
	}
	m, _ = key(m, typeKey(tea.KeyCtrlW)) // copy
	if m.Register() != "hello" {
		t.Fatalf("register = %q, want hello", m.Register())
	}
	// Move to end and paste.
	m, _ = key(m, typeKey(tea.KeyEnd))
	m, _ = key(m, typeKey(tea.KeyCtrlV))
	if m.Doc().Line(0) != "hello worldhello" {
		t.Fatalf("got %q, want hello worldhello", m.Doc().Line(0))
	}
}

func TestSelection_CutPaste(t *testing.T) {
	m := newEditor(t, "hello world", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 6})
	for i := 0; i < 5; i++ {
		m, _ = key(m, typeKey(tea.KeyShiftRight))
	}
	m, _ = key(m, typeKey(tea.KeyCtrlX)) // cut
	if m.Doc().Line(0) != "hello " {
		t.Fatalf("after cut got %q, want %q", m.Doc().Line(0), "hello ")
	}
	if m.Register() != "world" {
		t.Fatalf("register = %q, want world", m.Register())
	}
	m.cursorTo(doc.Position{Line: 0, Col: 0})
	m, _ = key(m, typeKey(tea.KeyCtrlV))
	if m.Doc().Line(0) != "worldhello " {
		t.Fatalf("got %q, want worldhello ", m.Doc().Line(0))
	}
}

func TestSelection_ShiftWordAndHomeEnd(t *testing.T) {
	m := newEditor(t, "one two three", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 0})
	m, _ = key(m, typeKey(tea.KeyCtrlShiftRight))
	// moveWordRight stops after the word (before following spaces).
	if got := m.SelectedText(); got != "one" {
		t.Fatalf("shift-word-right selected %q, want %q", got, "one")
	}
	m, _ = key(m, typeKey(tea.KeyShiftEnd))
	if got := m.SelectedText(); got != "one two three" {
		t.Fatalf("shift-end selected %q, want full line from anchor", got)
	}
}

func TestSelection_Multiline(t *testing.T) {
	m := newEditor(t, "aaa\nbbb\nccc", 40, 8)
	m.cursorTo(doc.Position{Line: 0, Col: 1})
	m, _ = key(m, typeKey(tea.KeyShiftDown))
	m, _ = key(m, typeKey(tea.KeyShiftDown))
	from, to, ok := m.Selection()
	if !ok {
		t.Fatal("expected selection")
	}
	if from.Line != 0 || to.Line != 2 {
		t.Fatalf("selection lines %d-%d, want 0-2", from.Line, to.Line)
	}
	if got := m.SelectedText(); got != "aa\nbbb\ncc" {
		// from col 1 on "aaa" through col 1 on "ccc" with goal-col behavior:
		// shift-down keeps goal column.
		_ = got
	}
	// Exact text with goal column 1: "aa\nbbb\nc" wait - after two ShiftDown cursor at line 2 col 1
	// range [0,1) to [2,1) = "aa\nbbb\nc"
	if got := m.SelectedText(); got != "aa\nbbb\nc" {
		t.Fatalf("SelectedText() = %q, want %q", got, "aa\nbbb\nc")
	}
}

func TestSelection_SelectAll(t *testing.T) {
	m := newEditor(t, "ab\ncd", 40, 6)
	m, _ = key(m, typeKey(tea.KeyCtrlA))
	if got := m.SelectedText(); got != "ab\ncd" {
		t.Fatalf("select-all got %q", got)
	}
}

func TestPaste_BracketedPasteInsertsLiterally(t *testing.T) {
	m := newEditor(t, "xx", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 1})
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a[[b"), Paste: true}
	m, cmd := key(m, msg)
	if m.Doc().Line(0) != "xa[[bx" {
		t.Fatalf("got %q, want xa[[bx", m.Doc().Line(0))
	}
	// Bracketed paste must not open wikilink autocomplete.
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, ok := msg.(AutocompleteMsg); ok {
				t.Fatal("paste must not emit AutocompleteMsg")
			}
		}
	}
}
