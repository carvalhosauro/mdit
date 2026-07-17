package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/theme"
	"github.com/carvalhosauro/mdit/internal/ui"
	"github.com/carvalhosauro/mdit/internal/vault"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		fmt.Println("mdit", version)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: mdit <file.md|folder>")
		os.Exit(1)
	}

	v, initial, err := openArg(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	app := ui.NewApp(v, initial, theme.DefaultDark())

	// Panic guard: dump the buffer next to the open file, then re-panic.
	// Bubble Tea restores the terminal on its own unwind path.
	defer func() {
		if r := recover(); r != nil {
			if d := app.Doc(); d != nil {
				path := d.Path()
				if path == "" {
					path = "untitled.md"
				}
				_ = os.WriteFile(path+".mdit-recover", []byte(d.Content()), 0o644)
			}
			panic(r)
		}
	}()

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// openArg resolves a file or directory argument into a vault and the document
// to open. A .md file uses its parent directory as the vault root; a directory
// is the vault root and opens the newest note (by mtime) or untitled.md.
func openArg(arg string) (*vault.Vault, *doc.Document, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Stat(abs)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}

	var root, file string
	if err == nil && info.IsDir() {
		root = abs
		v, err := vault.Open(root)
		if err != nil {
			return nil, nil, err
		}
		file, err = newestNotePath(v)
		if err != nil {
			return nil, nil, err
		}
		d, err := doc.Load(file)
		if err != nil {
			return nil, nil, err
		}
		return v, d, nil
	}

	// File path (existing or to-be-created).
	if filepath.Ext(abs) == "" && err == nil {
		// Non-.md existing file — treat as error.
		return nil, nil, fmt.Errorf("not a markdown file or directory: %s", abs)
	}
	root = filepath.Dir(abs)
	file = abs
	v, err := vault.Open(root)
	if err != nil {
		return nil, nil, err
	}
	d, err := doc.Load(file)
	if err != nil {
		return nil, nil, err
	}
	return v, d, nil
}

func newestNotePath(v *vault.Vault) (string, error) {
	notes := v.List()
	if len(notes) == 0 {
		return filepath.Join(v.Root(), "untitled.md"), nil
	}
	best := notes[0].Path
	bestTime := time.Time{}
	for _, n := range notes {
		info, err := os.Stat(n.Path)
		if err != nil {
			continue
		}
		if info.ModTime().After(bestTime) {
			bestTime = info.ModTime()
			best = n.Path
		}
	}
	return best, nil
}
