// Package render turns a single mdparse block into styled screen lines. It is a
// pure function of (block content, Width): given the same inputs it returns the
// same output, so the editor can cache results keyed on that pair. Every
// returned line's printable width is at most Context.Width cells (measured on
// ANSI-stripped text), and every block yields at least one line.
package render

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wrap"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/carvalhosauro/mdit/internal/mdparse"
	"github.com/carvalhosauro/mdit/internal/theme"
)

// Context carries everything a render needs beyond the block itself.
type Context struct {
	Width int
	Theme theme.Theme
	// IsBroken reports whether a wikilink target has no backing note. A nil
	// IsBroken means links are never treated as broken.
	IsBroken func(target string) bool
}

// Block renders block i of res into screen lines, each at most ctx.Width cells
// wide. A Blank block renders as []string{""}; the result always has length >= 1.
func Block(res mdparse.Result, i int, ctx Context) []string {
	if ctx.Width < 1 {
		ctx.Width = 1
	}
	b := res.Blocks[i]

	var lines []string
	switch b.Kind {
	case mdparse.Blank:
		return []string{""}
	case mdparse.Heading:
		lines = renderHeading(b, res.Source, ctx)
	case mdparse.Paragraph:
		lines = renderParagraph(b, res.Source, ctx)
	case mdparse.CodeFence:
		lines = renderFence(b, res.Source, ctx)
	case mdparse.IndentedCode:
		lines = renderIndentedCode(b, res.Source, ctx)
	case mdparse.List:
		lines = renderList(b, res.Source, ctx)
	case mdparse.Table:
		lines = renderTable(b, res.Source, ctx)
	case mdparse.Blockquote:
		lines = renderBlockquote(b, res.Source, ctx)
	case mdparse.ThematicBreak:
		lines = renderThematicBreak(ctx)
	case mdparse.Frontmatter:
		lines = renderRaw(res, b, ctx.Theme.Frontmatter, ctx)
	default: // Other
		lines = renderRaw(res, b, ctx.Theme.Text, ctx)
	}

	return fit(lines, ctx.Width)
}

// fit guarantees the width invariant defensively: any line still exceeding width
// is truncated ANSI-safely with an ellipsis. It also guarantees length >= 1.
func fit(lines []string, width int) []string {
	if len(lines) == 0 {
		return []string{""}
	}
	for i, ln := range lines {
		if cellWidth(ln) > width {
			lines[i] = ansi.Truncate(ln, width, "…")
		}
	}
	return lines
}

// rawLines recovers the raw source lines covered by block b.
func rawLines(res mdparse.Result, b mdparse.Block) []string {
	all := strings.Split(string(res.Source), "\n")
	if b.Start < 0 || b.End >= len(all) {
		return nil
	}
	return all[b.Start : b.End+1]
}

func renderHeading(b mdparse.Block, source []byte, ctx Context) []string {
	level := 1
	if h, ok := b.Node.(*ast.Heading); ok {
		level = h.Level
	}
	style := headingStyle(ctx.Theme, level)
	var out []string
	for _, ln := range wrapStyled(plainInline(b.Node, source), ctx.Width) {
		out = append(out, style.Render(ln))
	}
	if len(out) == 0 {
		out = []string{style.Render("")}
	}
	return out
}

func renderParagraph(b mdparse.Block, source []byte, ctx Context) []string {
	if b.Node == nil {
		return renderRaw(mdparse.Result{Source: source}, b, ctx.Theme.Text, ctx)
	}
	return wrapStyled(renderInlines(b.Node, source, ctx), ctx.Width)
}

func renderThematicBreak(ctx Context) []string {
	return []string{ctx.Theme.ThematicBreak.Render(strings.Repeat("─", ctx.Width))}
}

// renderRaw styles each raw source line with style, hard-wrapping over-long
// lines so the width invariant holds.
func renderRaw(res mdparse.Result, b mdparse.Block, style lipgloss.Style, ctx Context) []string {
	var out []string
	for _, ln := range rawLines(res, b) {
		if cellWidth(ln) <= ctx.Width {
			out = append(out, style.Render(ln))
			continue
		}
		for _, hl := range strings.Split(wrap.String(ln, ctx.Width), "\n") {
			out = append(out, style.Render(hl))
		}
	}
	if len(out) == 0 {
		out = []string{style.Render("")}
	}
	return out
}

// --- blockquote ---

func renderBlockquote(b mdparse.Block, source []byte, ctx Context) []string {
	th := ctx.Theme
	prefix := th.Quote.Render("│ ")
	inner := ctx.Width - 2
	if inner < 1 {
		inner = 1
	}
	var body []string
	first := true
	if b.Node != nil {
		for c := b.Node.FirstChild(); c != nil; c = c.NextSibling() {
			if !first {
				body = append(body, "") // blank separator between child blocks
			}
			first = false
			body = append(body, wrapStyled(renderInlines(c, source, ctx), inner)...)
		}
	}
	if len(body) == 0 {
		body = []string{""}
	}
	out := make([]string, len(body))
	for i, ln := range body {
		if strings.TrimSpace(ansi.Strip(ln)) == "" {
			out[i] = th.Quote.Render("│")
			continue
		}
		out[i] = prefix + ln
	}
	return out
}

// --- fenced / indented code ---

func renderFence(b mdparse.Block, source []byte, ctx Context) []string {
	th := ctx.Theme
	code, lang := codeContent(b.Node, source)

	if hl, ok := highlight(code, lang, th.ChromaStyle); ok {
		return codeLines(strings.Split(hl, "\n"), ctx.Width, nil)
	}
	return codeLines(strings.Split(code, "\n"), ctx.Width, &th.CodeBlock)
}

// renderIndentedCode renders an indented code block. Unlike a fenced block it
// has no info string and no language, so it always uses the plain CodeBlock
// style (no chroma highlighting) and there are no fence delimiter lines to strip.
func renderIndentedCode(b mdparse.Block, source []byte, ctx Context) []string {
	th := ctx.Theme
	code, _ := codeContent(b.Node, source)
	return codeLines(strings.Split(code, "\n"), ctx.Width, &th.CodeBlock)
}

// codeLines normalizes code output: drops a single trailing empty line, applies
// an optional style, and hard-wraps any line wider than width (mid-token wrap is
// acceptable for code — correctness of text beats aesthetics).
func codeLines(raw []string, width int, style *lipgloss.Style) []string {
	if n := len(raw); n > 0 && raw[n-1] == "" {
		raw = raw[:n-1]
	}
	var out []string
	for _, ln := range raw {
		pieces := []string{ln}
		if cellWidth(ln) > width {
			pieces = strings.Split(wrap.String(ln, width), "\n")
		}
		for _, p := range pieces {
			if style != nil {
				p = style.Render(p)
			}
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

// codeContent extracts the code body and language from a fenced or indented code
// block node.
func codeContent(n ast.Node, source []byte) (code, lang string) {
	switch t := n.(type) {
	case *ast.FencedCodeBlock:
		lang = string(t.Language(source))
		code = linesText(t.Lines(), source)
	case *ast.CodeBlock:
		code = linesText(t.Lines(), source)
	}
	return code, lang
}

func linesText(segs *text.Segments, src []byte) string {
	if segs == nil {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < segs.Len(); i++ {
		seg := segs.At(i)
		sb.Write(seg.Value(src))
	}
	return sb.String()
}

func highlight(code, lang, styleName string) (string, bool) {
	lexer := lexers.Get(lang)
	if lexer == nil {
		return "", false
	}
	lexer = chroma.Coalesce(lexer)
	style := styles.Get(styleName)
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return "", false
	}
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", false
	}
	var sb strings.Builder
	if err := formatter.Format(&sb, style, it); err != nil {
		return "", false
	}
	return sb.String(), true
}

// --- table ---

func renderTable(b mdparse.Block, source []byte, ctx Context) []string {
	th := ctx.Theme
	tbl, ok := b.Node.(*east.Table)
	if !ok {
		return []string{""}
	}

	var header []string
	var rows [][]string
	for c := tbl.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.Kind() {
		case east.KindTableHeader:
			header = rowCells(c, source)
		case east.KindTableRow:
			rows = append(rows, rowCells(c, source))
		}
	}

	cols := len(header)
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	if cols == 0 {
		return []string{""}
	}

	widths := make([]int, cols)
	measure := func(cells []string) {
		for i, cell := range cells {
			if w := runewidth.StringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}
	measure(header)
	for _, r := range rows {
		measure(r)
	}
	widths = fitColumns(widths, ctx.Width)

	var out []string
	if len(header) > 0 {
		out = append(out, renderRow(header, widths, th.TableHeader))
		out = append(out, separatorRow(widths, th.Table))
	}
	for _, r := range rows {
		out = append(out, renderRow(r, widths, th.Table))
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

func rowCells(row ast.Node, source []byte) []string {
	var cells []string
	for c := row.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == east.KindTableCell {
			cells = append(cells, strings.TrimSpace(plainInline(c, source)))
		}
	}
	return cells
}

// fitColumns shrinks column widths (proportionally) so the assembled row fits
// within width. Separators (" │ ") cost 3 cells between columns.
func fitColumns(widths []int, width int) []int {
	cols := len(widths)
	avail := width - 3*(cols-1)
	if avail < cols {
		avail = cols // one cell each minimum; fit() guards any residual overflow
	}
	sum := 0
	for i, w := range widths {
		if w < 1 {
			widths[i] = 1
			w = 1
		}
		sum += w
	}
	if sum <= avail {
		return widths
	}
	// Scale down proportionally, keeping at least 1 cell per column.
	out := make([]int, cols)
	for i, w := range widths {
		scaled := w * avail / sum
		if scaled < 1 {
			scaled = 1
		}
		out[i] = scaled
	}
	return out
}

func renderRow(cells []string, widths []int, style lipgloss.Style) string {
	parts := make([]string, len(widths))
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		cell = runewidth.Truncate(cell, widths[i], "…")
		cell = runewidth.FillRight(cell, widths[i])
		parts[i] = style.Render(cell)
	}
	return strings.Join(parts, " "+style.Render("│")+" ")
}

func separatorRow(widths []int, style lipgloss.Style) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = style.Render(strings.Repeat("─", w))
	}
	return strings.Join(parts, style.Render("─┼─"))
}

// --- list ---

func renderList(b mdparse.Block, source []byte, ctx Context) []string {
	list, ok := b.Node.(*ast.List)
	if !ok {
		return []string{""}
	}
	var out []string
	appendList(list, source, ctx, 0, &out)
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

func appendList(list *ast.List, source []byte, ctx Context, depth int, out *[]string) {
	ordinal := list.Start
	if ordinal < 1 {
		ordinal = 1
	}
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		if item.Kind() != ast.KindListItem {
			continue
		}
		appendItem(item, list, ordinal, source, ctx, depth, out)
		ordinal++
	}
}

func appendItem(item ast.Node, list *ast.List, ordinal int, source []byte, ctx Context, depth int, out *[]string) {
	indent := strings.Repeat("  ", depth)

	marker, isTask := itemMarker(item, list, ordinal, ctx.Theme, depth)
	markerWidth := runewidth.StringWidth(ansi.Strip(marker))

	textWidth := ctx.Width - runewidth.StringWidth(indent) - markerWidth
	if textWidth < 1 {
		textWidth = 1
	}

	// The item's own text is its first paragraph/text block; nested lists are
	// rendered afterwards at depth+1.
	wrapped := wrapStyled(itemText(item, source, ctx, isTask), textWidth)
	cont := indent + strings.Repeat(" ", markerWidth)
	for i, ln := range wrapped {
		if i == 0 {
			*out = append(*out, indent+marker+ln)
		} else {
			*out = append(*out, cont+ln)
		}
	}

	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		if sub, ok := c.(*ast.List); ok {
			appendList(sub, source, ctx, depth+1, out)
		}
	}
}

// itemMarker returns the styled marker (with trailing space) for a list item and
// whether the item is a task item.
func itemMarker(item ast.Node, list *ast.List, ordinal int, th theme.Theme, depth int) (marker string, isTask bool) {
	if box := taskCheckBox(item); box != nil {
		if box.IsChecked {
			return th.TaskDone.Render("☑") + " ", true
		}
		return th.TaskOpen.Render("☐") + " ", true
	}
	if list.IsOrdered() {
		return th.Bullet.Render(itoa(ordinal)+".") + " ", false
	}
	glyph := "•"
	if depth > 0 {
		glyph = "◦"
	}
	return th.Bullet.Render(glyph) + " ", false
}

// taskCheckBox returns the TaskCheckBox at the head of a list item's first
// block, or nil if the item is not a task item.
func taskCheckBox(item ast.Node) *east.TaskCheckBox {
	block := item.FirstChild()
	if block == nil {
		return nil
	}
	if box, ok := block.FirstChild().(*east.TaskCheckBox); ok {
		return box
	}
	return nil
}

// itemText renders the inline text of a list item's first paragraph/text block,
// skipping any leading task checkbox.
func itemText(item ast.Node, source []byte, ctx Context, isTask bool) string {
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		if k := c.Kind(); k == ast.KindParagraph || k == ast.KindTextBlock {
			s := renderInlines(c, source, ctx)
			if isTask {
				s = strings.TrimLeft(s, " ")
			}
			return s
		}
	}
	return ""
}

func headingStyle(t theme.Theme, level int) lipgloss.Style {
	switch level {
	case 1:
		return t.H1
	case 2:
		return t.H2
	case 3:
		return t.H3
	case 4:
		return t.H4
	case 5:
		return t.H5
	default:
		return t.H6
	}
}

// itoa is a tiny strconv.Itoa avoiding an import for one call site.
func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
