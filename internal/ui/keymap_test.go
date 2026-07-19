package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/theme"
)

func newTestApp(initial string) *App {
	return NewApp(nil, doc.NewFromString(initial), theme.DefaultDark())
}

// RC2: bindings are the single source of truth for dispatch + help + hints.
func TestBindings_SingleSourceOfTruth(t *testing.T) {
	for i, b := range bindings {
		if b.hint == "" || b.desc == "" {
			t.Errorf("binding %d has empty hint/desc: %+v", i, b)
		}
	}
	// Every bar hint must correspond to a real binding entry.
	hints := barHints()
	for _, b := range bindings {
		if b.barLabel == "" {
			continue
		}
		if !strings.Contains(hints, b.hint) || !strings.Contains(hints, b.barLabel) {
			t.Errorf("bar hints %q missing %q/%q", hints, b.hint, b.barLabel)
		}
	}
}

// BG8: ^C must be a live binding routed to quit (regression: it used to be dead).
func TestBindings_CtrlCRoutesToQuit(t *testing.T) {
	var found *keyBinding
	for i := range bindings {
		for _, ty := range bindings[i].types {
			if ty == tea.KeyCtrlC {
				found = &bindings[i]
			}
		}
	}
	if found == nil || found.run == nil {
		t.Fatal("no dispatchable binding handles ^C")
	}
	hasQ := false
	for _, ty := range found.types {
		if ty == tea.KeyCtrlQ {
			hasQ = true
		}
	}
	if !hasQ {
		t.Error("^C and ^Q should share the quit binding")
	}
}

// BG8: ^C on a clean doc quits.
func TestHandleEditKey_CtrlCCleanQuits(t *testing.T) {
	a := newTestApp("hello")
	_, cmd := a.handleEditKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected a quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", cmd())
	}
}

// BG8: ^C on a dirty doc opens the unsaved-changes prompt instead of quitting.
func TestHandleEditKey_CtrlCDirtyPrompts(t *testing.T) {
	a := newTestApp("")
	a.editor.InsertText("dirty")
	if !a.editor.Doc().Dirty() {
		t.Fatal("precondition: doc should be dirty")
	}
	_, cmd := a.handleEditKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd != nil {
		t.Fatal("dirty ^C should not quit immediately")
	}
	if a.mode != modePrompt || a.prompt != promptQuitDirty {
		t.Fatalf("expected quit-dirty prompt, got mode=%d prompt=%d", a.mode, a.prompt)
	}
}

// P2: Escape clears a lingering status flash.
func TestHandleEditKey_EscClearsFlash(t *testing.T) {
	a := newTestApp("x")
	a.statusErr = "boom"
	a.handleEditKey(tea.KeyMsg{Type: tea.KeyEscape})
	if a.statusErr != "" {
		t.Fatalf("Esc should clear statusErr, got %q", a.statusErr)
	}

	a.statusMsg = "✓ saved"
	a.handleEditKey(tea.KeyMsg{Type: tea.KeyEscape})
	if a.statusMsg != "" {
		t.Fatalf("Esc should clear statusMsg, got %q", a.statusMsg)
	}
}

// P5: word count counts whitespace-separated tokens across all lines.
func TestWordCount(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"one two three", 3},
		{"line one\nline two\n", 4},
	}
	for _, c := range cases {
		if got := wordCount(doc.NewFromString(c.in)); got != c.want {
			t.Errorf("wordCount(%q)=%d want %d", c.in, got, c.want)
		}
	}
}

// P3: an empty vault shows a hint in the finder overlay.
func TestFinder_EmptyVaultHint(t *testing.T) {
	a := newTestApp("")
	a.width, a.height = 80, 24
	a.layoutEditor()
	a.openFinder()
	if got := len(a.finder.Items()); got != 0 {
		t.Fatalf("precondition: expected empty finder, got %d items", got)
	}
	if !strings.Contains(a.viewFinder(), "No notes yet") {
		t.Error("empty finder should show a hint")
	}
}
