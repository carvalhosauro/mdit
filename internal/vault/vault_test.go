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

	// Test Resolve("B.md") - accepts .md extension; case-insensitive
	// tie-break is uniform, so this also resolves to the shortest path
	// (sub/b.md), same as Resolve("b").
	path, ok = v.Resolve("B.md")
	if !ok {
		t.Error("Resolve(\"B.md\") returned false, want true")
	}
	if path != filepath.Join(root, "sub", "b.md") {
		t.Errorf("Resolve(\"B.md\") = %q, want %q", path, filepath.Join(root, "sub", "b.md"))
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
	_, ok = v.Resolve("new")
	if !ok {
		t.Error("Resolve(\"new\") returned false after Rescan, want true")
	}
}

// TestOpenAbsolutePath verifies that Open normalizes a relative root into an
// absolute one, so every derived Note.Path (and Root()) is absolute.
func TestOpenAbsolutePath(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "notes")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	createFile(t, root, "a.md", "content a")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWD); err != nil {
			t.Fatalf("Chdir back to %q failed: %v", origWD, err)
		}
	}()

	if err := os.Chdir(parent); err != nil {
		t.Fatalf("Chdir to %q failed: %v", parent, err)
	}

	v, err := Open("notes")
	if err != nil {
		t.Fatalf("Open(\"notes\") failed: %v", err)
	}

	if !filepath.IsAbs(v.Root()) {
		t.Errorf("Root() = %q, want absolute path", v.Root())
	}

	list := v.List()
	if len(list) != 1 {
		t.Fatalf("List() returned %d notes, want 1", len(list))
	}
	if !filepath.IsAbs(list[0].Path) {
		t.Errorf("Note.Path = %q, want absolute path", list[0].Path)
	}
}

// TestResolveTieBreakShortestPath verifies that Resolve considers ALL
// case-insensitive matches uniformly and picks the shortest path, even when
// one of the matches is an exact-case match at a deeper (longer) path.
func TestResolveTieBreakShortestPath(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "FOO.md", "content FOO")
	createFile(t, root, "aaaaaaaaaa/foo.md", "content foo")

	v, err := Open(root)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	path, ok := v.Resolve("foo")
	if !ok {
		t.Fatal("Resolve(\"foo\") returned false, want true")
	}
	want := filepath.Join(root, "FOO.md")
	if path != want {
		t.Errorf("Resolve(\"foo\") = %q, want %q (shortest path should win over exact-case match)", path, want)
	}
}

// TestOpenHiddenRoot verifies that the dot-directory skip does not apply to
// the root entry itself: a vault rooted at a dot-prefixed directory must
// still be indexed normally.
func TestOpenHiddenRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, ".notes")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	createFile(t, root, "a.md", "content a")
	createFile(t, root, "sub/b.md", "content b")

	v, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", root, err)
	}

	list := v.List()
	if len(list) != 2 {
		t.Fatalf("List() returned %d notes, want 2 (hidden root should still be indexed)", len(list))
	}
}

// TestListDeterministicOrder verifies that List() orders case-insensitive
// duplicate names deterministically (by path) rather than relying on an
// unstable sort's incidental behavior, across a slice large enough to defeat
// small-slice insertion-sort fallbacks.
func TestListDeterministicOrder(t *testing.T) {
	root := t.TempDir()

	names := []string{"dup", "Dup", "DUP", "duP", "dUp", "dUP", "DUp", "DuP"}
	var wantPaths []string
	for i := 0; i < 20; i++ {
		dir := filepath.Join(root, "d"+padded(i))
		name := names[i%len(names)]
		createFile(t, root, filepath.Join("d"+padded(i), name+".md"), "content")
		wantPaths = append(wantPaths, filepath.Join(dir, name+".md"))
	}

	v, err := Open(root)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Run List() multiple times and verify the order is identical and
	// matches the lexicographic path order every time.
	for attempt := 0; attempt < 3; attempt++ {
		list := v.List()
		if len(list) != len(wantPaths) {
			t.Fatalf("List() returned %d notes, want %d", len(list), len(wantPaths))
		}
		for i, note := range list {
			if note.Path != wantPaths[i] {
				t.Errorf("attempt %d: List()[%d].Path = %q, want %q", attempt, i, note.Path, wantPaths[i])
			}
		}
	}
}

// padded zero-pads an int to 2 digits so directory names sort lexically in
// numeric order (d00, d01, ..., d19).
func padded(i int) string {
	if i < 10 {
		return "0" + string(rune('0'+i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

// TestResolveEqualLengthTieBreak verifies that when two case-insensitive
// matches have equal-length paths, Resolve deterministically picks the
// lexicographically smaller path.
func TestResolveEqualLengthTieBreak(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "bb/tie.md", "content bb")
	createFile(t, root, "aa/tie.md", "content aa")

	v, err := Open(root)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	pathA := filepath.Join(root, "aa", "tie.md")
	pathB := filepath.Join(root, "bb", "tie.md")
	if len(pathA) != len(pathB) {
		t.Fatalf("test setup invalid: paths must be equal length, got %d and %d", len(pathA), len(pathB))
	}

	path, ok := v.Resolve("tie")
	if !ok {
		t.Fatal("Resolve(\"tie\") returned false, want true")
	}
	if path != pathA {
		t.Errorf("Resolve(\"tie\") = %q, want %q (lexicographically smaller path should win on equal-length tie)", path, pathA)
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
