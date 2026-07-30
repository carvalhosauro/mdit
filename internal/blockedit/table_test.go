package blockedit

import (
	"strings"
	"testing"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/mattn/go-runewidth"
)

func TestOpenTable_RoundTripBasic(t *testing.T) {
	raw := []string{
		"| Name | Age |",
		"| --- | --- |",
		"| Ada | 36 |",
		"| Bob | 42 |",
	}
	w, ok := OpenTable(raw, doc.Position{}, 0)
	if !ok {
		t.Fatal("OpenTable ok=false")
	}
	got := w.CommitLines()
	if len(got) != 4 {
		t.Fatalf("CommitLines len=%d want 4: %v", len(got), got)
	}
	// Round-trip: re-parse commit output must succeed and match cell content.
	w2, ok := OpenTable(got, doc.Position{}, 0)
	if !ok {
		t.Fatalf("re-open failed on %v", got)
	}
	got2 := w2.CommitLines()
	assertSameCells(t, got, got2)
}

func TestOpenTable_Alignments(t *testing.T) {
	raw := []string{
		"| L | C | R | N |",
		"| :--- | :---: | ---: | --- |",
		"| a | b | c | d |",
	}
	w, ok := OpenTable(raw, doc.Position{}, 0)
	if !ok {
		t.Fatal("OpenTable ok=false")
	}
	got := w.CommitLines()
	if len(got) < 2 {
		t.Fatalf("CommitLines=%v", got)
	}
	sep := got[1]
	if !strings.Contains(sep, ":---") || !strings.Contains(sep, ":---:") || !strings.Contains(sep, "---:") {
		t.Fatalf("separator missing align markers: %q", sep)
	}
	// none column should not have leading/trailing colon on its marker alone —
	// check the fourth segment is dashes without colons wrapping oddly.
	parts := splitPipeRow(sep)
	if len(parts) != 4 {
		t.Fatalf("sep parts=%v", parts)
	}
	trim0 := strings.TrimSpace(parts[0])
	if !strings.HasPrefix(trim0, ":") || strings.HasSuffix(trim0, ":") {
		t.Fatalf("left align marker = %q", trim0)
	}
	trim1 := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(trim1, ":") || !strings.HasSuffix(trim1, ":") {
		t.Fatalf("center align marker = %q", trim1)
	}
	trim2 := strings.TrimSpace(parts[2])
	if strings.HasPrefix(trim2, ":") || !strings.HasSuffix(trim2, ":") {
		t.Fatalf("right align marker = %q", trim2)
	}
	trim3 := strings.TrimSpace(parts[3])
	if strings.Contains(trim3, ":") {
		t.Fatalf("none align marker = %q", trim3)
	}
}

func TestOpenTable_TrimsCellEdges(t *testing.T) {
	raw := []string{
		"|  hello  |  world  |",
		"| --- | --- |",
		"|  x  |  y  |",
	}
	w, ok := OpenTable(raw, doc.Position{}, 0)
	if !ok {
		t.Fatal("OpenTable ok=false")
	}
	got := w.CommitLines()
	cells := splitPipeRow(got[0])
	if cells[0] != "hello" || cells[1] != "world" {
		t.Fatalf("header cells = %v", cells)
	}
	body := splitPipeRow(got[2])
	if body[0] != "x" || body[1] != "y" {
		t.Fatalf("body cells = %v", body)
	}
}

func TestOpenTable_NoSeparator(t *testing.T) {
	raw := []string{
		"| A | B |",
		"| x | y |",
	}
	if _, ok := OpenTable(raw, doc.Position{}, 0); ok {
		t.Fatal("OpenTable ok=true, want false without separator")
	}
}

func TestOpenTable_TooFewLines(t *testing.T) {
	if _, ok := OpenTable([]string{"| A |"}, doc.Position{}, 0); ok {
		t.Fatal("want ok=false")
	}
	if _, ok := OpenTable(nil, doc.Position{}, 0); ok {
		t.Fatal("want ok=false")
	}
}

func TestCommitLines_AutoResize(t *testing.T) {
	raw := []string{
		"| a | b |",
		"| --- | --- |",
		"| hello | x |",
	}
	w, ok := OpenTable(raw, doc.Position{}, 0)
	if !ok {
		t.Fatal("OpenTable ok=false")
	}
	got := w.CommitLines()
	sepParts := splitPipeRow(got[1])
	dashCount := strings.Count(strings.TrimSpace(sepParts[0]), "-")
	// "hello" is 5 runes wide; separator dashes (excluding align colons) ≥ 5,
	// and always ≥ 3 per GFM.
	if dashCount < 5 {
		t.Fatalf("col0 dashes=%d want ≥5 in %q", dashCount, got[1])
	}
	if runewidth.StringWidth("hello") != 5 {
		t.Fatalf("sanity: hello width")
	}
}

func TestExitCursor_StubReturnsBlockStart(t *testing.T) {
	raw := []string{
		"| A | B |",
		"| --- | --- |",
		"| 1 | 2 |",
	}
	w, ok := OpenTable(raw, doc.Position{Line: 5, Col: 3}, 4)
	if !ok {
		t.Fatal("OpenTable ok=false")
	}
	pos := w.ExitCursor(Cancel)
	if pos.Line != 5 || pos.Col != 3 {
		t.Fatalf("ExitCursor(Cancel)=%+v want pre-open cursor", pos)
	}
}

func assertSameCells(t *testing.T, a, b []string) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("len %d vs %d\n%v\n%v", len(a), len(b), a, b)
	}
	for i := range a {
		if i == 1 {
			continue // separator widths may stabilize identically after 2nd pass
		}
		ca, cb := splitPipeRow(a[i]), splitPipeRow(b[i])
		if len(ca) != len(cb) {
			t.Fatalf("row %d cells %v vs %v", i, ca, cb)
		}
		for j := range ca {
			if ca[j] != cb[j] {
				t.Fatalf("row %d col %d: %q vs %q", i, j, ca[j], cb[j])
			}
		}
	}
}
