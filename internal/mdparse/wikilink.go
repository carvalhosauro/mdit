package mdparse

import (
	"regexp"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// WikiLink is an inline AST node representing an Obsidian-style wikilink,
// either [[target]] or [[target|alias]].
type WikiLink struct {
	ast.BaseInline
	Target string
	Alias  string
}

// KindWikiLink is the NodeKind for WikiLink nodes.
var KindWikiLink = ast.NewNodeKind("WikiLink")

// Kind implements ast.Node.
func (n *WikiLink) Kind() ast.NodeKind { return KindWikiLink }

// Dump implements ast.Node for debugging.
func (n *WikiLink) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Target": n.Target,
		"Alias":  n.Alias,
	}, nil)
}

// Label returns the text that should be displayed for the link: the alias when
// present, otherwise the target.
func (n *WikiLink) Label() string {
	if n.Alias != "" {
		return n.Alias
	}
	return n.Target
}

// wikiLinkParser is a goldmark inline parser for [[...]] spans. It is registered
// with a higher priority than the standard link parser so that "[[x]]" is never
// consumed as a regular markdown link.
type wikiLinkParser struct{}

// Trigger implements parser.InlineParser. The parser fires on '['.
func (p *wikiLinkParser) Trigger() []byte { return []byte{'['} }

// Parse implements parser.InlineParser. It only succeeds on a well-formed
// "[[target]]" or "[[target|alias]]" span on the current line; otherwise it
// returns nil so other parsers get a chance.
func (p *wikiLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	// Need at least "[[x]]".
	if len(line) < 5 || line[0] != '[' || line[1] != '[' {
		return nil
	}
	// Find the closing "]]" without allowing nested brackets in between.
	close := -1
	for i := 2; i+1 < len(line); i++ {
		c := line[i]
		if c == ']' && line[i+1] == ']' {
			close = i
			break
		}
		if c == '[' || c == ']' {
			return nil
		}
	}
	if close < 0 {
		return nil
	}
	content := line[2:close]
	if len(content) == 0 {
		return nil
	}
	target := content
	var alias []byte
	if pipe := indexByte(content, '|'); pipe >= 0 {
		target = content[:pipe]
		alias = content[pipe+1:]
	}
	if len(target) == 0 {
		return nil
	}
	block.Advance(close + 2)
	return &WikiLink{
		Target: string(target),
		Alias:  string(alias),
	}
}

func indexByte(b []byte, c byte) int {
	for i := 0; i < len(b); i++ {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// wikiLinkExtender registers the wikilink inline parser.
type wikiLinkExtender struct{}

// WikiLinkExt is the goldmark extender that installs the wikilink inline parser.
// It runs at priority 199, immediately ahead of the standard link parser (200).
var WikiLinkExt goldmark.Extender = &wikiLinkExtender{}

// Extend implements goldmark.Extender.
func (e *wikiLinkExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(&wikiLinkParser{}, 199),
		),
	)
}

// wikiLinkRe matches a [[target]] or [[target|alias]] span. Targets and aliases
// may not contain brackets; the target may not contain a pipe.
var wikiLinkRe = regexp.MustCompile(`\[\[([^\[\]|]+)(?:\|[^\[\]]*)?\]\]`)

// WikiLinkAt reports the wikilink target if the rune column col falls within a
// [[...]] span (inclusive of the surrounding brackets) on the given raw line.
// col is measured in runes. It is regex-based and independent of the AST, so it
// works directly on the raw text under the editor cursor.
func WikiLinkAt(line string, col int) (target string, ok bool) {
	for _, m := range wikiLinkRe.FindAllStringSubmatchIndex(line, -1) {
		// m[0]/m[1] are byte offsets of the whole match; convert to rune cols.
		startRune := utf8.RuneCountInString(line[:m[0]])
		endRune := utf8.RuneCountInString(line[:m[1]])
		// Span is inclusive of both brackets: [startRune, endRune-1].
		if col >= startRune && col <= endRune-1 {
			return line[m[2]:m[3]], true
		}
	}
	return "", false
}
