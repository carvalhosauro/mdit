package doc

import (
	"math/rand"
	"testing"
	"time"
)

func TestUndoRedoInsert(t *testing.T) {
	d := NewFromString("hello")
	d.Insert(Position{Line: 0, Col: 5}, " world")

	pos, ok := d.Undo()
	if !ok {
		t.Fatalf("Undo() ok = false, want true")
	}
	if d.Line(0) != "hello" {
		t.Fatalf("Line(0) after undo = %q, want hello", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 5}) {
		t.Fatalf("Undo() pos = %+v, want {0 5}", pos)
	}

	pos, ok = d.Redo()
	if !ok {
		t.Fatalf("Redo() ok = false")
	}
	if d.Line(0) != "hello world" {
		t.Fatalf("Line(0) after redo = %q", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 11}) {
		t.Fatalf("Redo() pos = %+v, want {0 11}", pos)
	}
}

func TestUndoEmptyStackReturnsFalse(t *testing.T) {
	d := NewFromString("hello")
	if _, ok := d.Undo(); ok {
		t.Fatalf("Undo() ok = true, want false on empty stack")
	}
}

func TestRedoEmptyStackReturnsFalse(t *testing.T) {
	d := NewFromString("hello")
	if _, ok := d.Redo(); ok {
		t.Fatalf("Redo() ok = true, want false on empty stack")
	}
}

func TestUndoDeleteMultiline(t *testing.T) {
	d := NewFromString("hello\nworld\nfoo")
	d.DeleteRange(Position{Line: 0, Col: 3}, Position{Line: 2, Col: 1})
	if d.Content() != "heloo\n" {
		t.Fatalf("Content() after delete = %q", d.Content())
	}

	pos, ok := d.Undo()
	if !ok {
		t.Fatalf("Undo() ok = false")
	}
	want := "hello\nworld\nfoo\n"
	if d.Content() != want {
		t.Fatalf("Content() after undo = %q, want %q", d.Content(), want)
	}
	if pos != (Position{Line: 2, Col: 1}) {
		t.Fatalf("Undo() pos = %+v, want {2 1}", pos)
	}
}

func TestRedoInvalidatedByNewEdit(t *testing.T) {
	d := NewFromString("hello")
	d.Insert(Position{Line: 0, Col: 5}, " world")
	d.Undo()
	d.Insert(Position{Line: 0, Col: 5}, "!")

	if _, ok := d.Redo(); ok {
		t.Fatalf("Redo() ok = true, want false after new edit invalidated redo stack")
	}
	if d.Line(0) != "hello!" {
		t.Fatalf("Line(0) = %q, want hello!", d.Line(0))
	}
}

func TestUndoCoalescesRapidSingleRuneInserts(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cur := base
	d := NewFromString("")
	d.now = func() time.Time { return cur }

	d.Insert(Position{Line: 0, Col: 0}, "a")
	cur = cur.Add(100 * time.Millisecond)
	d.Insert(Position{Line: 0, Col: 1}, "b")

	if d.Line(0) != "ab" {
		t.Fatalf("Line(0) = %q, want ab", d.Line(0))
	}

	pos, ok := d.Undo()
	if !ok {
		t.Fatalf("Undo() ok = false")
	}
	if d.Line(0) != "" {
		t.Fatalf("Line(0) after single coalesced undo = %q, want empty", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("pos = %+v, want {0 0}", pos)
	}
	if _, ok := d.Undo(); ok {
		t.Fatalf("expected only one undo group for coalesced inserts")
	}
}

func TestUndoDoesNotCoalesceAcrossTimeGap(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cur := base
	d := NewFromString("")
	d.now = func() time.Time { return cur }

	d.Insert(Position{Line: 0, Col: 0}, "a")
	cur = cur.Add(1 * time.Second)
	d.Insert(Position{Line: 0, Col: 1}, "b")

	if d.Line(0) != "ab" {
		t.Fatalf("Line(0) = %q, want ab", d.Line(0))
	}

	if _, ok := d.Undo(); !ok {
		t.Fatalf("expected first undo to succeed")
	}
	if d.Line(0) != "a" {
		t.Fatalf("Line(0) after first undo = %q, want a (only 2nd insert undone)", d.Line(0))
	}

	if _, ok := d.Undo(); !ok {
		t.Fatalf("expected second undo to succeed")
	}
	if d.Line(0) != "" {
		t.Fatalf("Line(0) after second undo = %q, want empty", d.Line(0))
	}
}

func TestUndoDoesNotCoalesceNonContiguousInserts(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := NewFromString("ac")
	d.now = func() time.Time { return base }

	d.Insert(Position{Line: 0, Col: 0}, "X") // "Xac"
	d.Insert(Position{Line: 0, Col: 3}, "Y") // "XacY" — not contiguous with the first insert

	if d.Line(0) != "XacY" {
		t.Fatalf("Line(0) = %q, want XacY", d.Line(0))
	}
	if _, ok := d.Undo(); !ok {
		t.Fatalf("expected first undo to succeed")
	}
	if d.Line(0) != "Xac" {
		t.Fatalf("Line(0) after first undo = %q, want Xac (non-contiguous inserts must not coalesce)", d.Line(0))
	}
}

// TestUndoAllRestoresOriginalContentRandomOps is the property test required
// by the brief: a fixed-seed sequence of 500 random insert/delete operations
// at valid positions, followed by undoing until the stack is empty, must
// restore Content() exactly to the original text.
func TestUndoAllRestoresOriginalContentRandomOps(t *testing.T) {
	const original = "the quick brown fox\njumps over\nthe lazy dog\n\nhéllo wörld"
	d := NewFromString(original)

	rng := rand.New(rand.NewSource(42))
	const numOps = 500
	choices := []string{"a", "bb", "x\n", "y", "z\nq", " ", "1", "\n", "é"}

	for i := 0; i < numOps; i++ {
		lineCount := d.LineCount()
		line := rng.Intn(lineCount)
		lineLen := len([]rune(d.Line(line)))
		col := rng.Intn(lineLen + 1)
		p := Position{Line: line, Col: col}

		switch rng.Intn(3) {
		case 0:
			d.Insert(p, choices[rng.Intn(len(choices))])
		case 1:
			d.DeleteBackward(p)
		case 2:
			d.DeleteForward(p)
		}
	}

	undone := 0
	for {
		if _, ok := d.Undo(); !ok {
			break
		}
		undone++
	}

	want := NewFromString(original).Content()
	if d.Content() != want {
		t.Fatalf("Content() after undoing all %d ops = %q, want %q", undone, d.Content(), want)
	}
}
