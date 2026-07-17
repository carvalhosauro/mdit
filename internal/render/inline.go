package render

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"

	"github.com/carvalhosauro/mdit/internal/mdparse"
)

// cellWidth reports the printable width of s in terminal cells, ignoring any
// ANSI escape sequences. It is the single measurement used everywhere the
// width invariant is enforced.
func cellWidth(s string) int {
	return runewidth.StringWidth(ansi.Strip(s))
}

// renderInlines walks the inline children of a block node and returns a single
// styled string (ANSI escapes included). Text nodes are emitted unstyled so that
// enclosing emphasis/strike spans wrap cleanly around them.
func renderInlines(n ast.Node, source []byte, ctx Context) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		sb.WriteString(styleInline(c, source, ctx))
	}
	return sb.String()
}

// styleInline renders a single inline node to a styled string.
func styleInline(c ast.Node, source []byte, ctx Context) string {
	th := ctx.Theme
	switch t := c.(type) {
	case *ast.Text:
		s := string(t.Segment.Value(source))
		if t.SoftLineBreak() || t.HardLineBreak() {
			s += " "
		}
		return s
	case *ast.String:
		return string(t.Value)
	case *ast.Emphasis:
		inner := renderInlines(t, source, ctx)
		if t.Level >= 2 {
			return th.Bold.Render(inner)
		}
		return th.Italic.Render(inner)
	case *ast.CodeSpan:
		return th.CodeSpan.Render(plainInline(t, source))
	case *ast.Link:
		return th.Link.Render(plainInline(t, source))
	case *ast.AutoLink:
		return th.Link.Render(string(t.Label(source)))
	case *east.Strikethrough:
		return th.Strike.Render(renderInlines(t, source, ctx))
	case *mdparse.WikiLink:
		if ctx.IsBroken != nil && ctx.IsBroken(t.Target) {
			return th.BrokenLink.Render(t.Label())
		}
		return th.WikiLink.Render(t.Label())
	case *east.TaskCheckBox:
		return "" // rendered as a list marker, not inline text
	case *ast.RawHTML:
		var b strings.Builder
		for i := 0; i < t.Segments.Len(); i++ {
			seg := t.Segments.At(i)
			b.Write(seg.Value(source))
		}
		return b.String()
	default:
		return renderInlines(c, source, ctx)
	}
}

// plainInline returns the unstyled text of an inline node's subtree, resolving
// ast.Text segments out of source. Soft/hard line breaks become spaces.
func plainInline(n ast.Node, source []byte) string {
	var sb strings.Builder
	collectText(n, source, &sb)
	return sb.String()
}

func collectText(n ast.Node, source []byte, sb *strings.Builder) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *ast.Text:
			sb.Write(t.Segment.Value(source))
			if t.SoftLineBreak() || t.HardLineBreak() {
				sb.WriteByte(' ')
			}
		case *ast.String:
			sb.Write(t.Value)
		case *ast.AutoLink:
			sb.Write(t.Label(source))
		case *mdparse.WikiLink:
			sb.WriteString(t.Label())
		case *east.TaskCheckBox:
			// skip
		default:
			collectText(c, source, sb)
		}
	}
}

// wrapStyled word-wraps an ANSI-styled string to width cells, falling back to a
// hard (mid-word) wrap for any line that still overflows (e.g. a single word
// longer than width). It never returns an empty slice.
func wrapStyled(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	var out []string
	for _, ln := range strings.Split(wordwrap.String(s, width), "\n") {
		if cellWidth(ln) <= width {
			out = append(out, ln)
			continue
		}
		out = append(out, strings.Split(wrap.String(ln, width), "\n")...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}
