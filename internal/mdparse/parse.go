// Package mdparse turns a line buffer into a contiguous sequence of line-range
// blocks backed by the goldmark AST, plus an Obsidian-style wikilink inline
// extension. It is the bridge between the raw document model (internal/doc) and
// the renderer/editor: the renderer walks each block's goldmark Node to style
// inline text, and the editor uses the line ranges to decide which block the
// cursor sits in. It imports no TUI library and no other internal packages.
package mdparse

import (
	"regexp"
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Kind classifies a block for rendering and editing decisions.
type Kind int

const (
	Blank Kind = iota
	Heading
	Paragraph
	CodeFence
	List
	Table
	Blockquote
	ThematicBreak
	Frontmatter
	Other // HTML block and any other unmapped content.
	// IndentedCode is a 4-space (or tab) indented code block. It is kept distinct
	// from CodeFence because it has no ``` delimiter lines and no info string.
	// Appended last so existing Kind values (Blank..Other) keep their integers.
	IndentedCode
)

// Block is a contiguous, inclusive range of raw line indices [Start, End] with
// its classified Kind and (for goldmark-derived blocks) the backing AST node.
// Node is nil for Blank and Frontmatter blocks.
type Block struct {
	Kind       Kind
	Start, End int
	Node       ast.Node
}

// Result is the output of Parse: the ordered blocks plus the exact bytes that
// were parsed (all lines joined with "\n"). Source is what the renderer must
// use to resolve AST text segments.
type Result struct {
	Blocks []Block
	Source []byte
}

// thematicBreakRe matches a CommonMark thematic break line: three or more -, *,
// or _ (optionally spaced), with up to three leading spaces.
var thematicBreakRe = regexp.MustCompile(`^ {0,3}(?:(?:-[ \t]*){3,}|(?:\*[ \t]*){3,}|(?:_[ \t]*){3,})$`)

// fenceRe matches a fenced-code delimiter line (``` or ~~~, 3+), possibly with
// an info string on the opening fence.
var fenceRe = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")

// setextUnderlineRe matches a setext heading underline line: a run of '=' or two
// or more '-', optionally indented up to three spaces and trailing whitespace.
// goldmark folds the underline into the setext Heading node but exposes no line
// segment for it, so it is reattached to the preceding Heading block manually.
var setextUnderlineRe = regexp.MustCompile(`^ {0,3}(=+|-{2,})\s*$`)

var (
	mdOnce sync.Once
	md     goldmark.Markdown
)

// Markdown returns the shared, configured goldmark instance used by Parse. The
// renderer can reuse it so inline parsing (including wikilinks) matches exactly.
func Markdown() goldmark.Markdown {
	mdOnce.Do(func() {
		md = goldmark.New(goldmark.WithExtensions(extension.GFM, WikiLinkExt))
	})
	return md
}

// Parse segments lines into contiguous blocks. The returned Blocks always cover
// every line index in [0, len(lines)-1] with no gaps and no overlap — the
// invariant the editor's virtualization depends on.
//
// goldmark parses the FULL source, so every Block.Node's inline segments align
// to Result.Source by construction (the render consumer slices those segments
// straight out of Source). Frontmatter is detected separately (line 0 "---" with
// a later "---" closing it) and the blocks goldmark produced within that line
// range are collapsed into a single Frontmatter block (Node nil).
func Parse(lines []string) Result {
	source := []byte(strings.Join(lines, "\n"))
	n := len(lines)
	if n == 0 {
		return Result{Source: source}
	}

	blocks := segmentLines(lines)

	// Frontmatter: line 0 == "---" with a later "---" closing it. goldmark parses
	// the frontmatter interior as ordinary markdown (thematic breaks, setext
	// headings, etc.); those blocks are fully absorbed into one Frontmatter block.
	if lines[0] == "---" {
		for i := 1; i < n; i++ {
			if lines[i] == "---" {
				blocks = absorbFrontmatter(blocks, i)
				break
			}
		}
	}

	return Result{Blocks: blocks, Source: source}
}

// absorbFrontmatter replaces every block within the frontmatter line range
// [0, k] with a single Frontmatter block (Node nil), preserving the coverage
// invariant. A block that crosses the boundary (starts <= k but ends > k) has
// its Start clamped to k+1 and is kept.
func absorbFrontmatter(blocks []Block, k int) []Block {
	out := []Block{{Kind: Frontmatter, Start: 0, End: k}}
	for _, b := range blocks {
		if b.End <= k {
			continue // fully inside the frontmatter range
		}
		if b.Start <= k {
			b.Start = k + 1 // crosses the boundary; clamp to just after frontmatter
		}
		out = append(out, b)
	}
	return out
}

// segmentLines segments the given lines into blocks whose ranges are relative to
// lines[0]. It runs goldmark over the joined lines so every node segment offset
// aligns with the joined bytes.
func segmentLines(remLines []string) []Block {
	n := len(remLines)

	// Byte offset of the start of each line within the joined source.
	offsets := make([]int, n+1)
	for i, l := range remLines {
		offsets[i+1] = offsets[i] + len(l) + 1
	}
	rem := strings.Join(remLines, "\n")

	root := Markdown().Parser().Parse(text.NewReader([]byte(rem)))

	kindOf := make([]Kind, n)
	nodeOf := make([]ast.Node, n)
	claimed := make([]bool, n)
	claim := func(i int, k Kind, node ast.Node) {
		if i < 0 || i >= n || claimed[i] {
			return
		}
		claimed[i] = true
		kindOf[i] = k
		nodeOf[i] = node
	}

	type item struct {
		node       ast.Node
		kind       Kind
		start, end int
		hasCore    bool
	}
	var items []item
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		s, e, ok := coreRange(c, offsets)
		items = append(items, item{node: c, kind: mapKind(c), start: s, end: e, hasCore: ok})
	}

	// Pass A: blocks with a known content range. Fenced code blocks are grown to
	// include their delimiter lines, which goldmark excludes from Lines().
	for _, it := range items {
		if !it.hasCore {
			continue
		}
		s, e := it.start, it.end
		if it.kind == CodeFence {
			if s-1 >= 0 && fenceRe.MatchString(remLines[s-1]) {
				s--
			}
			if e+1 < n && fenceRe.MatchString(remLines[e+1]) {
				e++
			}
		}
		for i := s; i <= e; i++ {
			claim(i, it.kind, it.node)
		}
	}

	// Pass B: blocks goldmark reports without any line range (ThematicBreak, and
	// degenerate/empty fenced code blocks). Placed onto the first matching
	// unclaimed line(s) in document order, preserving the node.
	for _, it := range items {
		if it.hasCore {
			continue
		}
		switch it.kind {
		case ThematicBreak:
			for i := 0; i < n; i++ {
				if !claimed[i] && thematicBreakRe.MatchString(remLines[i]) {
					claim(i, ThematicBreak, it.node)
					break
				}
			}
		case CodeFence:
			f1 := -1
			for i := 0; i < n; i++ {
				if !claimed[i] && fenceRe.MatchString(remLines[i]) {
					f1 = i
					break
				}
			}
			if f1 >= 0 {
				f2 := f1
				for i := f1 + 1; i < n && !claimed[i]; i++ {
					f2 = i
					if fenceRe.MatchString(remLines[i]) {
						break
					}
				}
				for i := f1; i <= f2; i++ {
					claim(i, CodeFence, it.node)
				}
			}
		}
	}

	// Pass C: fill any still-unclaimed line. A setext underline (=== or ---)
	// immediately after a Heading line is folded back into that Heading block —
	// goldmark exposes no line segment for the underline, so it lands here. Blank
	// stays Blank; a stray thematic break line is labeled as such (node nil);
	// everything else is Other.
	for i := 0; i < n; i++ {
		if claimed[i] {
			continue
		}
		switch {
		case i > 0 && kindOf[i-1] == Heading && setextUnderlineRe.MatchString(remLines[i]):
			kindOf[i] = Heading
			nodeOf[i] = nodeOf[i-1] // same node → coalesced into one Heading block
		case strings.TrimSpace(remLines[i]) == "":
			kindOf[i] = Blank
		case thematicBreakRe.MatchString(remLines[i]):
			kindOf[i] = ThematicBreak
		default:
			kindOf[i] = Other
		}
		claimed[i] = true
	}

	// Coalesce consecutive lines with the same kind and backing node into blocks.
	var blocks []Block
	for i := 0; i < n; i++ {
		start := i
		for i+1 < n && kindOf[i+1] == kindOf[start] && nodeOf[i+1] == nodeOf[start] {
			i++
		}
		blocks = append(blocks, Block{
			Kind:  kindOf[start],
			Start: start,
			End:   i,
			Node:  nodeOf[start],
		})
	}
	return blocks
}

// coreRange returns the inclusive [start, end] line range spanned by the content
// of a block node, derived from the min/max byte offsets of any descendant that
// carries line segments. It returns ok=false when the node exposes no line
// information at all (e.g. ThematicBreak or an empty fenced code block).
func coreRange(n ast.Node, offsets []int) (start, end int, ok bool) {
	minStart, maxStop := 1<<62, -1
	var walk func(ast.Node)
	walk = func(x ast.Node) {
		if x.Type() != ast.TypeBlock {
			return // Lines() panics on inline nodes; they carry no block range.
		}
		if l := x.Lines(); l != nil && l.Len() > 0 {
			if s := l.At(0).Start; s < minStart {
				minStart = s
			}
			if s := l.At(l.Len() - 1).Stop; s > maxStop {
				maxStop = s
			}
		}
		for c := x.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(n)
	if maxStop < 0 {
		return 0, 0, false
	}
	return lineOf(offsets, minStart), lineOf(offsets, maxStop-1), true
}

// lineOf maps a byte offset in the joined remainder to its line index.
func lineOf(offsets []int, off int) int {
	last := len(offsets) - 1
	if off >= offsets[last] {
		return last - 1
	}
	for i := 0; i+1 < len(offsets); i++ {
		if off < offsets[i+1] {
			return i
		}
	}
	return last - 1
}

// mapKind maps a goldmark node kind to a Block Kind.
func mapKind(n ast.Node) Kind {
	switch n.Kind() {
	case ast.KindHeading:
		return Heading
	case ast.KindParagraph, ast.KindTextBlock:
		return Paragraph
	case ast.KindFencedCodeBlock:
		return CodeFence
	case ast.KindCodeBlock:
		return IndentedCode
	case ast.KindList:
		return List
	case ast.KindBlockquote:
		return Blockquote
	case ast.KindThematicBreak:
		return ThematicBreak
	case east.KindTable:
		return Table
	default:
		return Other
	}
}
