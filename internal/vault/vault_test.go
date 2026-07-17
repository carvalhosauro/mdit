package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVault(t *testing.T) {
	root := t.TempDir()

	// Create test fixture: a.md, sub/b.md, sub/deep/B.md, .obsidian/x.md, c.txt
	createFile(t, root, "a.md", "content a")
	createFile(t, root, "sub/b.md", "content b")
	createFile(t, root, "sub/deep/B.md", "content B")
	createFile(t, root, ".obsidian/x.md", "content x")
	createFile(t, root, "c.txt", "content c")

	// Open vault
	v, err := Open(root)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Test Root()
	if v.Root() != root {
		t.Errorf("Root() = %q, want %q", v.Root(), root)
	}

	// Test List() - should return 3 notes sorted by Name: [a, b, B]
	list := v.List()
	if len(list) != 3 {
		t.Errorf("List() returned %d notes, want 3", len(list))
	}

	expectedNames := []string{"a", "b", "B"}
	if len(list) == 3 {
		for i, expected := range expectedNames {
			if list[i].Name != expected {
				t.Errorf("List()[%d].Name = %q, want %q", i, list[i].Name, expected)
			}
		}
	}

	// Test Resolve("b") - case-insensitive, shorter path wins (sub/b.md not sub/deep/B.md)
	path, ok := v.Resolve("b")
	if !ok {
		t.Error("Resolve(\"b\") returned false, want true")
	}
	if path != filepath.Join(root, "sub", "b.md") {
		t.Errorf("Resolve(\"b\") = %q, want %q", path, filepath.Join(root, "sub", "b.md"))
	}

	// Test Resolve("B.md") - accepts .md extension
	path, ok = v.Resolve("B.md")
	if !ok {
		t.Error("Resolve(\"B.md\") returned false, want true")
	}
	if path != filepath.Join(root, "sub", "deep", "B.md") {
		t.Errorf("Resolve(\"B.md\") = %q, want %q", path, filepath.Join(root, "sub", "deep", "B.md"))
	}

	// Test Resolve("a") works too
	path, ok = v.Resolve("a")
	if !ok {
		t.Error("Resolve(\"a\") returned false, want true")
	}
	if path != filepath.Join(root, "a.md") {
		t.Errorf("Resolve(\"a\") = %q, want %q", path, filepath.Join(root, "a.md"))
	}

	// Test Resolve("nope") - missing note
	path, ok = v.Resolve("nope")
	if ok {
		t.Error("Resolve(\"nope\") returned true, want false")
	}
	if path != "" {
		t.Errorf("Resolve(\"nope\") = %q, want \"\"", path)
	}

	// Test Rescan() picks up new file
	createFile(t, root, "new.md", "new content")
	if err := v.Rescan(); err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}
	list = v.List()
	if len(list) != 4 {
		t.Errorf("After Rescan(), List() returned %d notes, want 4", len(list))
	}

	// Check that "new" can be resolved
	path, ok = v.Resolve("new")
	if !ok {
		t.Error("Resolve(\"new\") returned false after Rescan, want true")
	}
}

// createFile is a test helper that creates a file at root/path with content.
// It creates parent directories as needed.
func createFile(t *testing.T, root, path, content string) {
	t.Helper()
	fullPath := filepath.Join(root, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll %q failed: %v", dir, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %q failed: %v", fullPath, err)
	}
}
