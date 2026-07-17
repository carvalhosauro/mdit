package editor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/theme"
)

func newEditor(t *testing.T, text string, w, h int) Model {
	t.Helper()
	d := doc.NewFromString(text)
	m := New(d, theme.DefaultDark(), nil)
	m.SetSize(w, h)
	return m
}

func key(m Model, k tea.KeyMsg) (Model, tea.Cmd) {
	return m.Update(k)
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func typeKey(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func TestKeys_InsertRune(t *testing.T) {
	m := newEditor(t, "ab", 20, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 1})
	m, _ = key(m, runeKey('Z'))
	if m.Doc().Line(0) != "aZb" {
		t.Fatalf("expected aZb, got %q", m.Doc().Line(0))
	}
	if m.Cursor().Col != 2 {
		t.Fatalf("cursor col should be 2, got %d", m.Cursor().Col)
	}
}

func TestKeys_EnterSplitsLine(t *testing.T) {
	m := newEditor(t, "abcd", 20, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 2})
	m, _ = key(m, typeKey(tea.KeyEnter))
	if m.Doc().LineCount() != 2 || m.Doc().Line(0) != "ab" || m.Doc().Line(1) != "cd" {
		t.Fatalf("enter split wrong: %q / %q", m.Doc().Line(0), m.Doc().Line(1))
	}
	if m.Cursor().Line != 1 || m.Cursor().Col != 0 {
		t.Fatalf("cursor after enter should be 1,0 got %d,%d", m.Cursor().Line, m.Cursor().Col)
	}
}

func TestKeys_BackspaceJoinsLines(t *testing.T) {
	m := newEditor(t, "ab\ncd", 20, 6)
	m.cursorTo(doc.Position{Line: 1, Col: 0})
	m, _ = key(m, typeKey(tea.KeyBackspace))
	if m.Doc().LineCount() != 1 || m.Doc().Line(0) != "abcd" {
		t.Fatalf("backspace should join to abcd, got %q (%d lines)", m.Doc().Line(0), m.Doc().LineCount())
	}
	if m.Cursor().Line != 0 || m.Cursor().Col != 2 {
		t.Fatalf("cursor should be at join point 0,2 got %d,%d", m.Cursor().Line, m.Cursor().Col)
	}
}

func TestKeys_DeleteForward(t *testing.T) {
	m := newEditor(t, "abc", 20, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 1})
	m, _ = key(m, typeKey(tea.KeyDelete))
	if m.Doc().Line(0) != "ac" {
		t.Fatalf("delete-forward should give ac, got %q", m.Doc().Line(0))
	}
}

func TestKeys_ArrowsDownUpRawLinesWithGoalColumn(t *testing.T) {
	// line 0 short, line 1 long, line 2 short: goal column preserved across the
	// short middle line is not applicable here; use varying lengths.
	m := newEditor(t, "hello\nhi\nworld!", 20, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 5}) // end of "hello"
	m, _ = key(m, typeKey(tea.KeyDown))
	// line 1 "hi" len 2 -> clamp col to 2, but goal column remembers 5.
	if m.Cursor().Line != 1 || m.Cursor().Col != 2 {
		t.Fatalf("down to short line should clamp col to 2, got %d,%d", m.Cursor().Line, m.Cursor().Col)
	}
	m, _ = key(m, typeKey(tea.KeyDown))
	// line 2 "world!" len 6 -> goal column 5 restored.
	if m.Cursor().Line != 2 || m.Cursor().Col != 5 {
		t.Fatalf("down should restore goal col 5, got %d,%d", m.Cursor().Line, m.Cursor().Col)
	}
	m, _ = key(m, typeKey(tea.KeyUp))
	m, _ = key(m, typeKey(tea.KeyUp))
	if m.Cursor().Line != 0 || m.Cursor().Col != 5 {
		t.Fatalf("up back to line 0 should keep goal col 5, got %d,%d", m.Cursor().Line, m.Cursor().Col)
	}
}

func TestKeys_LeftWrapsToPrevLineEnd(t *testing.T) {
	m := newEditor(t, "ab\ncd", 20, 6)
	m.cursorTo(doc.Position{Line: 1, Col: 0})
	m, _ = key(m, typeKey(tea.KeyLeft))
	if m.Cursor().Line != 0 || m.Cursor().Col != 2 {
		t.Fatalf("left at col0 should wrap to prev line end 0,2 got %d,%d", m.Cursor().Line, m.Cursor().Col)
	}
}

func TestKeys_RightWrapsToNextLineStart(t *testing.T) {
	m := newEditor(t, "ab\ncd", 20, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 2})
	m, _ = key(m, typeKey(tea.KeyRight))
	if m.Cursor().Line != 1 || m.Cursor().Col != 0 {
		t.Fatalf("right at line end should wrap to next line start 1,0 got %d,%d", m.Cursor().Line, m.Cursor().Col)
	}
}

func TestKeys_HomeEnd(t *testing.T) {
	m := newEditor(t, "hello world", 20, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 3})
	m, _ = key(m, typeKey(tea.KeyEnd))
	if m.Cursor().Col != 11 {
		t.Fatalf("end should move to col 11, got %d", m.Cursor().Col)
	}
	m, _ = key(m, typeKey(tea.KeyHome))
	if m.Cursor().Col != 0 {
		t.Fatalf("home should move to col 0, got %d", m.Cursor().Col)
	}
}

func TestKeys_UndoRedoRestoresCursor(t *testing.T) {
	m := newEditor(t, "abc", 20, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 3})
	m, _ = key(m, runeKey('!'))
	if m.Doc().Line(0) != "abc!" {
		t.Fatalf("expected abc!, got %q", m.Doc().Line(0))
	}
	m, _ = key(m, tea.KeyMsg{Type: tea.KeyCtrlZ})
	if m.Doc().Line(0) != "abc" {
		t.Fatalf("undo should restore abc, got %q", m.Doc().Line(0))
	}
	m, _ = key(m, tea.KeyMsg{Type: tea.KeyCtrlY})
	if m.Doc().Line(0) != "abc!" {
		t.Fatalf("redo should restore abc!, got %q", m.Doc().Line(0))
	}
}

func TestKeys_TableGoesRawOnEnterAndRendersOnExit(t *testing.T) {
	text := strings.Join([]string{
		"# T",
		"| A | B |",
		"| - | - |",
		"| 1 | 2 |",
		"",
		"end",
	}, "\n")
	m := newEditor(t, text, 20, 8)
	// Cursor at heading: table rendered.
	tb := m.testBlockForLine(2)
	if m.layouts[tb].raw {
		t.Fatalf("table should render when cursor outside")
	}
	// Enter the table (line 3).
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	tb = m.testBlockForLine(3)
	if !m.layouts[tb].raw {
		t.Fatalf("table should be raw when cursor enters it")
	}
	got := joinStrip(m.layouts[tb].lines)
	if !strings.Contains(got, "| 1 | 2 |") {
		t.Fatalf("raw table should show raw pipes, got %q", got)
	}
	// Leave the table (down to the "end" paragraph past the blank separator).
	m.cursorTo(doc.Position{Line: 5, Col: 0})
	tb = m.testBlockForLine(2)
	if m.layouts[tb].raw {
		t.Fatalf("table should re-render on exit")
	}
}

func TestKeys_FollowLinkEmitsCmd(t *testing.T) {
	m := newEditor(t, "see [[Target]] here", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 6}) // inside [[Target]]
	_, cmd := key(m, tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	if cmd == nil {
		t.Fatalf("ctrl+] over wikilink should emit a cmd")
	}
	msg := cmd()
	fl, ok := msg.(FollowLinkMsg)
	if !ok {
		t.Fatalf("expected FollowLinkMsg, got %T", msg)
	}
	if fl.Target != "Target" {
		t.Fatalf("expected target 'Target', got %q", fl.Target)
	}
}

func TestKeys_FollowLinkNoLinkNoCmd(t *testing.T) {
	m := newEditor(t, "no link here", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 2})
	_, cmd := key(m, tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	if cmd != nil {
		t.Fatalf("ctrl+] with no wikilink should not emit a cmd")
	}
}

func TestKeys_AutocompleteOnDoubleBracket(t *testing.T) {
	m := newEditor(t, "", 40, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 0})
	m, cmd := key(m, runeKey('['))
	if cmd != nil {
		t.Fatalf("single [ should not emit autocomplete")
	}
	m, cmd = key(m, runeKey('['))
	if cmd == nil {
		t.Fatalf("second [ should emit autocomplete cmd")
	}
	if _, ok := cmd().(AutocompleteMsg); !ok {
		t.Fatalf("expected AutocompleteMsg, got %T", cmd())
	}
}

func TestKeys_SetDocResetsState(t *testing.T) {
	m := newEditor(t, "one\ntwo\nthree", 20, 6)
	m.cursorTo(doc.Position{Line: 2, Col: 0})
	d2 := doc.NewFromString("fresh")
	m.SetDoc(d2)
	if m.Cursor().Line != 0 || m.Cursor().Col != 0 {
		t.Fatalf("SetDoc should reset cursor to 0,0 got %d,%d", m.Cursor().Line, m.Cursor().Col)
	}
	if m.Doc() != d2 {
		t.Fatalf("SetDoc should swap the document")
	}
	if m.scroll != 0 {
		t.Fatalf("SetDoc should reset scroll, got %d", m.scroll)
	}
}

func TestKeys_CJKWidthCursorMapping(t *testing.T) {
	// Wide runes are width 2; cursor cell column must account for that.
	m := newEditor(t, "你好x", 20, 6)
	m.cursorTo(doc.Position{Line: 0, Col: 2}) // after 你好
	_, col := m.cursorScreenRowCol()
	if col != 4 {
		t.Fatalf("cursor after two wide runes should be at cell 4, got %d", col)
	}
}
