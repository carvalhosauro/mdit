package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

const (
	promptQuitText     = "Unsaved changes: [s]ave / [d]iscard / [c]ancel"
	promptConflictText = "File changed on disk: [o]verwrite / [r]eload / [c]ancel"
)

// renderPrompt draws the confirmation bar in the alert style (high-contrast
// background) so the mode change is unmistakable, highlighting the answer keys.
func (a *App) renderPrompt() string {
	msg := ""
	switch a.prompt {
	case promptQuitDirty:
		msg = promptQuitText
	case promptSaveConflict:
		msg = promptConflictText
	case promptCreateNote:
		msg = fmt.Sprintf("Create note %q? [c]reate / [Esc] cancel", a.pendingTarget)
	}
	w := a.width
	if w < 1 {
		w = 1
	}
	msg = truncateCells(msg, w)

	// Style the letter inside each [x] with PromptKey; everything else with
	// PromptBar. Rendered per-segment so the bar reads as one surface.
	var b strings.Builder
	rest := msg
	for {
		i := strings.Index(rest, "[")
		j := strings.Index(rest, "]")
		if i < 0 || j < i+2 {
			b.WriteString(a.theme.PromptBar.Render(rest))
			break
		}
		b.WriteString(a.theme.PromptBar.Render(rest[:i+1]))
		b.WriteString(a.theme.PromptKey.Render(rest[i+1 : j]))
		rest = rest[j:]
	}
	if pad := w - runewidth.StringWidth(msg); pad > 0 {
		b.WriteString(a.theme.PromptBar.Render(strings.Repeat(" ", pad)))
	}
	return b.String()
}
