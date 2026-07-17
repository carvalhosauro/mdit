package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// overlayAt paints fg on top of bg (both multi-line strings) with fg's top-left
// corner at cell column x, screen row y. Background lines are cut ANSI-safely
// around the foreground box; short background lines are padded with spaces.
// Styling that spans the cut may lose its opening sequence on the right
// fragment; overlays are drawn over dimmed or single-style content, where that
// is invisible.
func overlayAt(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	fgW := 0
	for _, ln := range fgLines {
		if w := lipgloss.Width(ln); w > fgW {
			fgW = w
		}
	}

	for i, ln := range fgLines {
		row := y + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		bgLine := bgLines[row]
		left := ansi.Truncate(bgLine, x, "")
		if pad := x - lipgloss.Width(left); pad > 0 {
			left += strings.Repeat(" ", pad)
		}
		right := ansi.TruncateLeft(bgLine, x+fgW, "")
		if pad := fgW - lipgloss.Width(ln); pad > 0 {
			ln += strings.Repeat(" ", pad)
		}
		bgLines[row] = left + "\x1b[0m" + ln + right
	}
	return strings.Join(bgLines, "\n")
}

// overlayCenter centers box over bg within a termW x termH screen.
func overlayCenter(bg, box string, termW, termH int) string {
	w := lipgloss.Width(box)
	h := lipgloss.Height(box)
	x := (termW - w) / 2
	if x < 0 {
		x = 0
	}
	y := (termH - h) / 2
	if y < 0 {
		y = 0
	}
	return overlayAt(bg, box, x, y)
}

// dimContent strips colors from s and re-renders every line in the theme's Dim
// style, so overlay backgrounds recede without disappearing (dim-to-focus).
func (a *App) dimContent(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = a.theme.Dim.Render(ansi.Strip(ln))
	}
	return strings.Join(lines, "\n")
}
