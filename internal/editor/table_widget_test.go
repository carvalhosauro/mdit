package editor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/carvalhosauro/mdit/internal/blockedit"
	"github.com/carvalhosauro/mdit/internal/doc"
)

func TestTableWidget_EnterOpensWidgetNotRaw(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	tb := m.testBlockForLine(3)
	v0 := m.Doc().Version()

	m, _ = key(m, typeKey(tea.KeyEnter))
	if !m.hasBlockEdit() {
		t.Fatal("Enter on table should open blockedit widget")
	}
	if m.layouts[tb].raw {
		t.Fatal("table widget must not use raw layout")
	}
	if m.Doc().Version() != v0 {
		t.Fatalf("opening widget must not mutate doc, version %d→%d", v0, m.Doc().Version())
	}
	if m.Doc().Line(3) != "| A | B |" {
		t.Fatalf("Enter must not insert newline, line=%q", m.Doc().Line(3))
	}
	got := joinStrip(m.layouts[tb].lines)
	if strings.Contains(got, "| A | B |") {
		t.Fatalf("widget view should not be pipe source, got %q", got)
	}
	if !strings.Contains(got, "A") {
		t.Fatalf("widget view should show cell text, got %q", got)
	}
}

func TestTableWidget_FirstRuneOpensAndEditsMemory(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	before := m.Doc().Content()
	v0 := m.Doc().Version()

	m, _ = key(m, runeKey('X'))
	if !m.hasBlockEdit() {
		t.Fatal("first rune should open table widget")
	}
	if m.Doc().Version() != v0 || m.Doc().Content() != before {
		t.Fatal("first rune must not write to doc until commit")
	}
}

func TestTableWidget_EscCancels(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	m, _ = key(m, typeKey(tea.KeyEnter))
	if !m.hasBlockEdit() {
		t.Fatal("precondition")
	}
	m, _ = key(m, typeKey(tea.KeyEscape))
	tb := m.testBlockForLine(3)
	if m.hasBlockEdit() {
		t.Fatal("Esc should clear widget")
	}
	if m.layouts[tb].raw {
		t.Fatal("table should be rendered after Esc")
	}
	if m.Cursor().Line != 3 {
		t.Fatalf("cursor should stay on line 3, got %d", m.Cursor().Line)
	}
}

func TestTableWidget_FenceStillLazyRaw(t *testing.T) {
	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 8, Col: 0})
	cb := m.testBlockForLine(8)
	if m.layouts[cb].raw {
		t.Fatal("code fence should be lazy-rendered")
	}
	m, _ = key(m, typeKey(tea.KeyEnter))
	if m.hasBlockEdit() {
		t.Fatal("fence must not open table widget")
	}
	if !m.layouts[cb].raw {
		t.Fatal("Enter should activate raw on code fence")
	}
}

func TestTableWidget_MalformedFallsBackToRaw(t *testing.T) {
	prev := openTableFn
	openTableFn = func([]string, doc.Position, int) (blockedit.Widget, bool) {
		return nil, false
	}
	t.Cleanup(func() { openTableFn = prev })

	m := newFixture(t)
	m.cursorTo(doc.Position{Line: 3, Col: 0})
	m, _ = key(m, typeKey(tea.KeyEnter))
	if m.hasBlockEdit() {
		t.Fatal("OpenTable failure must not open widget")
	}
	tb := m.testBlockForLine(3)
	if !m.layouts[tb].raw {
		t.Fatal("OpenTable failure should fall back to lazy-raw")
	}
}
