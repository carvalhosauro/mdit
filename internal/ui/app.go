// Package ui is the root Bubble Tea application for mdit: it embeds the
// editor widget, draws the status bar, and owns save/quit confirmation prompts,
// the fuzzy finder, wikilink navigation, autocomplete, and zen mode.
package ui

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/editor"
	"github.com/carvalhosauro/mdit/internal/theme"
	"github.com/carvalhosauro/mdit/internal/vault"
)

type mode int

const (
	modeEdit mode = iota
	modePrompt
	modeFinder
	modeAutocomplete
	modeZen
)

type promptKind int

const (
	promptNone promptKind = iota
	promptQuitDirty
	promptSaveConflict
)

// openNoteMsg requests opening a note path (after dirty prompts resolve).
type openNoteMsg struct{ path string }

// App is the root tea.Model. Construct with NewApp.
type App struct {
	vault  *vault.Vault
	theme  theme.Theme
	editor editor.Model

	mode   mode
	prompt promptKind
	// afterPrompt runs after a successful save that was started for some
	// larger intent (e.g. tea.Quit after quit→save, or open-note).
	afterPrompt tea.Cmd

	width, height int
	statusErr     string

	history []string // navigation stack (paths); pushed on finder/follow open

	finder list.Model

	acItems  []string // autocomplete candidate names
	acIndex  int
	acActive bool

	// zenSavedScroll is restored when leaving zen if needed.
	zenSavedScroll int
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
	a := &App{
		vault:  v,
		theme:  th,
		editor: ed,
		mode:   modeEdit,
		finder: newFinderList(nil, 40, 12),
	}
	a.refreshFinderItems()
	return a
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
		a.resizeFinder()
		return a, nil

	case openNoteMsg:
		return a.doOpenNote(msg.path, true)

	case editor.FollowLinkMsg:
		return a.handleFollowLink(msg.Target)

	case editor.AutocompleteMsg:
		a.openAutocomplete()
		return a, nil

	case tea.KeyMsg:
		switch a.mode {
		case modePrompt:
			return a.handlePromptKey(msg)
		case modeFinder:
			return a.handleFinderKey(msg)
		case modeAutocomplete:
			return a.handleAutocompleteKey(msg)
		case modeZen:
			return a.handleZenKey(msg)
		default:
			return a.handleEditKey(msg)
		}
	}

	// Forward async msgs (e.g. list.FilterMatchesMsg) to the active child.
	if a.mode == modeFinder {
		var cmd tea.Cmd
		a.finder, cmd = a.finder.Update(msg)
		return a, cmd
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
	case tea.KeyCtrlP:
		a.openFinder()
		return a, nil
	case tea.KeyCtrlE:
		return a.enterZen()
	case tea.KeyCtrlB:
		return a.goBack()
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

// finishPrompt leaves prompt mode and runs afterPrompt if set.
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
					a.prompt = promptSaveConflict
					return a, nil
				}
				a.statusErr = err.Error()
				a.clearPrompt()
				return a, nil
			}
			return a.finishPrompt()
		case 'd':
			cmd := a.afterPrompt
			a.clearPrompt()
			if cmd != nil {
				return a, cmd
			}
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
	if a.mode == modeZen {
		a.layoutZenEditor()
		return
	}
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
	switch a.mode {
	case modeFinder:
		return a.viewFinder()
	case modeZen:
		return a.viewZen()
	case modeAutocomplete:
		return a.viewAutocomplete()
	}

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

// requestOpen opens path, prompting if the current doc is dirty.
func (a *App) requestOpen(path string) (tea.Model, tea.Cmd) {
	if a.editor.Doc().Dirty() {
		p := path
		a.enterPrompt(promptQuitDirty, func() tea.Msg { return openNoteMsg{path: p} })
		return a, nil
	}
	return a.doOpenNote(path, true)
}

// doOpenNote loads path into the editor. If pushHistory is true and the current
// path differs, the current path is pushed onto the navigation stack.
func (a *App) doOpenNote(path string, pushHistory bool) (tea.Model, tea.Cmd) {
	cur := a.editor.Doc().Path()
	if pushHistory && cur != "" && cur != path {
		a.history = append(a.history, cur)
	}
	d, err := doc.Load(path)
	if err != nil {
		a.statusErr = err.Error()
		a.mode = modeEdit
		return a, nil
	}
	a.editor.SetDoc(d)
	a.statusErr = ""
	a.mode = modeEdit
	a.prompt = promptNone
	a.afterPrompt = nil
	a.acActive = false
	a.layoutEditor()
	return a, nil
}

func (a *App) goBack() (tea.Model, tea.Cmd) {
	if len(a.history) == 0 {
		return a, nil
	}
	path := a.history[len(a.history)-1]
	a.history = a.history[:len(a.history)-1]
	return a.doOpenNote(path, false)
}

func (a *App) handleFollowLink(target string) (tea.Model, tea.Cmd) {
	if a.vault == nil {
		a.statusErr = "broken link: " + target
		return a, nil
	}
	path, ok := a.vault.Resolve(target)
	if !ok {
		a.statusErr = "broken link: " + target
		return a, nil
	}
	a.statusErr = ""
	return a.requestOpen(path)
}

// centerBlock wraps content in a full-terminal frame, centering horizontally
// and vertically when smaller than the terminal.
func centerBlock(content string, termW, termH int) string {
	lines := strings.Split(content, "\n")
	contentH := len(lines)
	contentW := 0
	for _, ln := range lines {
		if w := lipgloss.Width(ln); w > contentW {
			contentW = w
		}
	}
	padX := (termW - contentW) / 2
	if padX < 0 {
		padX = 0
	}
	padY := (termH - contentH) / 2
	if padY < 0 {
		padY = 0
	}
	var b strings.Builder
	for i := 0; i < padY; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat(" ", termW))
	}
	for i, ln := range lines {
		if padY > 0 || i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat(" ", padX))
		b.WriteString(ln)
		right := termW - padX - lipgloss.Width(ln)
		if right > 0 {
			b.WriteString(strings.Repeat(" ", right))
		}
	}
	for i := padY + contentH; i < termH; i++ {
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(" ", termW))
	}
	return b.String()
}
