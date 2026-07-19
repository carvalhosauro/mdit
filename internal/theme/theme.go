// Package theme defines the visual styles used by the renderer. A Theme is a
// flat bag of lipgloss.Style values, one per markdown construct, so the renderer
// never hard-codes colors. Default returns an adaptive palette (Catppuccin
// Latte on light terminals, Mocha on dark); DefaultDark returns the fixed dark
// palette used by tests. The package imports only lipgloss.
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

	// Status bar segments: file name, dirty marker, key hints, transient
	// success flash. All share the StatusBar background.
	StatusBarName, StatusBarDirty lipgloss.Style
	StatusBarHint, StatusBarOK    lipgloss.Style

	// PromptBar is the destructive/confirmation bar (high-contrast alert
	// background); PromptKey highlights the answer keys inside it.
	PromptBar, PromptKey lipgloss.Style

	// Menu styles cover popup overlays (autocomplete, help): body text,
	// selected row, border, and title.
	Menu, MenuSelected, MenuBorder, MenuTitle lipgloss.Style

	// Dim is applied to background content while an overlay (finder, help)
	// is focused.
	Dim lipgloss.Style

	// ChromaStyle is the name of the chroma highlighting style used for fenced
	// code blocks. It is not a lipgloss.Style; it selects a chroma palette.
	ChromaStyle string
}

// palette abstracts the color set a Theme is built from, so the same style
// wiring serves the fixed dark palette and the adaptive light/dark one.
type palette struct {
	text, subtext, overlay   lipgloss.TerminalColor
	surface, surface2        lipgloss.TerminalColor
	pink, mauve, red, peach  lipgloss.TerminalColor
	yellow, green, sky, blue lipgloss.TerminalColor
	base                     lipgloss.TerminalColor // terminal-ish background, used as fg on alert bars
	chroma                   string
}

// Catppuccin Mocha; dark background.
var mocha = palette{
	text:     lipgloss.Color("#CDD6F4"),
	subtext:  lipgloss.Color("#A6ADC8"),
	overlay:  lipgloss.Color("#6C7086"),
	surface:  lipgloss.Color("#313244"),
	surface2: lipgloss.Color("#585B70"),
	pink:     lipgloss.Color("#F5C2E7"),
	mauve:    lipgloss.Color("#CBA6F7"),
	red:      lipgloss.Color("#F38BA8"),
	peach:    lipgloss.Color("#FAB387"),
	yellow:   lipgloss.Color("#F9E2AF"),
	green:    lipgloss.Color("#A6E3A1"),
	sky:      lipgloss.Color("#89DCEB"),
	blue:     lipgloss.Color("#89B4FA"),
	base:     lipgloss.Color("#1E1E2E"),
	chroma:   "catppuccin-mocha",
}

// adaptive pairs Catppuccin Latte (light terminals) with Mocha (dark).
var adaptive = palette{
	text:     lipgloss.AdaptiveColor{Light: "#4C4F69", Dark: "#CDD6F4"},
	subtext:  lipgloss.AdaptiveColor{Light: "#6C6F85", Dark: "#A6ADC8"},
	overlay:  lipgloss.AdaptiveColor{Light: "#9CA0B0", Dark: "#6C7086"},
	surface:  lipgloss.AdaptiveColor{Light: "#CCD0DA", Dark: "#313244"},
	surface2: lipgloss.AdaptiveColor{Light: "#ACB0BE", Dark: "#585B70"},
	pink:     lipgloss.AdaptiveColor{Light: "#EA76CB", Dark: "#F5C2E7"},
	mauve:    lipgloss.AdaptiveColor{Light: "#8839EF", Dark: "#CBA6F7"},
	red:      lipgloss.AdaptiveColor{Light: "#D20F39", Dark: "#F38BA8"},
	peach:    lipgloss.AdaptiveColor{Light: "#FE640B", Dark: "#FAB387"},
	yellow:   lipgloss.AdaptiveColor{Light: "#DF8E1D", Dark: "#F9E2AF"},
	green:    lipgloss.AdaptiveColor{Light: "#40A02B", Dark: "#A6E3A1"},
	sky:      lipgloss.AdaptiveColor{Light: "#04A5E5", Dark: "#89DCEB"},
	blue:     lipgloss.AdaptiveColor{Light: "#1E66F5", Dark: "#89B4FA"},
	base:     lipgloss.AdaptiveColor{Light: "#EFF1F5", Dark: "#1E1E2E"},
	chroma:   "", // chosen at construction from the terminal background
}

// Default returns the adaptive theme: Latte colors on a light terminal
// background, Mocha on a dark one (including the chroma code-block palette).
func Default() Theme {
	p := adaptive
	if lipgloss.HasDarkBackground() {
		p.chroma = "catppuccin-mocha"
	} else {
		p.chroma = "catppuccin-latte"
	}
	return fromPalette(p)
}

// DefaultDark returns the built-in fixed dark theme.
func DefaultDark() Theme {
	return fromPalette(mocha)
}

func fromPalette(p palette) Theme {
	base := lipgloss.NewStyle()
	bar := base.Foreground(p.text).Background(p.surface)
	return Theme{
		H1: base.Bold(true).Foreground(p.pink),
		H2: base.Bold(true).Foreground(p.mauve),
		H3: base.Bold(true).Foreground(p.blue),
		H4: base.Bold(true).Foreground(p.sky),
		H5: base.Bold(true).Foreground(p.green),
		H6: base.Bold(true).Foreground(p.yellow),

		Bold:     base.Bold(true),
		Italic:   base.Italic(true),
		CodeSpan: base.Foreground(p.peach).Background(p.surface),
		Strike:   base.Strikethrough(true).Foreground(p.overlay),

		CodeBlock:       base.Foreground(p.text),
		CodeBlockBorder: base.Foreground(p.surface2),

		Link: base.Underline(true).Foreground(p.blue),
		// WikiLinks are bold sky (no underline) so they read as vault-internal,
		// distinct from blue-underlined markdown [text](url) links.
		WikiLink:   base.Bold(true).Foreground(p.sky),
		BrokenLink: base.Underline(true).Foreground(p.red),

		Quote:    base.Italic(true).Foreground(p.subtext),
		Bullet:   base.Foreground(p.blue),
		TaskDone: base.Foreground(p.green),
		TaskOpen: base.Foreground(p.yellow),

		Table:       base.Foreground(p.text),
		TableHeader: base.Bold(true).Foreground(p.yellow),

		Frontmatter:   base.Foreground(p.overlay),
		ThematicBreak: base.Foreground(p.surface2),

		Text:     base.Foreground(p.text),
		RawBlock: base.Background(p.surface),

		StatusBar:      bar,
		StatusBarError: base.Foreground(p.red).Background(p.surface),

		StatusBarName:  bar.Bold(true),
		StatusBarDirty: base.Foreground(p.peach).Background(p.surface),
		StatusBarHint:  base.Foreground(p.overlay).Background(p.surface),
		StatusBarOK:    base.Foreground(p.green).Background(p.surface),

		PromptBar: base.Foreground(p.base).Background(p.peach),
		PromptKey: base.Foreground(p.base).Background(p.peach).Bold(true).Underline(true),

		Menu:         base.Foreground(p.text),
		MenuSelected: base.Bold(true).Foreground(p.blue),
		MenuBorder:   base.Foreground(p.surface2),
		MenuTitle:    base.Bold(true).Foreground(p.mauve),

		Dim: base.Foreground(p.overlay),

		ChromaStyle: p.chroma,
	}
}
