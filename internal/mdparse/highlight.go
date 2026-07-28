package mdparse

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Highlight is an Obsidian-style ==marked== inline span.
type Highlight struct {
	ast.BaseInline
}

// KindHighlight is the NodeKind for Highlight nodes.
var KindHighlight = ast.NewNodeKind("Highlight")

// Kind implements ast.Node.
func (n *Highlight) Kind() ast.NodeKind { return KindHighlight }

// Dump implements ast.Node for debugging.
func (n *Highlight) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type highlightParser struct{}

func (p *highlightParser) Trigger() []byte { return []byte{'='} }

func (p *highlightParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	// Need at least "==x==".
	if len(line) < 5 || line[0] != '=' || line[1] != '=' {
		return nil
	}
	closeIdx := -1
	for i := 2; i+1 < len(line); i++ {
		if line[i] == '=' && line[i+1] == '=' {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 || closeIdx == 2 {
		return nil // empty == ==
	}
	inner := line[2:closeIdx]
	block.Advance(closeIdx + 2)

	n := &Highlight{}
	n.AppendChild(n, ast.NewString(inner))
	return n
}

type highlightExtender struct{}

// HighlightExt installs the ==highlight== inline parser.
var HighlightExt goldmark.Extender = &highlightExtender{}

func (e *highlightExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(&highlightParser{}, 150),
		),
	)
}
