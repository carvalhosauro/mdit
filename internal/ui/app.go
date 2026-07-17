// Package ui is the root Bubble Tea application for mdit: it embeds the
// editor widget, draws the status bar, and owns save/quit confirmation prompts.
package ui

import (
	"errors"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/editor"
	"github.com/carvalhosauro/mdit/internal/theme"
	"github.com/carvalhosauro/mdit/internal/vault"
)

type mode int

const (
	modeEdit mode = iota
	modePrompt
)

type promptKind int

const (
	promptNone promptKind = iota
	promptQuitDirty
	promptSaveConflict
)

// App is the root tea.Model. Construct with NewApp.
type App struct {
	vault  *vault.Vault
	theme  theme.Theme
	editor editor.Model

	mode   mode
	prompt promptKind
	// afterPrompt runs after a successful save that was started for some
	// larger intent (e.g. tea.Quit after quit→save, or open-note in Task 8).
	// Cleared on cancel/reload.
	afterPrompt tea.Cmd

	width, height int
	statusErr     string
}

// NewApp builds an App over the given vault and initial document.
func NewApp(v *vault.Vault, initial *doc.Document, th theme.Theme) *App {
	isBroken := func(target string) bool {
		if v == nil {
			return false
		}
		_, ok := v.Resolve(target)
		return !ok
	}
	ed := editor.New(initial, th, isBroken)
	return &App{
		vault:  v,
		theme:  th,
		editor: ed,
		mode:   modeEdit,
	}
}

// Doc returns the document currently open in the editor.
func (a *App) Doc() *doc.Document {
	return a.editor.Doc()
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.layoutEditor()
		return a, nil
	case tea.KeyMsg:
		if a.mode == modePrompt {
			return a.handlePromptKey(msg)
		}
		return a.handleEditKey(msg)
	case editor.FollowLinkMsg, editor.AutocompleteMsg:
		// Handled in Tasks 8/9; ignore for now so typing [[ does not break.
		return a, nil
	}

	var cmd tea.Cmd
	a.editor, cmd = a.editor.Update(msg)
	return a, cmd
}

func (a *App) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlS:
		return a.doSave()
	case tea.KeyCtrlQ:
		return a.doQuit()
	}

	var cmd tea.Cmd
	a.editor, cmd = a.editor.Update(msg)
	return a, cmd
}

func (a *App) doSave() (tea.Model, tea.Cmd) {
	a.statusErr = ""
	a.afterPrompt = nil
	err := a.editor.Doc().Save()
	if err == nil {
		return a, nil
	}
	if errors.Is(err, doc.ErrExternalChange) {
		a.enterPrompt(promptSaveConflict, nil)
		return a, nil
	}
	a.statusErr = err.Error()
	return a, nil
}

func (a *App) doQuit() (tea.Model, tea.Cmd) {
	if a.editor.Doc().Dirty() {
		a.enterPrompt(promptQuitDirty, tea.Quit)
		return a, nil
	}
	return a, tea.Quit
}

func (a *App) enterPrompt(kind promptKind, after tea.Cmd) {
	a.mode = modePrompt
	a.prompt = kind
	a.afterPrompt = after
}

func (a *App) clearPrompt() {
	a.mode = modeEdit
	a.prompt = promptNone
	a.afterPrompt = nil
}

// finishPrompt leaves prompt mode and runs afterPrompt if set (e.g. quit
// after a successful save-from-quit flow).
func (a *App) finishPrompt() (tea.Model, tea.Cmd) {
	cmd := a.afterPrompt
	a.mode = modeEdit
	a.prompt = promptNone
	a.afterPrompt = nil
	return a, cmd
}

func (a *App) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ch := promptChar(msg)
	switch a.prompt {
	case promptQuitDirty:
		switch ch {
		case 's':
			err := a.editor.Doc().Save()
			if err != nil {
				if errors.Is(err, doc.ErrExternalChange) {
					// Keep afterPrompt (tea.Quit) so overwrite can still quit.
					a.prompt = promptSaveConflict
					return a, nil
				}
				a.statusErr = err.Error()
				a.clearPrompt()
				return a, nil
			}
			return a.finishPrompt()
		case 'd':
			a.afterPrompt = nil
			return a, tea.Quit
		case 'c':
			a.clearPrompt()
			return a, nil
		}
	case promptSaveConflict:
		switch ch {
		case 'o':
			if err := a.editor.Doc().SaveForce(); err != nil {
				a.statusErr = err.Error()
				a.clearPrompt()
				return a, nil
			}
			return a.finishPrompt()
		case 'r':
			path := a.editor.Doc().Path()
			d, err := doc.Load(path)
			if err != nil {
				a.statusErr = err.Error()
			} else {
				a.editor.SetDoc(d)
				a.layoutEditor()
			}
			// Reload abandons the pending intent (quit / open-other).
			a.clearPrompt()
			return a, nil
		case 'c':
			a.clearPrompt()
			return a, nil
		}
	}
	return a, nil
}

func promptChar(msg tea.KeyMsg) rune {
	switch msg.Type {
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			r := msg.Runes[0]
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
			return r
		}
	}
	return 0
}

func (a *App) layoutEditor() {
	h := a.height - 1
	if h < 1 {
		h = 1
	}
	w := a.width
	if w < 1 {
		w = 1
	}
	a.editor.SetSize(w, h)
}

// View implements tea.Model.
func (a *App) View() string {
	ed := a.editor.View()
	bar := a.renderBottom()
	if ed == "" {
		return bar
	}
	return ed + "\n" + bar
}

func (a *App) renderBottom() string {
	if a.mode == modePrompt {
		return a.renderPrompt()
	}
	return a.renderStatusBar()
}

func (a *App) fileName() string {
	p := a.editor.Doc().Path()
	if p == "" {
		return "untitled.md"
	}
	return filepath.Base(p)
}
