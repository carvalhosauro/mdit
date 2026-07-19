// Package doc implements the line-buffer document model for mdit: rune-aware
// edits, undo/redo with coalescing, and safe (mtime-checked) saves. It has no
// dependency on any TUI library — it is consumed by the editor widget and UI
// layers built on top of it.
package doc

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Position identifies a location in the document buffer. Col is measured in
// runes, not bytes, so unicode text addresses correctly.
type Position struct {
	Line, Col int
}

// Document is a mutable line buffer with undo/redo history and safe-save
// support. The zero value is not usable; construct one with Load or
// NewFromString.
type Document struct {
	lines []string
	path  string

	version int

	// savedContent holds the buffer content (as returned by Content) at the
	// time of the last successful Load/NewFromString/Save, used by Dirty to
	// compare actual content rather than version counters (undoing back to
	// the saved state must report clean).
	savedContent string

	// dirty memoizes the last Dirty() result so the per-frame status bar
	// doesn't rebuild Content() every render. It is keyed by version (bumped
	// on every mutation) and invalidated explicitly on save, the only event
	// that changes dirtiness without a version bump.
	dirtyMemo  bool
	dirtyVer   int
	dirtyValid bool

	modTime time.Time

	// now supplies the current time for undo-coalescing decisions. It
	// defaults to time.Now (set by Load/NewFromString) and is settable in
	// tests for deterministic coalescing behavior.
	now func() time.Time

	undoStack []*patch
	redoStack []*patch
}

// NewFromString creates a document from in-memory text with no backing file.
// Save/SaveForce return an error until the document has a path (this task's
// interface has no SaveAs yet).
func NewFromString(s string) *Document {
	d := &Document{
		lines: splitLines(s),
		now:   time.Now,
	}
	d.savedContent = d.Content()
	return d
}

// splitLines turns file content into a line buffer that round-trips through
// Content(): a single trailing "\n" is not represented as an extra empty
// line, but Content() always re-appends exactly one.
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	lines := strings.Split(s, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// Lines returns a shallow copy of the buffer; callers must not mutate the
// document through it.
func (d *Document) Lines() []string {
	cp := make([]string, len(d.lines))
	copy(cp, d.lines)
	return cp
}

// Line returns line i (0-indexed).
func (d *Document) Line(i int) string {
	return d.lines[i]
}

// LineCount returns the number of lines in the buffer.
func (d *Document) LineCount() int {
	return len(d.lines)
}

// Path returns the file path associated with the document, or "" if the
// document was created with NewFromString and never saved.
func (d *Document) Path() string {
	return d.path
}

// Version increases by one on every mutation (edit, undo, or redo) and is
// intended as a cheap cache-invalidation key for renderers.
func (d *Document) Version() int {
	return d.version
}

// Dirty reports whether the document has unsaved changes. It compares the
// current content against the content at the last Load/NewFromString/Save,
// not version counters: undoing back to the saved state must report clean
// even though Version() has moved on.
func (d *Document) Dirty() bool {
	if d.dirtyValid && d.dirtyVer == d.version {
		return d.dirtyMemo
	}
	d.dirtyMemo = d.Content() != d.savedContent
	d.dirtyVer = d.version
	d.dirtyValid = true
	return d.dirtyMemo
}

// Content renders the buffer back to text: lines joined by "\n" with a
// trailing newline. An empty document (a single empty line) renders as the
// empty string so Save writes a true 0-byte file.
func (d *Document) Content() string {
	if len(d.lines) == 1 && d.lines[0] == "" {
		return ""
	}
	return strings.Join(d.lines, "\n") + "\n"
}

// clamp constrains p to a valid position within the buffer: Line is clamped
// to [0, len(d.lines)-1] and Col is clamped to [0, rune count of that line].
// The public mutators apply it to every incoming Position so they never
// panic on out-of-range input (e.g. a Position computed from stale layout
// state).
func (d *Document) clamp(p Position) Position {
	if p.Line < 0 {
		p.Line = 0
	} else if maxLine := len(d.lines) - 1; p.Line > maxLine {
		p.Line = maxLine
	}
	if p.Col < 0 {
		p.Col = 0
	} else if maxCol := utf8.RuneCountInString(d.lines[p.Line]); p.Col > maxCol {
		p.Col = maxCol
	}
	return p
}

// Insert inserts text (which may contain "\n") at p and returns the cursor
// position immediately after the inserted text.
func (d *Document) Insert(p Position, text string) Position {
	p = d.clamp(p)
	if text == "" {
		return p
	}
	end := d.applyReplace(p, p, text)
	d.recordEdit(p, "", text, p, end)
	return end
}

// DeleteRange removes the half-open range [from, to), joining lines if the
// range crosses a "\n". from/to are normalized so order doesn't matter. It
// returns the (now smaller) position, the cursor position after deletion.
func (d *Document) DeleteRange(from, to Position) Position {
	from = d.clamp(from)
	to = d.clamp(to)
	if to.Line < from.Line || (to.Line == from.Line && to.Col < from.Col) {
		from, to = to, from
	}
	if from == to {
		return from
	}
	old := d.textRange(from, to)
	d.applyReplace(from, to, "")
	d.recordEdit(from, old, "", to, from)
	return from
}

// DeleteBackward deletes the rune before p (like backspace), joining with
// the previous line if p is at column 0. It is a no-op at the start of the
// document.
func (d *Document) DeleteBackward(p Position) Position {
	p = d.clamp(p)
	if p.Col > 0 {
		return d.DeleteRange(Position{Line: p.Line, Col: p.Col - 1}, p)
	}
	if p.Line > 0 {
		prevLen := utf8.RuneCountInString(d.lines[p.Line-1])
		return d.DeleteRange(Position{Line: p.Line - 1, Col: prevLen}, p)
	}
	return p
}

// DeleteForward deletes the rune at p (like the delete key), joining with
// the next line if p is at the end of the line. It is a no-op at the end of
// the document.
func (d *Document) DeleteForward(p Position) Position {
	p = d.clamp(p)
	lineLen := utf8.RuneCountInString(d.lines[p.Line])
	if p.Col < lineLen {
		d.DeleteRange(p, Position{Line: p.Line, Col: p.Col + 1})
		return p
	}
	if p.Line < d.LineCount()-1 {
		d.DeleteRange(p, Position{Line: p.Line + 1, Col: 0})
		return p
	}
	return p
}

// textRange extracts the text in the half-open range [from, to), used for
// undo bookkeeping before the range is removed.
func (d *Document) textRange(from, to Position) string {
	if from.Line == to.Line {
		r := []rune(d.lines[from.Line])
		return string(r[from.Col:to.Col])
	}
	var b strings.Builder
	first := []rune(d.lines[from.Line])
	b.WriteString(string(first[from.Col:]))
	for l := from.Line + 1; l < to.Line; l++ {
		b.WriteByte('\n')
		b.WriteString(d.lines[l])
	}
	b.WriteByte('\n')
	last := []rune(d.lines[to.Line])
	b.WriteString(string(last[:to.Col]))
	return b.String()
}

// applyReplace replaces the half-open range [from, to) with newText and
// returns the cursor position immediately after the inserted text. It is the
// single mutation primitive: Insert, DeleteRange, Undo, and Redo all funnel
// through it. Callers are responsible for undo bookkeeping and the version
// bump.
func (d *Document) applyReplace(from, to Position, newText string) Position {
	firstLine := []rune(d.lines[from.Line])
	lastLine := []rune(d.lines[to.Line])
	prefix := string(firstLine[:from.Col])
	suffix := string(lastLine[to.Col:])

	parts := strings.Split(newText, "\n")
	var newLines []string
	if len(parts) == 1 {
		newLines = []string{prefix + parts[0] + suffix}
	} else {
		newLines = make([]string, 0, len(parts))
		newLines = append(newLines, prefix+parts[0])
		newLines = append(newLines, parts[1:len(parts)-1]...)
		newLines = append(newLines, parts[len(parts)-1]+suffix)
	}

	removed := to.Line - from.Line + 1
	replaced := make([]string, 0, len(d.lines)-removed+len(newLines))
	replaced = append(replaced, d.lines[:from.Line]...)
	replaced = append(replaced, newLines...)
	replaced = append(replaced, d.lines[to.Line+1:]...)
	d.lines = replaced

	return endPosition(from, newText)
}

// endPosition computes the cursor position immediately after conceptually
// inserting text at from, using rune-aware column arithmetic. Shared by
// applyReplace and the undo/redo bookkeeping in undo.go, both of which need
// it without necessarily having applied the mutation yet.
func endPosition(from Position, text string) Position {
	parts := strings.Split(text, "\n")
	if len(parts) == 1 {
		return Position{Line: from.Line, Col: from.Col + utf8.RuneCountInString(parts[0])}
	}
	return Position{Line: from.Line + len(parts) - 1, Col: utf8.RuneCountInString(parts[len(parts)-1])}
}
