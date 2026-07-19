package ui

import tea "github.com/charmbracelet/bubbletea"

// keyBinding is one app-level binding. It is the single source of truth for the
// edit-mode dispatch (handleEditKey), the help overlay, and the status-bar
// hints, so those three can never drift apart (the drift that once left ^C
// dead).
//
// Bindings with a nil run are handled by the editor widget, not the App (e.g.
// follow-link, undo/redo, autocomplete); they appear in help but are forwarded
// to the editor by handleEditKey's fallthrough.
type keyBinding struct {
	types    []tea.KeyType // keys that trigger this binding (empty = typed, editor-handled)
	hint     string        // display label in help, e.g. "^S"
	desc     string        // help description
	barLabel string        // short label for the status bar; non-empty ⇒ shown there
	run      func(*App) (tea.Model, tea.Cmd)
}

// bindings is the ordered list of every user-facing binding. Order is the help
// overlay's row order.
var bindings = []keyBinding{
	{types: []tea.KeyType{tea.KeyCtrlS}, hint: "^S", desc: "save", barLabel: "save", run: (*App).doSave},
	{types: []tea.KeyType{tea.KeyCtrlP}, hint: "^P", desc: "find note", barLabel: "notes", run: (*App).cmdOpenFinder},
	{types: []tea.KeyType{tea.KeyCtrlE}, hint: "^E", desc: "zen mode", barLabel: "zen", run: (*App).enterZen},
	{types: []tea.KeyType{tea.KeyCtrlB}, hint: "^B", desc: "back to previous note", run: (*App).goBack},
	{types: []tea.KeyType{tea.KeyCtrlCloseBracket}, hint: "^]", desc: "follow wikilink under cursor"},
	{hint: "[[", desc: "wikilink autocomplete"},
	{hint: "^Z / ^Y", desc: "undo / redo"},
	{types: []tea.KeyType{tea.KeyCtrlG}, hint: "^G", desc: "this help", barLabel: "help", run: (*App).cmdHelp},
	{types: []tea.KeyType{tea.KeyCtrlC, tea.KeyCtrlQ}, hint: "^C / ^Q", desc: "quit", run: (*App).doQuit},
}

// barHints builds the status-bar hint string from the bindings that opt into it
// (barLabel set), so the bar can't advertise a key the dispatch doesn't honor.
func barHints() string {
	s := ""
	for _, b := range bindings {
		if b.barLabel == "" {
			continue
		}
		if s != "" {
			s += "  "
		}
		s += b.hint + " " + b.barLabel
	}
	return s
}

// cmdOpenFinder / cmdHelp adapt the mode-setting helpers to the run signature.
func (a *App) cmdOpenFinder() (tea.Model, tea.Cmd) {
	a.openFinder()
	return a, nil
}

func (a *App) cmdHelp() (tea.Model, tea.Cmd) {
	a.mode = modeHelp
	return a, nil
}
