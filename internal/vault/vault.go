// Package vault implements a note index with wikilink resolution. It provides
// a simple interface to walk a directory tree, collect all markdown files, and
// resolve note references by name (case-insensitive) with tie-breaking on path length.
package vault

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Note represents a markdown file in the vault.
type Note struct {
	Name string // base filename without ".md"
	Path string // absolute path
}

// Vault is a note index built from a root directory.
type Vault struct {
	root     string
	notes    []Note
	warnings []string
}

// Open walks the root directory recursively, collecting all .md files (skipping
// directories whose names start with "."), and returns a new Vault. It returns
// an error if the root directory cannot be accessed.
func Open(root string) (*Vault, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	v := &Vault{root: absRoot}
	if err := v.scan(); err != nil {
		return nil, err
	}
	return v, nil
}

// Root returns the vault's root directory.
func (v *Vault) Root() string {
	return v.root
}

// List returns all notes in the vault, sorted by Name.
func (v *Vault) List() []Note {
	// Return a copy to prevent mutation
	notes := make([]Note, len(v.notes))
	copy(notes, v.notes)
	return notes
}

// Warnings returns human-readable warnings about paths that were skipped
// during the last scan (e.g. unreadable subdirectories), if any.
func (v *Vault) Warnings() []string {
	// Return a copy to prevent mutation
	warnings := make([]string, len(v.warnings))
	copy(warnings, v.warnings)
	return warnings
}

// Resolve resolves a note by name (case-insensitive). It accepts both "nota" and
// "nota.md". If multiple notes match (differ only in path), it returns the one with
// the shortest path; if paths are equal length, the lexicographically smaller path
// wins, so the result is deterministic. It returns ("", false) if no note matches.
func (v *Vault) Resolve(target string) (string, bool) {
	// Normalize target: strip a trailing known markdown extension
	// (".md" or ".markdown"), matched case-insensitively.
	if ext, ok := mdExt(target); ok {
		target = target[:len(target)-len(ext)]
	}

	// Match case-insensitively on Name across all notes uniformly.
	targetLower := strings.ToLower(target)
	var matches []Note
	for _, n := range v.notes {
		if strings.ToLower(n.Name) == targetLower {
			matches = append(matches, n)
		}
	}

	if len(matches) == 0 {
		return "", false
	}

	// Tie-break: shortest path wins; if paths are equal length, the
	// lexicographically smaller path wins deterministically.
	best := matches[0]
	for i := 1; i < len(matches); i++ {
		m := matches[i]
		if len(m.Path) < len(best.Path) {
			best = m
		} else if len(m.Path) == len(best.Path) && m.Path < best.Path {
			best = m
		}
	}

	return best.Path, true
}

// Rescan rebuilds the note index by walking the root directory again.
func (v *Vault) Rescan() error {
	v.notes = nil
	return v.scan()
}

// scan walks the root directory and populates v.notes, then sorts by Name.
func (v *Vault) scan() error {
	v.warnings = nil
	if err := filepath.WalkDir(v.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Don't abort the whole scan because one entry is unreadable
			// (e.g. a subdirectory with no read/execute permission): skip
			// it and keep indexing everything else that is readable.
			v.warnings = append(v.warnings, fmt.Sprintf("skipping %s: %v", path, err))
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories starting with "." — but not the root entry
		// itself, so a vault rooted at a dot-prefixed directory (e.g.
		// "~/.notes") is still indexed normally.
		if d.IsDir() {
			if path != v.root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Collect known markdown files (".md", ".markdown", matched
		// case-insensitively).
		if ext, ok := mdExt(d.Name()); ok {
			name := d.Name()[:len(d.Name())-len(ext)]
			v.notes = append(v.notes, Note{
				Name: name,
				Path: path,
			})
		}

		return nil
	}); err != nil {
		return err
	}

	// Sort by Name (case-insensitive), with Path as a deterministic
	// secondary key so case-duplicate names (e.g. "b" vs "B") always sort
	// in the same order regardless of sort algorithm or slice size.
	sort.SliceStable(v.notes, func(i, j int) bool {
		li, lj := strings.ToLower(v.notes[i].Name), strings.ToLower(v.notes[j].Name)
		if li != lj {
			return li < lj
		}
		return v.notes[i].Path < v.notes[j].Path
	})

	return nil
}

// mdExt returns the lowercase file extension of name if it is a recognized
// markdown extension (".md" or ".markdown"), matched case-insensitively,
// and whether it matched.
func mdExt(name string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".md" || ext == ".markdown" {
		return ext, true
	}
	return "", false
}
