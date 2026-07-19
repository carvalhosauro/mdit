package doc

import (
	"os"
	"path/filepath"
	"testing"
)

// Dirty() memoizes by version; a Save changes dirtiness WITHOUT a version bump,
// so the memo must be invalidated on save. Guards the perf-memo optimization.
func TestDirtyMemoInvalidatedOnSave(t *testing.T) {
	p := filepath.Join(t.TempDir(), "n.md")
	if err := os.WriteFile(p, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Dirty() {
		t.Fatal("clean after load")
	}
	d.Insert(Position{Line: 0, Col: 5}, "X")
	if !d.Dirty() {
		t.Fatal("dirty after edit")
	}
	_ = d.Dirty() // exercise the memo at the same version
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if d.Dirty() {
		t.Fatal("save must report clean even though Version() did not change")
	}
}
