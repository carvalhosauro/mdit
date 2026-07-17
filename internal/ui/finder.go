package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type noteItem struct {
	name, path string
}

func (n noteItem) FilterValue() string { return n.name }
func (n noteItem) Title() string       { return n.name }
func (n noteItem) Description() string { return "" }

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
		return
	}
	notes := a.vault.List()
	items := make([]list.Item, len(notes))
	for i, n := range notes {
		items[i] = noteItem{name: n.Name, path: n.Path}
	}
	a.finder.SetItems(items)
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

func (a *App) viewFinder() string {
	box := a.finder.View()
	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Render(box)
	return centerBlock(frame, a.width, a.height)
}
