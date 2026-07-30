package blockedit

import (
	"strings"
	"testing"

	"github.com/carvalhosauro/mdit/internal/doc"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func fixture2x2(t *testing.T) Widget {
	t.Helper()
	raw := []string{
		"| A | B |",
		"| --- | --- |",
		"| 1 | 2 |",
	}
	w, ok := OpenTable(raw, doc.Position{Line: 0, Col: 0}, 0)
	if !ok {
		t.Fatal("OpenTable failed")
	}
	return w
}

func TestTableKeys_TabCycles(t *testing.T) {
	w := fixture2x2(t)
	tw := mustTable(t, w)
	if tw.focusRow != 0 || tw.focusCol != 0 {
		t.Fatalf("initial focus = (%d,%d), want (0,0)", tw.focusRow, tw.focusCol)
	}

	seq := []struct{ row, col int }{{0, 1}, {1, 0}, {1, 1}, {0, 0}}
	for i, want := range seq {
		var sig Signal
		w, _, sig = w.Update(tea.KeyMsg{Type: tea.KeyTab})
		if sig != Continue {
			t.Fatalf("tab %d: sig=%v", i, sig)
		}
		tw = mustTable(t, w)
		if tw.focusRow != want.row || tw.focusCol != want.col {
			t.Fatalf("tab %d: focus=(%d,%d) want (%d,%d)", i, tw.focusRow, tw.focusCol, want.row, want.col)
		}
	}
}

func TestTableKeys_ShiftTabReverse(t *testing.T) {
	w := fixture2x2(t)
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	tw := mustTable(t, w)
	if tw.focusRow != 1 || tw.focusCol != 1 {
		t.Fatalf("shift+tab from (0,0) → (%d,%d), want (1,1)", tw.focusRow, tw.focusCol)
	}
}

func TestTableKeys_TypeIntoCell(t *testing.T) {
	w := fixture2x2(t)
	// Place cursor at start of cell so "Hi" prefixes.
	tw := mustTable(t, w)
	tw.cellCol = 0
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	got := w.CommitLines()
	header := splitPipeRow(got[0])
	if !strings.Contains(header[0], "Hi") {
		t.Fatalf("header[0]=%q want to contain Hi", header[0])
	}
	if header[1] != "B" {
		t.Fatalf("header[1]=%q unchanged", header[1])
	}
	body := splitPipeRow(got[2])
	if body[0] != "1" || body[1] != "2" {
		t.Fatalf("body mutated: %v", body)
	}
}

func TestTableKeys_EscCancel(t *testing.T) {
	w := fixture2x2(t)
	_, _, sig := w.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if sig != Cancel {
		t.Fatalf("sig=%v want Cancel", sig)
	}
}

func TestTableKeys_EnterMovesDown(t *testing.T) {
	w := fixture2x2(t)
	w, _, sig := w.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if sig != Continue {
		t.Fatalf("sig=%v", sig)
	}
	tw := mustTable(t, w)
	if tw.focusRow != 1 || tw.focusCol != 0 {
		t.Fatalf("focus=(%d,%d) want (1,0)", tw.focusRow, tw.focusCol)
	}
}

func TestTableKeys_LinesShowsCellsAndFocus(t *testing.T) {
	w := fixture2x2(t)
	lines := w.Lines(80)
	joined := ansi.Strip(strings.Join(lines, "\n"))
	for _, want := range []string{"A", "B", "1", "2"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("Lines missing %q in %q", want, joined)
		}
	}
	rawJoin := strings.Join(lines, "\n")
	if rawJoin == joined {
		t.Fatalf("expected ANSI styling on focused cell, got plain %q", joined)
	}
}

func mustTable(t *testing.T, w Widget) *tableWidget {
	t.Helper()
	tw, ok := w.(*tableWidget)
	if !ok {
		t.Fatalf("want *tableWidget, got %T", w)
	}
	return tw
}
