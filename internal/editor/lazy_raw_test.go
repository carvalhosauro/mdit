package editor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
	if !m.layouts[tb].raw {
		t.Fatal("Enter should activate raw editing on the table")
	}
	if got := joinStrip(m.layouts[tb].lines); !strings.Contains(got, "| A | B |") {
		t.Fatalf("raw table missing source, got %q", got)
	}
	// Enter activates only — it must not insert a newline into the table.
	if m.Doc().Line(3) != "| A | B |" {
		t.Fatalf("Enter must not mutate table, line 3 = %q", m.Doc().Line(3))
	}
}

func TestLazyRaw_FirstRuneActivatesAndInserts(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	tb := m.testBlockForLine(3)
	m, _ = key(m, runeKey('X'))
	if !m.layouts[tb].raw {
		t.Fatal("first rune should activate raw mode")
	}
	if m.Doc().Line(3) != "X| A | B |" {
		t.Fatalf("first rune should insert, got %q", m.Doc().Line(3))
	}
}

func TestLazyRaw_EscExitsRawWhileStayingOnBlock(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	m, _ = key(m, typeKey(tea.KeyEnter))
	tb := m.testBlockForLine(3)
	if !m.layouts[tb].raw {
		t.Fatal("precondition: editing")
	}
	m, _ = key(m, typeKey(tea.KeyEscape))
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
