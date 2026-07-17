package doc

import (
	"strings"
	"time"
	"unicode/utf8"
)

// coalesceWindow is the maximum gap between two single-rune inserts for them
// to be coalesced into the same undo group.
const coalesceWindow = 500 * time.Millisecond

// patch is an inverse-patch undo/redo entry. It records enough information
// to both undo (restore oldText over the range newText now occupies) and
// redo (reapply newText over the range oldText occupies once undone).
type patch struct {
	from         Position
	oldText      string
	newText      string
	cursorBefore Position
	cursorAfter  Position

	// coalescible marks groups built purely from single-rune, no-"\n"
	// inserts, eligible to absorb further such inserts.
	coalescible bool
	lastTime    time.Time
}

// recordEdit updates the undo/redo stacks after a real (non-undo, non-redo)
// mutation. A new edit always clears the redo stack and bumps Version.
//
// Single-rune inserts without "\n", on the same line, at contiguous columns,
// within coalesceWindow of the previous one are merged into the same undo
// group instead of pushing a new entry.
func (d *Document) recordEdit(from Position, oldText, newText string, cursorBefore, cursorAfter Position) {
	d.redoStack = nil
	d.version++

	now := d.clock()
	singleRuneInsert := oldText == "" &&
		utf8.RuneCountInString(newText) == 1 &&
		!strings.ContainsRune(newText, '\n')

	if singleRuneInsert && len(d.undoStack) > 0 {
		top := d.undoStack[len(d.undoStack)-1]
		if top.coalescible && from == top.cursorAfter && now.Sub(top.lastTime) < coalesceWindow {
			top.newText += newText
			top.cursorAfter = cursorAfter
			top.lastTime = now
			return
		}
	}

	d.undoStack = append(d.undoStack, &patch{
		from:         from,
		oldText:      oldText,
		newText:      newText,
		cursorBefore: cursorBefore,
		cursorAfter:  cursorAfter,
		coalescible:  singleRuneInsert,
		lastTime:     now,
	})
}

// clock returns the current time via the injected now func, defaulting to
// time.Now so production documents (Load/NewFromString) behave normally
// while tests can control coalescing deterministically.
func (d *Document) clock() time.Time {
	if d.now != nil {
		return d.now()
	}
	return time.Now()
}

// Undo reverts the most recent undo group, returning the cursor position to
// restore and true. It returns false if the undo stack is empty.
func (d *Document) Undo() (Position, bool) {
	if len(d.undoStack) == 0 {
		return Position{}, false
	}
	n := len(d.undoStack)
	p := d.undoStack[n-1]
	d.undoStack = d.undoStack[:n-1]

	to := endPosition(p.from, p.newText)
	d.applyReplace(p.from, to, p.oldText)
	d.version++

	d.redoStack = append(d.redoStack, p)
	return p.cursorBefore, true
}

// Redo reapplies the most recently undone group, returning the cursor
// position to restore and true. It returns false if the redo stack is empty
// (including whenever a subsequent edit has invalidated it).
func (d *Document) Redo() (Position, bool) {
	if len(d.redoStack) == 0 {
		return Position{}, false
	}
	n := len(d.redoStack)
	p := d.redoStack[n-1]
	d.redoStack = d.redoStack[:n-1]

	to := endPosition(p.from, p.oldText)
	d.applyReplace(p.from, to, p.newText)
	d.version++

	d.undoStack = append(d.undoStack, p)
	return p.cursorAfter, true
}
