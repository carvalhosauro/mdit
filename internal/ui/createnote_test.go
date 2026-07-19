package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/theme"
	"github.com/carvalhosauro/mdit/internal/vault"
)

// P4 safety: a create-note target must not escape the vault via "..".
func TestDoCreateNote_RejectsTraversal(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "home.md")
	if err := os.WriteFile(notePath, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	d, err := doc.Load(notePath)
	if err != nil {
		t.Fatal(err)
	}
	a := NewApp(v, d, theme.DefaultDark())

	a.doCreateNote("../../evil")
	if a.statusErr == "" {
		t.Error("traversal target should be rejected with an error")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "evil.md")); err == nil {
		t.Error("traversal note was created outside the vault")
	}

	// A normal target still succeeds inside the note's directory.
	a.doCreateNote("child")
	if _, err := os.Stat(filepath.Join(root, "child.md")); err != nil {
		t.Errorf("normal create should succeed: %v", err)
	}
}
