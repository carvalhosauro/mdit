package editor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/carvalhosauro/mdit/internal/doc"
)

func TestLazyRaw_TableRenderedUntilEnter(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0}) // first table row
	tb := m.testBlockForLine(3)
	if m.layouts[tb].raw {
		t.Fatal("table under cursor should stay rendered until edit intent")
	}
	got := joinStrip(m.layouts[tb].lines)
	if !strings.Contains(got, "┼") {
		t.Fatalf("expected rendered table box, got %q", got)
	}

	m, _ = key(m, typeKey(tea.KeyEnter))
	if !m.hasBlockEdit() {
		t.Fatal("Enter should open table widget")
	}
	if m.layouts[tb].raw {
		t.Fatal("table widget must not use raw layout")
	}
	// Enter activates only — it must not insert a newline into the table.
	if m.Doc().Line(3) != "| A | B |" {
		t.Fatalf("Enter must not mutate table, line 3 = %q", m.Doc().Line(3))
	}
}

func TestLazyRaw_FirstRuneActivatesAndInserts(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	before := m.Doc().Content()
	m, _ = key(m, runeKey('X'))
	if !m.hasBlockEdit() {
		t.Fatal("first rune should open table widget")
	}
	if m.Doc().Content() != before {
		t.Fatalf("first rune must not mutate doc until commit, got %q", m.Doc().Content())
	}
}

func TestLazyRaw_EscExitsRawWhileStayingOnBlock(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	m, _ = key(m, typeKey(tea.KeyEnter))
	tb := m.testBlockForLine(3)
	if !m.hasBlockEdit() {
		t.Fatal("precondition: widget open")
	}
	m, _ = key(m, typeKey(tea.KeyEscape))
	if m.hasBlockEdit() {
		t.Fatal("Esc should clear table widget")
	}
	if m.layouts[tb].raw {
		t.Fatal("Esc should leave structural block rendered")
	}
	if m.Cursor().Line != 3 {
		t.Fatalf("Esc must keep cursor on line 3, got %d", m.Cursor().Line)
	}
}

func TestLazyRaw_LeavingBlockClearsEditing(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	m, _ = key(m, typeKey(tea.KeyEnter))
	// Move up to the blank line above the table (line 2).
	m, _ = key(m, typeKey(tea.KeyUp))
	tb := m.testBlockForLine(3)
	if m.hasBlockEdit() {
		t.Fatal("leaving the table should clear widget")
	}
	if m.layouts[tb].raw {
		t.Fatal("leaving the table should clear editing; table rendered again")
	}
}

func TestLazyRaw_ParagraphStillEager(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 1, Col: 0})
	pb := m.testBlockForLine(1)
	if !m.layouts[pb].raw {
		t.Fatal("paragraph (text) must stay eager-raw under cursor")
	}
}

func TestLazyRaw_CodeFenceLazy(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 8, Col: 0}) // "code line one"
	cb := m.testBlockForLine(8)
	if m.layouts[cb].raw {
		t.Fatal("code fence should be lazy-rendered under cursor")
	}
	m, _ = key(m, typeKey(tea.KeyEnter))
	if !m.layouts[cb].raw {
		t.Fatal("Enter should activate raw on code fence")
	}
}

func TestLazyRaw_CursorTracksRenderedTableRows(t *testing.T) {
	m := newFixture(t)
	m.SetSize(40, 12)
	m.cursorTo(doc.Position{Line: 3, Col: 2})
	rowTop, _ := m.cursorScreenRowCol()
	v := m.View()
	if !strings.Contains(ansi.Strip(v), "A") {
		t.Fatalf("view should still show rendered table text, got %q", ansi.Strip(v))
	}
	// Focus row must carry a reverse caret (lipgloss Reverse → ANSI).
	if !strings.Contains(v, "\x1b[7m") && !strings.Contains(v, "\x1b[7;") {
		t.Fatalf("expected reverse-video caret on rendered structural focus, view=%q", v)
	}

	m.cursorTo(doc.Position{Line: 5, Col: 2})
	rowBottom, _ := m.cursorScreenRowCol()
	if rowBottom <= rowTop {
		t.Fatalf("cursor row should advance with table source lines: top=%d bottom=%d", rowTop, rowBottom)
	}
}
