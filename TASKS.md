# mdit — TASKS (handoff para qualquer agente: Cursor, Claude Code, etc.)

> **MVP (feito):** `docs/superpowers/plans/2026-07-16-mdit-mvp.md` +
> `docs/superpowers/specs/2026-07-16-mdit-mvp-design.md`.
> **v3 / S1 (próximo):** `docs/superpowers/plans/2026-07-30-mdit-v3-table-widget.md` +
> `docs/superpowers/specs/2026-07-30-mdit-v3-table-widget-design.md`.
> Roadmap / decisões travadas: `docs/ROADMAP.md`. Release train: `docs/RELEASE_FLOW.md`
> (`feat/*` → `stable`).
>
> Processo: TDD por task → commit convencional → code review → QA
> (`CGO_ENABLED=0 go test ./... && go vet ./... && gofmt -l .` limpos).
> UI em inglês.

## Critérios de aceite globais (valem para TODAS as tasks)

- `go build ./...`, `go vet ./...`, `go test ./...` verdes; `gofmt -l .` vazio; `CGO_ENABLED=0`.
- Cursor sempre em coordenadas cruas (linha, coluna-em-runes) fora de widgets; tela é derivada.
- Mensagens de UI em inglês. Commits conventional commits.
- PRs: branch `feat/*` → base **`stable`** (não `main`).

## Status — MVP (histórico)

| # | Task | Status | Commits |
|---|------|--------|---------|
| 1 | Scaffold (go.mod, LICENSE MIT, README, main --version) | ✅ | 1640f45, ae8c9cb |
| 2 | `internal/doc` — buffer, edits, undo/redo, save | ✅ | c6ecd22 |
| 3 | `internal/vault` — índice nome→path, Resolve | ✅ | 839e0a6, e4eb8f7 |
| 4 | `internal/mdparse` — goldmark+GFM → `[]Block`, wikilink | ✅ | 53af578, 65a8ef6 |
| 5 | `internal/theme` + `internal/render` | ✅ | 084878e |
| 6 | `internal/editor` — viewport, bloco-sob-cursor | ✅ | e2254d4 |
| 7 | `internal/ui` + `cmd/mdit` | ✅ | 6a89f2f, e591a1c |
| 8 | Fuzzy finder (Ctrl+P) | ✅ | 704b569 |
| 9 | Wikilinks follow / back / autocomplete | ✅ | 704b569 |
| 10 | Zen mode (Ctrl+E) | ✅ | 704b569 |
| 11 | Integração e2e, CI, README | ✅ | 9d1a7fe |

## Status — Track A pós-MVP (pré-v3)

| # | Task | Status | Notas |
|---|------|--------|-------|
| M0 | Bugfix & hardening | ✅ | ver ROADMAP M0 |
| v2 | Polish (Esc flash, placeholders, create-note, word count) | ✅ | |
| S0 | Selection + clipboard (OSC 52) | ✅ | `7771899` |
| L1 | Lazy-raw (structural gate) | ✅ | `e0ba9f5`, `7a3e5d5` |
| Track B (parcial) | Callouts, highlight, typographer | ✅ | `2bf3ad9`; pendente B3/B4 |

## Status — v3 / S1 (Table Widget) ← **COMPLETO**

> Spec + plan TDD acima. S1 entregue; S2/S3 só como follow-up.

| # | Task | Status | Commit alvo |
|---|------|--------|-------------|
| S1.1 | `doc.ReplaceLines` — mutação atômica de range de linhas + 1 undo | ✅ `2de67a8` | `feat(doc): add ReplaceLines for atomic block rewrites` |
| S1.2 | `internal/blockedit` — interface `Widget` + parse/serialize pipe table | ✅ `09ef3cb` | `feat(blockedit): pipe-table model with parse/serialize round-trip` |
| S1.3 | Table widget — Tab/setas, edição de célula, Esc→Cancel | ✅ `f29efac` | `feat(blockedit): table cell navigation and in-memory editing` |
| S1.4 | Grade: add/del row/col, Ctrl+L align, auto-resize no serialize | ✅ `55420ad` | `feat(blockedit): table row/column ops and alignment cycle` |
| S1.5 | Editor seam: `active` + branch widget em `layout.go`; Table Enter→widget; fence continua lazy-raw; atualizar testes L1 de tabela | ✅ `4278ca8` | `feat(editor): open table widget on edit intent via lazy-raw seam` |
| S1.6 | Commit via `ReplaceLines` / Esc cancel / leave-block commit; undo atômico | ✅ `54df372` | `feat(editor): atomic table commit via ReplaceLines and Esc cancel` |
| S1.7 | `BlockEditHint` + statusbar + teatest smoke | ✅ `ccc7db5` | `feat(ui): table-widget status hint and smoke teatest` |
| S1.8 | Gate QA + marcar status nesta tabela | ✅ | `docs(tasks): mark v3/S1 table widget complete` |

### Aceite S1 (checklist — espelho do plan)

- [x] Tabela sob cursor fica renderizada até Enter / 1º rune.
- [x] Enter abre widget (não raw); não insere `\n`; `Version` intacto até Commit.
- [x] Editar célula + sair do bloco → markdown atualizado; um `^Z` desfaz o bloco inteiro.
- [x] Esc descarta; doc idêntico ao pré-open.
- [x] Add coluna reflete no source após Commit.
- [x] Code fence ainda lazy-raw.
- [x] Malformed table → fallback raw.
- [x] `CGO_ENABLED=0 go test ./...` + vet + gofmt limpos.

### Como executar (agente)

1. Ler o **spec** completo, depois a task corrente no **plan** (passos `- [ ]`).
2. Branch: `feat/v3-table-widget` (ou continuar numa `feat/*` existente alinhada a S1).
3. TDD: testes → FAIL → código → PASS → commit convencional da task.
4. Não implementar S2/S3; não reabrir arquitetura (editor-centric / mesmo seam / ReplaceLines).
5. Ao terminar S1.8: PR → `stable`.

### Follow-ups (NÃO fazer em S1)

| # | Item | Status |
|---|------|--------|
| S2 | Padrão reusável: code fence / links / callouts widgets | ⏳ depois de S1 |
| S3 | In-doc find (`^F`) | ⏳ depois de S1 |
| B3/B4 | Footnotes / definition lists (Track B) | ⏳ paralelo ok |

## Contratos entre pacotes (NÃO quebrar)

- `doc`: `Position{Line,Col}`; `Insert/DeleteRange/DeleteBackward/DeleteForward` → Position; `Undo/Redo() (Position, bool)`; `Version()`; `Save` / `SaveForce`; **novo em S1.1:** `ReplaceLines(start, end int, lines []string) Position` (1 patch de undo, não coalescible).
- `vault`: `Open` / `List` / `Resolve` / `Rescan` (inalterado em S1).
- `mdparse`: `Parse` → `[]Block` cobertura total; `Kind` inclui `Table`; `WikiLinkAt` (inalterado em S1).
- `render`: `Block(…)` determinístico; tabelas via `renderTable` (widget **não** precisa chamar render — View própria em `blockedit`).
- `editor`: `New` / `Update` / `View` / `SetSize` / `Cursor` / `Doc` / `SetDoc`; msgs `FollowLinkMsg` / `AutocompleteMsg`; **novo:** `active blockedit.Widget`, `BlockEditHint() string`; lazy-raw (`editing`) permanece para fence/code.
- `blockedit` (**novo**): `Widget` + `OpenTable` + `Signal{Continue,Commit,Cancel}`.

## Keybindings relevantes (S1)

| Contexto | Tecla | Ação |
|----------|-------|------|
| Tabela renderizada | Enter / 1º rune | Abre table widget |
| Widget | Tab / S-Tab / setas | Navega células |
| Widget | Esc | Cancel (sem Write) |
| Widget | Ctrl+Shift+↑↓←→ | Del/add row/col |
| Widget | Ctrl+L | Cicla alinhamento da coluna |
| Widget ativo + sair do bloco | ↑/↓ além da borda | Commit + move |
| Fence (inalterado) | Enter / 1º rune | Lazy-raw |

## Fora de escopo (reafirmado)

Preview pane, line numbers, vim modes, mouse, imagens, math, temas por arquivo de config, SQLite no vault — ver `docs/ROADMAP.md` Non-goals.
