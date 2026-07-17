# mdit

**Your markdown, rendered as you type — without leaving the terminal.**

mdit is a WYSIWYG markdown editor for the terminal: every block is styled in place, and only the block under your cursor turns back into raw markdown for editing. No preview pane. No browser. No Electron.

![mdit demo — live render, wikilinks, fuzzy finder, zen mode](docs/demo/demo.gif)

## Install

```bash
go install github.com/carvalhosauro/mdit/cmd/mdit@latest
```

Requires Go 1.26+. Single static binary (`CGO_ENABLED=0`), zero config.

## Why not something else?

|  | mdit | glow | Obsidian | vim/emacs + plugins |
|--|:----:|:----:|:--------:|:-------------------:|
| Rendered markdown in the terminal | ✅ | ✅ | ❌ | partial |
| Edit and read in the same buffer | ✅ | ❌ read-only | ✅ | ❌ raw text |
| Wikilinks (`[[note]]`) across a vault | ✅ | ❌ | ✅ | plugins |
| Setup | none | none | GUI app | hours of config |

## Features

- **Live inline render** — headings, lists, code, quotes styled in place; the block under the cursor stays raw for editing
- **Obsidian-style wikilinks** — `[[note]]` / `[[note|alias]]` with follow (Enter or Ctrl+]), back (Ctrl+B), autocomplete on `[[`, and broken-link highlighting
- **Fuzzy note finder** — Ctrl+P over the vault
- **Zen mode** — Ctrl+E read-only, centered ≤80 columns
- **Safe save** — mtime conflict prompt (overwrite / reload / cancel); panic dumps to `<file>.mdit-recover`
- **Undo / redo** — coalesced edits (Ctrl+Z / Ctrl+Y)

## Usage

```bash
mdit note.md          # open a file (vault = its directory)
mdit ~/notes          # open a vault folder (newest note, or untitled.md)
mdit --version
```

## Keybindings

| Key | Action |
|-----|--------|
| Ctrl+S | Save |
| Ctrl+Q | Quit (prompt if unsaved) |
| Ctrl+P | Fuzzy note finder |
| Ctrl+E | Toggle zen mode |
| Enter / Ctrl+] | Follow wikilink under cursor |
| Ctrl+B | Back (navigation history) |
| Ctrl+Z / Ctrl+Y | Undo / Redo |
| Arrows, PgUp/PgDn, Home/End | Navigate |
| Ctrl+← / Ctrl+→ | Move by word |

## Why I built this

My notes are a folder of markdown files, and my day happens inside a terminal. Every time I wanted to *read* a note I had to leave it — open Obsidian, a browser tab, or pipe the file through a viewer that couldn't edit. mdit is the tool I wanted: open the note, see it rendered, fix the typo, close it.

— [Gustavo Carvalho](https://github.com/carvalhosauro)

## License

MIT License © 2026 Gustavo Carvalho
