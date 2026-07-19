package mdparse

import "testing"

func TestLinkAt(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		col      int
		wantDest string
		wantOK   bool
	}{
		{"inside label", "see [docs](note) here", 5, "note", true},
		{"on opening bracket", "see [docs](note) here", 4, "note", true},
		{"inside dest", "see [docs](note) here", 12, "note", true},
		{"before", "see [docs](note) here", 0, "", false},
		{"after", "see [docs](note) here", 17, "", false},
		{"url dest", "[x](https://example.com)", 1, "https://example.com", true},
		{"path dest", "[x](folder/a.md)", 1, "folder/a.md", true},
		{"not a wikilink", "[[nota]]", 3, "", false},
		{"wikilink then md", "[[a]] and [b](c)", 12, "c", true},
		{"in code span", "x `[y](z)` w", 4, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LinkAt(tc.line, tc.col)
			if ok != tc.wantOK || got != tc.wantDest {
				t.Errorf("LinkAt(%q, %d)=(%q,%v) want (%q,%v)",
					tc.line, tc.col, got, ok, tc.wantDest, tc.wantOK)
			}
		})
	}
}
