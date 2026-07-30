# mdit — Design v3 / S1: Table Widget

**Data:** 2026-07-30
**Status:** Pronto para implementação (arquitetura travada no ROADMAP)
**Escopo:** só **S1** (editor de tabela). S2/S3 são follow-ups.
**Licença do projeto:** MIT (OSS)

**Fontes de verdade:**
- `docs/ROADMAP.md` — seção **v3** + **Decisões travadas**
- Pré-requisitos entregues: S0 selection/clipboard, L1 lazy-raw
- Formato espelha: `docs/superpowers/specs/2026-07-16-mdit-mvp-design.md`

## Visão geral

No MVP/lazy-raw, o bloco sob o cursor é **cru** (texto) ou **renderizado**.
Para tabelas GFM, editar o pipe-source é frágil (alinhar `|`, separator, colunas).
S1 troca o branch **cru** da tabela por um **widget estruturado**: navega célula,
edita conteúdo isolado, altera grade (linha/coluna/alinhamento) e, ao sair,
reescreve o bloco markdown de uma vez.

O widget é o **3º estado** do `cursorBlock` no **mesmo seam** do lazy-raw
(`internal/editor/layout.go`, condição `raw := …`). Arquitetura
**editor-centric** (ROADMAP opção B): o editor segura `active blockedit.Widget`;
a lógica pesada mora em `internal/blockedit` (testável sem TUI).

## Objetivos (S1)

1. `Kind Table` sob cursor → Enter (ou 1º caractere imprimível) abre o **table widget**
   (não o raw source).
2. Navegação de células: Tab / Shift+Tab / setas; edição de célula isolada.
3. Operações de grade: add/del linha, add/del coluna, ciclo de alinhamento,
   auto-resize do separator no commit.
4. **Commit atômico** via `doc.ReplaceLines` → 1 grupo de undo; **Esc cancela**
   sem escrever no doc.
5. Code fence / indented code continuam no lazy-raw atual (fora de S1).
6. Cursor cru reposto best-effort ao sair (Princípio 6 do ROADMAP).

## Não-objetivos (S1)

| Item | Por quê |
|------|---------|
| Widgets para code / link / callout (S2) | Follow-up; reusa a interface de S1. |
| In-doc find `^F` (S3) | Ortogonal; não bloqueia tabela. |
| Escape hatch “editar raw da tabela” | Em S1, tabela só edita via widget. Malformed → ver abaixo. |
| Células com markdown rico (negrito, links, wikilinks) | Conteúdo de célula = texto plano (trim); inline markup no source é preservado só se o parse→serialize round-trip for literal por célula (S1: plain text). |
| Merge de células / tabelas aninhadas / HTML tables | Fora do GFM pipe table. |
| Mouse / vim modes / preview pane | Princípios 1–3 do ROADMAP. |
| Reabrir arquitetura App-centric (opção C) | Travado: opção B. |
| Callout como structural no lazy-raw | Hoje `isStructural` = Table/CodeFence/IndentedCode; callouts são Blockquote. Fora de S1. |

## Decisões de escopo (com alternativas rejeitadas)

| Decisão | Escolha | Rejeitado |
|---|---|---|
| Posse do widget | Editor (`active`), pacote `internal/blockedit` | App-centric (C); widget embutido em `editor` sem pacote |
| Seam | Mesmo `layout.go` do lazy-raw; 3º branch `widget` | Novo viewport / overlay fullscreen |
| Mutação | Nada no `doc` até Commit; `ReplaceLines` 1× | Edit por tecla no buffer (reparse a cada rune) |
| Cancel | Esc → discard, `Version()` intacto | Esc = commit |
| Ativação (tabela) | Enter **ou** 1º rune → widget | Só Enter; ou continuar raw e “promover” depois |
| Fence/code em S1 | Permanecem lazy-raw | Forçar widget de linguagem já em S1 |
| Undo durante widget | `^Z`/`^Y` **ignorados** no widget (edits só em memória) | Undo local por célula (mais estado) |
| Sair com commit | Motions que **saem do bloco** → Commit; Esc → Cancel | Só commit explícito (Enter) |

## Arquitetura

### Estados do cursorBlock (seam)

Para o índice `i == cursorBlock` (e `!zen`):

| Estado | Quando | O que `layout` materializa |
|--------|--------|----------------------------|
| **rendered** | Structural sem intent; ou textual fora de eager-raw N/A | `renderedLines(i)` (cacheado) |
| **raw** | Structural **não-tabela** com `editing==true` (fence/code); textual eager | `rawBlockLines(i)` (não cacheado) |
| **widget** | `active != nil` (S1: só Table) | `active.Lines(width)` (não cacheado) |

Pseudocódigo do seam (evolução de `layout.go`):

```
if !zen && i == cursorBlock && m.active != nil {
    lines = m.active.Lines(width)          // widget
} else {
    raw := !zen && i == cursorBlock && (!isStructural(kind) || m.editing)
    lines = raw ? rawBlockLines(i) : renderedLines(i)
}
```

`editing` (lazy-raw) e `active` (widget) são **mutuamente exclusivos**.
Abrir widget: `active = …; editing = false`.
Fechar widget: `active = nil` (+ Commit ou Cancel).

### Pacote `internal/blockedit`

Interface pura (sem dependência de `editor`):

```go
package blockedit

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/carvalhosauro/mdit/internal/doc"
    "github.com/carvalhosauro/mdit/internal/mdparse"
)

// Signal is how a widget tells the editor to keep going or exit.
type Signal int

const (
    Continue Signal = iota // keep widget active
    Commit                 // write CommitLines() via ReplaceLines, then clear active
    Cancel                 // discard; doc untouched
)

// Widget is the structured editor for one block. Testable without a TUI loop.
type Widget interface {
    // Update handles input. Returns the (possibly replaced) widget, a tea.Cmd,
    // and a signal. On Commit/Cancel the editor must clear active after applying.
    Update(msg tea.Msg) (Widget, tea.Cmd, Signal)

    // Lines returns screen rows for the block viewport (≤ width printable cells
    // where practical; soft-wrap cells like render.Table).
    Lines(width int) []string

    // CommitLines returns the raw markdown lines that replace the block range.
    CommitLines() []string

    // ExitCursor returns a best-effort raw doc.Position after leaving the widget
    // (absolute coordinates in the document as it will exist after Commit, or
    // the pre-open cursor on Cancel).
    ExitCursor(sig Signal) doc.Position
}

// OpenTable builds a table widget from the block's raw lines.
// ok=false → caller falls back to lazy-raw (malformed / not a pipe table).
func OpenTable(rawLines []string, cursor doc.Position, blockStart int) (w Widget, ok bool)
```

Factory futura (S2): `Open(kind mdparse.Kind, …)` despacha; S1 só exporta `OpenTable`.

### `doc.ReplaceLines`

Não existe hoje — é artefato novo do v3 (também útil se lazy-raw um dia
reescrever blocos em lote). Contrato:

```go
// ReplaceLines replaces the inclusive line range [start, end] with lines.
// len(lines) may differ from end-start+1. Empty lines is allowed (deletes the
// range). Records a single undo patch (not coalescible with typing).
// Returns the cursor position at the start of the first inserted line (Col 0),
// clamped. Panics are forbidden: start/end are clamped like other doc ops.
func (d *Document) ReplaceLines(start, end int, lines []string) Position
```

Implementação sugerida: um `applyReplace` de `Position{start,0}` até
`Position{end, len(runes(line end))}` com `newText = strings.Join(lines, "\n")`
(sem newline final extra além do join), **ou** mutação direta de `d.lines` +
`recordEdit` com `oldText`/`newText` multi-linha — desde que **um** patch e
`Version++` uma vez. `coalescible=false`.

### Fluxo de dados

```
keypress → editor.handleKey
  ├─ se active != nil → active.Update
  │     Continue → recompute (só View do widget)
  │     Commit   → doc.ReplaceLines(block.Start, block.End, CommitLines)
  │                active=nil; cursor=ExitCursor(Commit); recompute
  │     Cancel   → active=nil; cursor=ExitCursor(Cancel); recompute
  ├─ se Table + shouldOpenWidget (Enter / 1º rune) → OpenTable; se !ok → lazy-raw
  └─ senão → caminho atual (lazy-raw / eager text / selection)
```

Durante o widget: **nenhuma** chamada a `Insert`/`Delete*` no doc.
`doc.Version()` só muda no Commit.

### Interação com lazy-raw

| Kind | Enter / 1º rune hoje (L1) | Após S1 |
|------|---------------------------|---------|
| Table | `editing=true` → raw | `active=OpenTable` → widget; se `OpenTable` falhar → raw (fallback) |
| CodeFence / IndentedCode | lazy-raw | **inalterado** |
| Paragraph / Heading / List / Quote | eager-raw | **inalterado** |

Testes L1 de **tabela** que esperam raw após Enter devem ser **atualizados**
para esperar widget (não raw). Testes de fence/parágrafo permanecem.

Sair do bloco com lazy-raw (`editing=false` ao mudar `cursorBlock`) continua;
com widget, motion que cruzaria a borda do bloco → **Commit** (não discard),
depois move o cursor. Esc no widget → Cancel (equivalente ao Esc do lazy-raw
que desarma sem mutar — e aqui também não há mutação pendente no doc).

### Selection (S0)

Ao abrir o widget: `clearSelection()`. Enquanto `active != nil`, teclas de
seleção/shift e yank/paste **não** operam no doc (o widget pode ignorar ou,
opcionalmente, tratar paste como texto na célula — S1: paste (`^V` / bracketed)
insere na célula focada; copy da seleção de célula é nice-to-have, não gate).

### Zen mode

Zen continua sem raw e sem widget (`active` forçado nil / Update ignora abertura).

## Modelo do table widget

### Dados em memória

```text
header  []string           // row 0 (GFM exige header)
align   []Align            // por coluna: None | Left | Center | Right
body    [][]string         // rows ≥ 0; S1 garante ≥1 ao abrir se source tinha
focus   {row, col}         // row 0 = header; body starts at 1
cellCol int                // cursor rune dentro da célula focada
```

Parse a partir das linhas cruas do bloco (pipe table GFM):
- Linha header `| a | b |`
- Separator `| --- | :---: |`
- Body rows

Serialize (`CommitLines`):
- Reemite pipes, espaços consistentes, separator com largura ≥3 (`---`) e
  marcadores de alinhamento (`:---` / `:---:` / `---:`).
- **Auto-resize:** largura visual de cada coluna = max(`runewidth` das células,
  mínimo do marker de align); padding com `-` no separator.

### Keybindings (UI em inglês; hints na statusbar quando ativo)

| Tecla | Ação |
|---|---|
| Tab | Próxima célula (wrap na grade) |
| Shift+Tab | Célula anterior |
| ←/→ | Dentro da célula; na borda → célula vizinha |
| ↑/↓ | Célula na mesma coluna (header ↔ body) |
| Runas / Space / Backspace / Delete | Editar célula focada |
| Enter | Célula abaixo (mesma col); na última body row **não** cria linha sozinho |
| Esc | **Cancel** (descarta; doc intacto) |
| Ctrl+Shift+↓ | Inserir row abaixo da focada (não acima do header) |
| Ctrl+Shift+↑ | Apagar row focada (header intocável; última body row → limpa células, não remove grade) |
| Ctrl+Shift+→ | Inserir coluna à direita |
| Ctrl+Shift+← | Apagar coluna focada (mínimo 1 coluna) |
| Ctrl+L | Ciclar alinhamento da coluna focada: none → left → center → right → none |

Motion ↑ na primeira row / ↓ na última que **sairia do bloco** (editor detecta
tentativa de `moveVertical` para fora do range do bloco): **Commit** + aplica o
movimento no doc pós-replace. Left/Right não saem do bloco por si.

`^Z` / `^Y` / `^S` / `^Q` / `^P` / zen / follow: com widget ativo, o editor
**primeiro Commit** (salvo Esc path) **ou** deixa o App rotear só depois do
commit — regra S1: atalhos globais de App (`^S`/`^Q`/`^P`/`^E`) forçam
**Commit** implícito antes de propagar; `Esc` é só cancel do widget (não limpa
flash da App — a App já trata Esc no modo edit quando o editor não consome).

### View do widget

- Aparência próxima de `render.renderTable` (box / `│` / header style), com a
  célula focada em **reverse** (reusar `theme.RawBlock` ou estilo dedicado
  `theme.TableFocus` se necessário — preferir reusar para menos churn).
- Soft-wrap de células longas como o render atual.
- Altura = `len(Lines(width))`; entra no prefix sum como raw (não cacheada).

### Malformed / fallback

`OpenTable` retorna `ok=false` se:
- < 2 linhas, ou separator inválido, ou zero colunas.

Nesse caso o editor usa o caminho lazy-raw existente (`activateEditing()`),
para o usuário ainda conseguir consertar o source.

## Undo

- Durante o widget: sem patches no `doc`.
- No Commit: **um** `ReplaceLines` → **um** undo. `^Z` restaura o bloco inteiro
  anterior e o cursor do patch.
- Cancel: zero patches; `Dirty()`/`Version()` inalterados.

## App / statusbar

Editor expõe algo mínimo para hints (sem lógica de App no widget):

```go
func (m Model) BlockEditHint() string // "" | "table │ tab cell · ^⇧↓ row · esc cancel"
```

A statusbar (`internal/ui`) mostra o hint quando não vazio (S1 pode ser task
fina no fim). Mensagens em **inglês**.

## Testes

- `internal/doc`: `ReplaceLines` — resize de range, undo único, clamp, unicode.
- `internal/blockedit`: parse/serialize round-trip; navegação; edit célula;
  add/del row/col; align; auto-resize; Esc signal vs Commit signal (unit, sem tea loop completo — `Update` com `tea.KeyMsg`).
- `internal/editor`: seam widget vs raw vs rendered; Enter abre widget em Table;
  fence ainda lazy-raw; Esc cancela sem `Version++`; leave-block faz Commit;
  teatest smoke: editar célula → sair → markdown atualizado.
- Gate: `CGO_ENABLED=0`, `go test ./...`, `go vet ./...`, `gofmt -l .` vazio.

## PRs / release train

Branches `feat/*` → PR em **`stable`** (merge commit), conforme
`docs/RELEASE_FLOW.md`. Não abrir PR direto em `main`.

## Follow-ups (não implementar em S1)

- **S2:** generalizar `Open(kind)` — code fence (language picker), links,
  callouts (trocar tipo).
- **S3:** in-doc find (`^F`).
- Migração B→C (posse no App) só se o v3 ficar App-heavy — `blockedit` não muda.

## Referência competitiva / motivação

Editar pipe-tables no terminal como texto é a maior fonte de atrito estrutural
pós-lazy-raw. O widget mantém o pitch WYSIWYG (bloco editável no lugar) sem
violár o invariante do cursor cru: o cursor “de verdade” só é reconciliado na
saída.
