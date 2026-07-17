# mdit

WYSIWYG markdown editor for your terminal — live inline render, wikilinks, zen mode.

## Features

- **Live inline render** — every markdown block is styled in place; the block under the cursor stays raw for editing
- **Wikilinks** — `[[note]]` / `[[note|alias]]` with follow (Ctrl+]), back (Ctrl+B), autocomplete on `[[`, and broken-link highlighting
- **Fuzzy note finder** — Ctrl+P over the vault
- **Zen mode** — Ctrl+E read-only, centered ≤80 columns
- **Safe save** — mtime conflict prompt (overwrite / reload / cancel); panic dumps to `<file>.mdit-recover`
- **Undo / redo** — coalesced edits (Ctrl+Z / Ctrl+Y)

## Install

```bash
go install github.com/carvalhosauro/mdit/cmd/mdit@latest
```

Requires Go 1.26+. Builds with `CGO_ENABLED=0`.

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
| Ctrl+] | Follow wikilink under cursor |
| Ctrl+B | Back (navigation history) |
| Ctrl+Z / Ctrl+Y | Undo / Redo |
| Arrows, PgUp/PgDn, Home/End | Navigate |
| Ctrl+← / Ctrl+→ | Move by word |

## License

MIT License © 2026 Gustavo Carvalho
