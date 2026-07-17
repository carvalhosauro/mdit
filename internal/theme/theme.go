// Package theme defines the visual styles used by the renderer. A Theme is a
// flat bag of lipgloss.Style values, one per markdown construct, so the renderer
// never hard-codes colors. DefaultDark returns the built-in palette tuned for a
// dark terminal background. The package imports only lipgloss.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme holds one lipgloss.Style per renderable markdown construct. All fields
// are value types, so a Theme is cheap to copy into a render.Context.
type Theme struct {
	H1, H2, H3, H4, H5, H6            lipgloss.Style
	Bold, Italic, CodeSpan, Strike    lipgloss.Style
	CodeBlock, CodeBlockBorder        lipgloss.Style
	Link, WikiLink, BrokenLink        lipgloss.Style // BrokenLink: red fg
	Quote, Bullet, TaskDone, TaskOpen lipgloss.Style
	Table, TableHeader                lipgloss.Style
	Frontmatter, ThematicBreak        lipgloss.Style
	Text, RawBlock                    lipgloss.Style // RawBlock: subtle bg on the raw block under the cursor
	StatusBar, StatusBarError         lipgloss.Style

	// ChromaStyle is the name of the chroma highlighting style used for fenced
	// code blocks. It is not a lipgloss.Style; it selects a chroma palette.
	ChromaStyle string
}

// Catppuccin Mocha-ish palette; dark background assumed.
const (
	cText     = "#CDD6F4"
	cSubtext  = "#A6ADC8"
	cOverlay  = "#6C7086"
	cSurface  = "#313244"
	cSurface2 = "#585B70"
	cPink     = "#F5C2E7"
	cMauve    = "#CBA6F7"
	cRed      = "#F38BA8"
	cPeach    = "#FAB387"
	cYellow   = "#F9E2AF"
	cGreen    = "#A6E3A1"
	cSky      = "#89DCEB"
	cBlue     = "#89B4FA"
)

// DefaultDark returns the built-in dark theme.
func DefaultDark() Theme {
	base := lipgloss.NewStyle()
	return Theme{
		H1: base.Bold(true).Foreground(lipgloss.Color(cPink)),
		H2: base.Bold(true).Foreground(lipgloss.Color(cMauve)),
		H3: base.Bold(true).Foreground(lipgloss.Color(cBlue)),
		H4: base.Bold(true).Foreground(lipgloss.Color(cSky)),
		H5: base.Bold(true).Foreground(lipgloss.Color(cGreen)),
		H6: base.Bold(true).Foreground(lipgloss.Color(cYellow)),

		Bold:     base.Bold(true),
		Italic:   base.Italic(true),
		CodeSpan: base.Foreground(lipgloss.Color(cPeach)).Background(lipgloss.Color(cSurface)),
		Strike:   base.Strikethrough(true).Foreground(lipgloss.Color(cOverlay)),

		CodeBlock:       base.Foreground(lipgloss.Color(cText)),
		CodeBlockBorder: base.Foreground(lipgloss.Color(cSurface2)),

		Link:       base.Underline(true).Foreground(lipgloss.Color(cBlue)),
		WikiLink:   base.Underline(true).Foreground(lipgloss.Color(cSky)),
		BrokenLink: base.Underline(true).Foreground(lipgloss.Color(cRed)),

		Quote:    base.Italic(true).Foreground(lipgloss.Color(cSubtext)),
		Bullet:   base.Foreground(lipgloss.Color(cBlue)),
		TaskDone: base.Foreground(lipgloss.Color(cGreen)),
		TaskOpen: base.Foreground(lipgloss.Color(cYellow)),

		Table:       base.Foreground(lipgloss.Color(cText)),
		TableHeader: base.Bold(true).Foreground(lipgloss.Color(cYellow)),

		Frontmatter:   base.Foreground(lipgloss.Color(cOverlay)),
		ThematicBreak: base.Foreground(lipgloss.Color(cSurface2)),

		Text:     base.Foreground(lipgloss.Color(cText)),
		RawBlock: base.Background(lipgloss.Color(cSurface)),

		StatusBar:      base.Foreground(lipgloss.Color(cText)).Background(lipgloss.Color(cSurface)),
		StatusBarError: base.Foreground(lipgloss.Color(cRed)).Background(lipgloss.Color(cSurface)),

		ChromaStyle: "catppuccin-mocha",
	}
}
