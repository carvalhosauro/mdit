package editor

import "github.com/carvalhosauro/mdit/internal/mdparse"

// isStructural reports kinds that stay rendered under the cursor until the
// user signals edit intent (Enter or the first typed character). Textual kinds
// (paragraph, heading, list, quote, …) remain eager-raw.
func isStructural(k mdparse.Kind) bool {
	switch k {
	case mdparse.Table, mdparse.CodeFence, mdparse.IndentedCode:
		return true
	default:
		return false
	}
}

// cursorBlockRaw reports whether the cursor's block is currently shown raw.
func (m Model) cursorBlockRaw() bool {
	if m.zen || len(m.layouts) == 0 || m.cursorBlock < 0 || m.cursorBlock >= len(m.layouts) {
		return false
	}
	return m.layouts[m.cursorBlock].raw
}

// shouldLazyActivate is true when the cursor sits on a structural block that
// is still in the rendered (non-editing) state.
func (m Model) shouldLazyActivate() bool {
	if m.zen || m.editing || len(m.blocks) == 0 {
		return false
	}
	if m.cursorBlock < 0 || m.cursorBlock >= len(m.blocks) {
		return false
	}
	return isStructural(m.blocks[m.cursorBlock].Kind)
}

// activateEditing flips the lazy-raw gate on so the structural cursor block
// re-layouts as raw source.
func (m *Model) activateEditing() {
	m.editing = true
}
