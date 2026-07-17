// Package vault implements a note index with wikilink resolution. It provides
// a simple interface to walk a directory tree, collect all markdown files, and
// resolve note references by name (case-insensitive) with tie-breaking on path length.
package vault

import (
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
	root  string
	notes []Note
}

// Open walks the root directory recursively, collecting all .md files (skipping
// directories whose names start with "."), and returns a new Vault. It returns
// an error if the root directory cannot be accessed.
func Open(root string) (*Vault, error) {
	v := &Vault{root: root}
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

// Resolve resolves a note by name (case-insensitive). It accepts both "nota" and
// "nota.md". If multiple notes match (differ only in path), it returns the one with
// the shortest path. It returns ("", false) if no note matches.
func (v *Vault) Resolve(target string) (string, bool) {
	// Normalize target: remove .md suffix if present
	target = strings.TrimSuffix(target, ".md")
	target = strings.TrimSuffix(target, ".MD")

	// First, try exact case-sensitive match
	for _, n := range v.notes {
		if n.Name == target {
			return n.Path, true
		}
	}

	// Fall back to case-insensitive match
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

	// If multiple matches, return the one with shortest path
	best := matches[0]
	for i := 1; i < len(matches); i++ {
		if len(matches[i].Path) < len(best.Path) {
			best = matches[i]
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
	if err := filepath.WalkDir(v.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories starting with "."
		if d.IsDir() {
			if d.Name()[0] == '.' {
				return filepath.SkipDir
			}
			return nil
		}

		// Collect .md files
		if strings.HasSuffix(d.Name(), ".md") {
			name := d.Name()[:len(d.Name())-3] // remove .md suffix
			v.notes = append(v.notes, Note{
				Name: name,
				Path: path,
			})
		}

		return nil
	}); err != nil {
		return err
	}

	// Sort by Name (case-insensitive)
	sort.Slice(v.notes, func(i, j int) bool {
		return strings.ToLower(v.notes[i].Name) < strings.ToLower(v.notes[j].Name)
	})

	return nil
}
