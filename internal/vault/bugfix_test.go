package vault

import (
	"os"
	"path/filepath"
	"testing"
)

// TestOpenSkipsUnreadableSubdir is a regression test for BG4: a single
// unreadable subdirectory must not abort the whole scan. Open should skip
// the unreadable subtree, still index everything readable, and record a
// warning about what was skipped.
func TestOpenSkipsUnreadableSubdir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits are not enforced")
	}

	root := t.TempDir()
	createFile(t, root, "a.md", "content a")

	sub := filepath.Join(root, "secret")
	createFile(t, root, "secret/hidden.md", "content hidden")
	if err := os.Chmod(sub, 0o000); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}
	defer func() { _ = os.Chmod(sub, 0o755) }()

	v, err := Open(root)
	if err != nil {
		t.Fatalf("Open failed: %v, want no error (unreadable subdir should not abort scan)", err)
	}

	list := v.List()
	if len(list) != 1 {
		t.Fatalf("List() returned %d notes, want 1 (only a.md readable)", len(list))
	}
	if list[0].Name != "a" {
		t.Errorf("List()[0].Name = %q, want %q", list[0].Name, "a")
	}

	warnings := v.Warnings()
	if len(warnings) == 0 {
		t.Error("Warnings() returned empty, want at least one warning about the unreadable subdir")
	}

	// Fix permissions and Rescan: warnings should reset (not accumulate),
	// and hidden.md should now be indexed.
	if err := os.Chmod(sub, 0o755); err != nil {
		t.Fatalf("Chmod restore failed: %v", err)
	}
	if err := v.Rescan(); err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}
	if len(v.Warnings()) != 0 {
		t.Errorf("Warnings() after Rescan with fixed permissions = %v, want empty", v.Warnings())
	}
	list = v.List()
	if len(list) != 2 {
		t.Errorf("List() after Rescan returned %d notes, want 2", len(list))
	}
}

// TestResolveIndexesMarkdownVariants is a regression test for BG5: files
// with a ".markdown" extension or non-lowercase ".md"/".Md" extensions must
// be indexed and resolvable just like plain lowercase ".md" files.
func TestResolveIndexesMarkdownVariants(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "foo.markdown", "content foo")
	createFile(t, root, "bar.Md", "content bar")
	createFile(t, root, "baz.md", "content baz")

	v, err := Open(root)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	list := v.List()
	if len(list) != 3 {
		t.Fatalf("List() returned %d notes, want 3 (foo.markdown, bar.Md, baz.md should all be indexed)", len(list))
	}

	for _, name := range []string{"foo", "bar", "baz"} {
		path, ok := v.Resolve(name)
		if !ok {
			t.Errorf("Resolve(%q) returned false, want true", name)
			continue
		}
		if path == "" {
			t.Errorf("Resolve(%q) returned empty path", name)
		}
	}
}
