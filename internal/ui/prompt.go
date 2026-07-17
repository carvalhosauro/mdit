package ui

const (
	promptQuitText     = "Unsaved changes: [s]ave / [d]iscard / [c]ancel"
	promptConflictText = "File changed on disk: [o]verwrite / [r]eload / [c]ancel"
)

func (a *App) renderPrompt() string {
	msg := ""
	switch a.prompt {
	case promptQuitDirty:
		msg = promptQuitText
	case promptSaveConflict:
		msg = promptConflictText
	}
	w := a.width
	if w < 1 {
		w = 1
	}
	return a.theme.StatusBar.Width(w).Render(truncateCells(msg, w))
}
