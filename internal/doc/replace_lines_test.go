package doc

import (
	"testing"
	"time"
	"unicode/utf8"
)

func TestReplaceLines_OneForOne(t *testing.T) {
	d := NewFromString("a\nb\nc")
	v0 := d.Version()
	pos := d.ReplaceLines(1, 1, []string{"B"})
	if d.LineCount() != 3 {
		t.Fatalf("LineCount() = %d, want 3", d.LineCount())
	}
	if got := d.Lines(); got[0] != "a" || got[1] != "B" || got[2] != "c" {
		t.Fatalf("Lines() = %v, want [a B c]", got)
	}
	if pos != (Position{Line: 1, Col: 0}) {
		t.Fatalf("pos = %+v, want {1 0}", pos)
	}
	if d.Version() != v0+1 {
		t.Fatalf("Version() = %d, want %d", d.Version(), v0+1)
	}
}

func TestReplaceLines_Grow(t *testing.T) {
	d := NewFromString("a\nb\nc\nd")
	pos := d.ReplaceLines(1, 3, []string{"X", "Y", "Z", "W", "V"})
	want := []string{"a", "X", "Y", "Z", "W", "V"}
	if got := d.Lines(); len(got) != len(want) {
		t.Fatalf("Lines() = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Lines() = %v, want %v", got, want)
			}
		}
	}
	if pos != (Position{Line: 1, Col: 0}) {
		t.Fatalf("pos = %+v, want {1 0}", pos)
	}
}

func TestReplaceLines_Shrink(t *testing.T) {
	d := NewFromString("a\nb\nc\nd")
	pos := d.ReplaceLines(1, 3, []string{"X"})
	want := []string{"a", "X"}
	if got := d.Lines(); len(got) != len(want) {
		t.Fatalf("Lines() = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Lines() = %v, want %v", got, want)
			}
		}
	}
	if pos != (Position{Line: 1, Col: 0}) {
		t.Fatalf("pos = %+v, want {1 0}", pos)
	}
}

func TestReplaceLines_DeleteRangeNil(t *testing.T) {
	d := NewFromString("a\nb\nc\nd")
	pos := d.ReplaceLines(1, 2, nil)
	want := []string{"a", "d"}
	if got := d.Lines(); len(got) != len(want) {
		t.Fatalf("Lines() = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Lines() = %v, want %v", got, want)
			}
		}
	}
	if pos != (Position{Line: 1, Col: 0}) {
		t.Fatalf("pos = %+v, want {1 0} (clamped to remaining)", pos)
	}
}

func TestReplaceLines_DeleteRangeEmpty(t *testing.T) {
	d := NewFromString("a\nb\nc\nd")
	_ = d.ReplaceLines(1, 2, []string{})
	want := []string{"a", "d"}
	if got := d.Lines(); len(got) != len(want) {
		t.Fatalf("Lines() = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Lines() = %v, want %v", got, want)
			}
		}
	}
}

func TestReplaceLines_AtEnd(t *testing.T) {
	d := NewFromString("a\nb\nc")
	pos := d.ReplaceLines(2, 2, []string{"C1", "C2"})
	want := []string{"a", "b", "C1", "C2"}
	if got := d.Lines(); len(got) != len(want) {
		t.Fatalf("Lines() = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Lines() = %v, want %v", got, want)
			}
		}
	}
	if pos != (Position{Line: 2, Col: 0}) {
		t.Fatalf("pos = %+v, want {2 0}", pos)
	}
}

func TestReplaceLines_StartEqualsEnd(t *testing.T) {
	d := NewFromString("only")
	pos := d.ReplaceLines(0, 0, []string{"replaced"})
	if d.LineCount() != 1 || d.Line(0) != "replaced" {
		t.Fatalf("Lines() = %v", d.Lines())
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("pos = %+v, want {0 0}", pos)
	}
}

func TestReplaceLines_Unicode(t *testing.T) {
	d := NewFromString("α\nββ\nγ")
	pos := d.ReplaceLines(1, 1, []string{"δελτα"})
	if d.Line(1) != "δελτα" {
		t.Fatalf("Line(1) = %q", d.Line(1))
	}
	if pos != (Position{Line: 1, Col: 0}) {
		t.Fatalf("pos = %+v", pos)
	}
	if utf8.RuneCountInString(d.Line(1)) != 5 {
		t.Fatalf("rune count = %d", utf8.RuneCountInString(d.Line(1)))
	}
}

func TestReplaceLines_VersionExactlyOne(t *testing.T) {
	d := NewFromString("a\nb\nc")
	v0 := d.Version()
	d.ReplaceLines(0, 2, []string{"x", "y"})
	if d.Version() != v0+1 {
		t.Fatalf("Version() = %d, want %d", d.Version(), v0+1)
	}
}

func TestReplaceLines_UndoRedo(t *testing.T) {
	d := NewFromString("a\nb\nc")
	before := d.Content()
	d.ReplaceLines(1, 1, []string{"B1", "B2"})
	after := d.Content()
	if after == before {
		t.Fatalf("content unchanged after replace")
	}

	pos, ok := d.Undo()
	if !ok {
		t.Fatalf("Undo() ok = false")
	}
	if d.Content() != before {
		t.Fatalf("Content() after undo = %q, want %q", d.Content(), before)
	}
	if pos != (Position{Line: 1, Col: 0}) {
		// cursorBefore recorded at start of replaced range
		t.Fatalf("Undo() pos = %+v, want {1 0}", pos)
	}

	pos, ok = d.Redo()
	if !ok {
		t.Fatalf("Redo() ok = false")
	}
	if d.Content() != after {
		t.Fatalf("Content() after redo = %q, want %q", d.Content(), after)
	}
	if pos != (Position{Line: 1, Col: 0}) {
		t.Fatalf("Redo() pos = %+v, want {1 0}", pos)
	}
}

func TestReplaceLines_NewEditClearsRedo(t *testing.T) {
	d := NewFromString("a\nb\nc")
	d.ReplaceLines(1, 1, []string{"B"})
	d.Undo()
	d.ReplaceLines(0, 0, []string{"A"})
	if _, ok := d.Redo(); ok {
		t.Fatalf("Redo() ok = true, want false after new edit")
	}
}

func TestReplaceLines_NotCoalescibleWithTyping(t *testing.T) {
	base := mustParseTime(t)
	d := NewFromString("a\nb")
	d.now = func() time.Time { return base }

	d.Insert(Position{Line: 0, Col: 1}, "x") // coalescible typing
	d.ReplaceLines(1, 1, []string{"B"})
	d.Insert(Position{Line: 0, Col: 2}, "y")

	// ReplaceLines must be its own undo group; typing after must not merge into it.
	// Undo typing "y", then undo ReplaceLines, then undo "x".
	if _, ok := d.Undo(); !ok {
		t.Fatal("undo 1")
	}
	if d.Line(1) != "B" {
		t.Fatalf("after undo typing, Line(1)=%q want B (ReplaceLines still applied)", d.Line(1))
	}
	if _, ok := d.Undo(); !ok {
		t.Fatal("undo 2")
	}
	if d.Line(1) != "b" {
		t.Fatalf("after undo ReplaceLines, Line(1)=%q want b", d.Line(1))
	}
}

func TestReplaceLines_ClampOutOfBounds(t *testing.T) {
	d := NewFromString("a\nb\nc")
	// No panic; clamp into valid range.
	pos := d.ReplaceLines(-5, 99, []string{"ALL"})
	if d.LineCount() != 1 || d.Line(0) != "ALL" {
		t.Fatalf("Lines() = %v", d.Lines())
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("pos = %+v", pos)
	}
}

func TestReplaceLines_DeleteEntireDoc(t *testing.T) {
	d := NewFromString("a\nb")
	pos := d.ReplaceLines(0, 1, nil)
	if d.Content() != "" {
		t.Fatalf("Content() = %q, want empty", d.Content())
	}
	if d.LineCount() != 1 || d.Line(0) != "" {
		t.Fatalf("Lines() = %v", d.Lines())
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("pos = %+v", pos)
	}
}

func mustParseTime(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}
