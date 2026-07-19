package doc

import (
	"os"
	"path/filepath"
	"testing"
)

// --- BG1: edit methods must clamp out-of-range Position instead of panicking.

func TestInsertClampsOutOfRangeLine(t *testing.T) {
	d := NewFromString("hello")
	// Line 50 clamps to the only line (0); Col defaults to 0, so the insert
	// lands at the start of that line.
	pos := d.Insert(Position{Line: 50}, "!")
	if pos != (Position{Line: 0, Col: 1}) {
		t.Fatalf("Insert() pos = %+v, want {0 1}", pos)
	}
	if d.Line(0) != "!hello" {
		t.Fatalf("Line(0) = %q, want !hello", d.Line(0))
	}
}

func TestInsertClampsOutOfRangeCol(t *testing.T) {
	d := NewFromString("hello")
	pos := d.Insert(Position{Line: 0, Col: 999}, "!")
	if pos != (Position{Line: 0, Col: 6}) {
		t.Fatalf("Insert() pos = %+v, want {0 6}", pos)
	}
	if d.Line(0) != "hello!" {
		t.Fatalf("Line(0) = %q, want hello!", d.Line(0))
	}
}

func TestDeleteForwardClampsOutOfRangeLine(t *testing.T) {
	d := NewFromString("hello")
	pos := d.DeleteForward(Position{Line: 50})
	if d.Line(0) != "ello" {
		t.Fatalf("Line(0) = %q, want ello", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("DeleteForward() pos = %+v, want {0 0}", pos)
	}
}

func TestDeleteBackwardClampsNegativeLine(t *testing.T) {
	d := NewFromString("hello")
	pos := d.DeleteBackward(Position{Line: -5})
	if d.Line(0) != "hello" {
		t.Fatalf("Line(0) = %q, want unchanged hello (clamped to start of doc)", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("DeleteBackward() pos = %+v, want {0 0}", pos)
	}
}

func TestDeleteRangeClampsBothEndpoints(t *testing.T) {
	d := NewFromString("hello")
	// Both endpoints are wildly out of range but clamp to the same point
	// (line 0, col 0), so this is a no-op rather than a panic.
	pos := d.DeleteRange(Position{Line: 50, Col: 0}, Position{Line: 99, Col: 0})
	if d.Line(0) != "hello" {
		t.Fatalf("Line(0) = %q, want unchanged hello", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("DeleteRange() pos = %+v, want {0 0}", pos)
	}
}

// --- BG2: Dirty() must reflect content diff, not version counters.

func TestDirtyReflectsContentNotVersionAfterUndo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := d.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	d.Insert(Position{Line: 0, Col: 5}, "X")
	if !d.Dirty() {
		t.Fatalf("Dirty() = false after edit, want true")
	}
	if _, ok := d.Undo(); !ok {
		t.Fatalf("Undo() ok = false")
	}
	if d.Content() != "hello\n" {
		t.Fatalf("Content() after undo = %q, want %q", d.Content(), "hello\n")
	}
	if d.Dirty() {
		t.Fatalf("Dirty() = true after undo back to saved content, want false")
	}
}

func TestDirtyTrueOnRealEdit(t *testing.T) {
	d := NewFromString("hello")
	if d.Dirty() {
		t.Fatalf("Dirty() = true right after NewFromString, want false")
	}
	d.Insert(Position{Line: 0, Col: 0}, "X")
	if !d.Dirty() {
		t.Fatalf("Dirty() = false after real edit, want true")
	}
}

// --- BG3: an empty document must render/save as 0 bytes, not "\n".

func TestEmptyDocContentIsEmptyString(t *testing.T) {
	d := NewFromString("")
	if d.Content() != "" {
		t.Fatalf("Content() = %q, want empty string", d.Content())
	}
}

func TestLoadNonexistentThenSaveForceWritesZeroBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := d.SaveForce(); err != nil {
		t.Fatalf("SaveForce() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("file size = %d, want 0", info.Size())
	}
}
