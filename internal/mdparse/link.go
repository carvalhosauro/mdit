package mdparse

import (
	"regexp"
	"unicode/utf8"
)

// mdLinkRe matches a CommonMark-style [label](destination) span. Destination
// may not contain ')' (no nested parens); label may not contain ']'. Wikilinks
// ([[...]]) are a different form and are handled by WikiLinkAt.
var mdLinkRe = regexp.MustCompile(`\[([^\[\]]+)\]\(([^)]+)\)`)

// LinkAt reports the markdown link destination if the rune column col falls
// within a [label](dest) span on the given raw line. col is measured in runes.
// Prefer WikiLinkAt first when both could apply — [[x]] is never a md link.
func LinkAt(line string, col int) (dest string, ok bool) {
	codeSpans := codeSpanRanges(line)
	for _, m := range mdLinkRe.FindAllStringSubmatchIndex(line, -1) {
		if overlapsAny(m[0], m[1], codeSpans) {
			continue
		}
		// Skip spans that are actually the inside of a wikilink: a match
		// starting at '[' that is preceded by another '[' is [[..., not [label].
		if m[0] > 0 && line[m[0]-1] == '[' {
			continue
		}
		startRune := utf8.RuneCountInString(line[:m[0]])
		endRune := utf8.RuneCountInString(line[:m[1]])
		if col >= startRune && col <= endRune-1 {
			return line[m[4]:m[5]], true
		}
	}
	return "", false
}
