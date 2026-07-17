package doc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadNonexistentReturnsEmptyDoc(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.md")

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if d.LineCount() != 1 || d.Line(0) != "" {
		t.Fatalf("want empty doc with 1 empty line, got lines=%v", d.Lines())
	}
	if d.Path() != path {
		t.Fatalf("Path() = %q, want %q", d.Path(), path)
	}
	if d.Dirty() {
		t.Fatalf("Dirty() = true, want false right after Load")
	}
}

func TestLoadEditSaveReloadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	d.Insert(Position{Line: 0, Col: 5}, "!")
	if err := d.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload Load() error = %v", err)
	}
	want := "hello!\nworld\n"
	if reloaded.Content() != want {
		t.Fatalf("reloaded Content() = %q, want %q", reloaded.Content(), want)
	}
}

func TestSaveWithoutExternalChangeSucceeds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("a\n"), 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	d.Insert(Position{Line: 0, Col: 1}, "b")
	if err := d.Save(); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
}

func TestSaveDetectsExternalChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("a\n"), 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Simulate an external modification after Load, forcing mtime forward
	// well past filesystem timestamp granularity.
	future := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(path, []byte("external change\n"), 0o644); err != nil {
		t.Fatalf("external write: %v", err)
	}
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	d.Insert(Position{Line: 0, Col: 1}, "b")
	err = d.Save()
	if !errors.Is(err, ErrExternalChange) {
		t.Fatalf("Save() error = %v, want ErrExternalChange", err)
	}

	// The failed save must not have touched the file.
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if string(data) != "external change\n" {
		t.Fatalf("file content = %q, want unchanged external content", data)
	}
}

func TestSaveForceOverwritesDespiteExternalChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("a\n"), 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	future := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(path, []byte("external change\n"), 0o644); err != nil {
		t.Fatalf("external write: %v", err)
	}
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	d.Insert(Position{Line: 0, Col: 1}, "b")
	if err := d.SaveForce(); err != nil {
		t.Fatalf("SaveForce() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != d.Content() {
		t.Fatalf("file content = %q, want %q", data, d.Content())
	}
}

func TestSaveClearsDirtyAndUpdatesMtime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("a\n"), 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	d.Insert(Position{Line: 0, Col: 1}, "b")
	if !d.Dirty() {
		t.Fatalf("Dirty() = false after edit, want true")
	}
	if err := d.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if d.Dirty() {
		t.Fatalf("Dirty() = true after Save, want false")
	}

	// A second edit+save cycle must still succeed: the internal mtime was
	// updated to match what our own write produced.
	d.Insert(Position{Line: 0, Col: 2}, "c")
	if !d.Dirty() {
		t.Fatalf("Dirty() = false after second edit, want true")
	}
	if err := d.Save(); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	if d.Dirty() {
		t.Fatalf("Dirty() = true after second Save, want false")
	}
}

func TestLoadNonexistentFileThenSaveCreatesIt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.md")

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("file should not exist yet before first Save")
	}

	d.Insert(Position{Line: 0, Col: 0}, "fresh content")
	if err := d.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after first Save: %v", err)
	}
	want := "fresh content\n"
	if string(data) != want {
		t.Fatalf("file content = %q, want %q", data, want)
	}
	if d.Dirty() {
		t.Fatalf("Dirty() = true after first Save, want false")
	}
}

func TestSaveOnPathlessDocReturnsError(t *testing.T) {
	d := NewFromString("hello")
	if err := d.Save(); err == nil {
		t.Fatalf("Save() error = nil, want error for pathless doc")
	}
	if err := d.SaveForce(); err == nil {
		t.Fatalf("SaveForce() error = nil, want error for pathless doc")
	}
}
