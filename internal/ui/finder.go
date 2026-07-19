package ui

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type noteItem struct {
	name, path string
	dir        string // vault-relative directory, "" at the root
}

func (n noteItem) FilterValue() string { return n.name }
func (n noteItem) Title() string       { return n.name }
func (n noteItem) Description() string { return n.dir }

func newFinderList(items []list.Item, width, height int) list.Model {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	l := list.New(items, d, width, height)
	l.Title = "Notes"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	return l
}

func (a *App) refreshFinderItems() {
	if a.vault == nil {
		a.finder.SetItems(nil)
		a.finder.Title = "Notes"
		return
	}
	notes := a.vault.List()
	root := a.vault.Root()
	items := make([]list.Item, len(notes))
	hasSubdir := false
	for i, n := range notes {
		dir := ""
		if rel, err := filepath.Rel(root, filepath.Dir(n.Path)); err == nil && rel != "." {
			dir = rel
			hasSubdir = true
		}
		items[i] = noteItem{name: n.Name, path: n.Path, dir: dir}
	}
	a.finder.SetItems(items)
	a.finder.Title = fmt.Sprintf("Notes (%d)", len(items))

	// Show the directory line only when it disambiguates something; a flat
	// vault keeps single-line rows (twice the visible items).
	d := list.NewDefaultDelegate()
	d.ShowDescription = hasSubdir
	d.SetSpacing(0)
	a.finder.SetDelegate(d)
}

func (a *App) resizeFinder() {
	fw := min(60, max(24, a.width*2/3))
	fh := min(18, max(10, a.height*2/3))
	if a.width > 0 && fw > a.width-4 {
		fw = a.width - 4
	}
	if a.height > 0 && fh > a.height-2 {
		fh = a.height - 2
	}
	if fw < 1 {
		fw = 1
	}
	if fh < 1 {
		fh = 1
	}
	a.finder.SetSize(fw, fh)
}

func (a *App) openFinder() {
	if a.vault != nil {
		_ = a.vault.Rescan()
	}
	a.refreshFinderItems()
	a.resizeFinder()
	a.mode = modeFinder
	a.statusErr = ""
	a.statusMsg = ""
}

func (a *App) handleFinderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		a.mode = modeEdit
		return a, nil
	case tea.KeyEnter:
		// While filtering, let the list accept the filter first if needed.
		if a.finder.FilterState() == list.Filtering {
			var cmd tea.Cmd
			a.finder, cmd = a.finder.Update(msg)
			return a, cmd
		}
		item, ok := a.finder.SelectedItem().(noteItem)
		if !ok {
			return a, nil
		}
		a.mode = modeEdit
		return a.requestOpen(item.path)
	}

	var cmd tea.Cmd
	a.finder, cmd = a.finder.Update(msg)
	return a, cmd
}

// viewFinder draws the finder as a modal over the dimmed editor, so the note
// being edited stays visible as context instead of vanishing.
func (a *App) viewFinder() string {
	content := a.finder.View()
	// P3: an empty vault gets a hint instead of a bare, blank list.
	if len(a.finder.Items()) == 0 {
		content += "\n" + a.theme.Dim.Render("No notes yet — save a note to populate the vault.")
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.MenuBorder.GetForeground()).
		Padding(0, 1).
		Render(content)
	bg := a.dimContent(a.editor.View() + "\n" + a.renderStatusBar())
	return overlayCenter(bg, box, a.width, a.height)
}
