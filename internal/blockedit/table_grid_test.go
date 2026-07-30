package blockedit

import (
	"strings"
	"testing"

	"github.com/carvalhosauro/mdit/internal/doc"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTableGrid_AddRowBelow(t *testing.T) {
	w := fixture2x2(t)
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyEnter}) // focus body row 0
	before := len(w.CommitLines())
	w, _, sig := w.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftDown})
	if sig != Continue {
		t.Fatalf("sig=%v", sig)
	}
	got := w.CommitLines()
	if len(got) != before+1 {
		t.Fatalf("CommitLines len=%d want %d: %v", len(got), before+1, got)
	}
	last := splitPipeRow(got[len(got)-1])
	if last[0] != "" || last[1] != "" {
		t.Fatalf("new row cells should be empty, got %v", last)
	}
}

func TestTableGrid_DeleteBodyRow(t *testing.T) {
	raw := []string{
		"| A | B |",
		"| --- | --- |",
		"| 1 | 2 |",
		"| 3 | 4 |",
	}
	w, ok := OpenTable(raw, doc.Position{}, 0)
	if !ok {
		t.Fatal("OpenTable")
	}
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyEnter}) // body row 0
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftUp})
	got := w.CommitLines()
	if len(got) != 3 {
		t.Fatalf("len=%d want 3 after delete: %v", len(got), got)
	}
	body := splitPipeRow(got[2])
	if body[0] != "3" || body[1] != "4" {
		t.Fatalf("remaining body=%v want 3/4", body)
	}
}

func TestTableGrid_DeleteLastBodyClears(t *testing.T) {
	w := fixture2x2(t)
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyEnter})
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftUp})
	got := w.CommitLines()
	if len(got) != 3 {
		t.Fatalf("len=%d want 3 (grid kept): %v", len(got), got)
	}
	body := splitPipeRow(got[2])
	if body[0] != "" || body[1] != "" {
		t.Fatalf("last body should clear, got %v", body)
	}
}

func TestTableGrid_HeaderNeverRemoved(t *testing.T) {
	w := fixture2x2(t)
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftUp}) // focus still header
	got := w.CommitLines()
	if len(got) != 3 {
		t.Fatalf("header delete must be no-op, got %v", got)
	}
	if splitPipeRow(got[0])[0] != "A" {
		t.Fatalf("header mutated: %v", got)
	}
}

func TestTableGrid_AddColumnRight(t *testing.T) {
	w := fixture2x2(t)
	// Move to last column so "insert right" appends.
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyTab})
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftRight})
	got := w.CommitLines()
	header := splitPipeRow(got[0])
	if len(header) != 3 {
		t.Fatalf("header cols=%d want 3: %v", len(header), header)
	}
	if header[2] != "" {
		t.Fatalf("new header cell should be empty, got %v", header)
	}
	sep := splitPipeRow(got[1])
	if len(sep) != 3 {
		t.Fatalf("sep cols=%d want 3", len(sep))
	}
	body := splitPipeRow(got[2])
	if len(body) != 3 || body[2] != "" {
		t.Fatalf("body=%v", body)
	}
	// Auto-resize still produces valid separator dashes.
	if strings.Count(strings.TrimSpace(sep[2]), "-") < 3 {
		t.Fatalf("new col separator too short: %q", got[1])
	}
}

func TestTableGrid_DeleteColumnMinOne(t *testing.T) {
	w := fixture2x2(t)
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftLeft})
	got := w.CommitLines()
	if len(splitPipeRow(got[0])) != 1 {
		t.Fatalf("want 1 col after delete: %v", got)
	}
	w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftLeft})
	got = w.CommitLines()
	if len(splitPipeRow(got[0])) != 1 {
		t.Fatalf("min 1 col: %v", got)
	}
}

func TestTableGrid_AlignCycle(t *testing.T) {
	w := fixture2x2(t)
	// none → left → center → right → none
	expect := []string{":", ":---:", "---:", "---"}
	markers := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		w, _, _ = w.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
		sep := strings.TrimSpace(splitPipeRow(w.CommitLines()[1])[0])
		markers = append(markers, sep)
	}
	// Check progression qualitatively.
	if !strings.HasPrefix(markers[0], ":") || strings.HasSuffix(markers[0], ":") {
		t.Fatalf("after 1×Ctrl+L want left, got %q", markers[0])
	}
	if !strings.HasPrefix(markers[1], ":") || !strings.HasSuffix(markers[1], ":") {
		t.Fatalf("after 2× want center, got %q", markers[1])
	}
	if strings.HasPrefix(markers[2], ":") || !strings.HasSuffix(markers[2], ":") {
		t.Fatalf("after 3× want right, got %q", markers[2])
	}
	if strings.Contains(markers[3], ":") {
		t.Fatalf("after 4× want none, got %q", markers[3])
	}
	_ = expect
}
