# mdit — TASKS (handoff para qualquer agente: Cursor, Claude Code, etc.)

> Fonte de verdade detalhada: `docs/superpowers/plans/2026-07-16-mdit-mvp.md` (plano completo, passos TDD, código) e `docs/superpowers/specs/2026-07-16-mdit-mvp-design.md` (design/decisões).
> Progresso corrente: `.superpowers/sdd/progress.md` (ledger) + relatórios/reviews por task em `.superpowers/sdd/task-N-{brief,report,review}.md`.
> Branch de trabalho: `feat/mvp`. Processo: TDD por task → commit convencional → code review → QA (`go test ./... && go vet ./... && gofmt -l .` limpos).

## Critérios de aceite globais (valem para TODAS as tasks)

- `go build ./...`, `go vet ./...`, `go test ./...` verdes; `gofmt -l .` vazio; CGO_ENABLED=0.
- Cursor sempre em coordenadas cruas (linha, coluna-em-runes); tela é derivada.
- Keybindings: Ctrl+S salvar, Ctrl+Q sair, Ctrl+P finder, Ctrl+E zen, Ctrl+] seguir link, Ctrl+B voltar, Ctrl+Z/Y undo/redo.
- Mensagens de UI em inglês. Commits conventional commits.

## Status

| # | Task | Status | Commits |
|---|------|--------|---------|
| 1 | Scaffold (go.mod, LICENSE MIT, README, main --version) | ✅ feito + review | 1640f45, ae8c9cb |
| 2 | `internal/doc` — buffer de linhas, Insert/DeleteRange rune-aware, undo/redo coalescido (500ms, clock injetável), Save com ErrExternalChange (mtime) | ✅ feito + review | c6ecd22 |
| 3 | `internal/vault` — índice nome→path, Resolve case-insensitive (empate: path mais curto, depois lexicográfico), List estável, paths absolutos, root oculto ok | ✅ feito + review | 839e0a6, e4eb8f7 |
| 4 | `internal/mdparse` — goldmark+GFM → `[]Block` cobrindo TODAS as linhas (invariante), extensão wikilink `[[t]]`/`[[t\|alias]]`, `WikiLinkAt`, frontmatter, setext | ✅ feito + review | 53af578, 65a8ef6 |
| 5 | `internal/theme` + `internal/render` — bloco → linhas estilizadas ≤ Width, chroma, wrap ANSI-safe, wikilink broken style | ✅ implementado; ⏳ review pendente | 084878e |
| 6 | `internal/editor` — widget virtualizado (prefix sums), bloco-sob-cursor cru, teclas de edição, undo | ⬜ | — |
| 7 | `internal/ui` + `cmd/mdit` — app rodável, statusbar, prompts (dirty/conflito), panic recover → `.mdit-recover` | ⬜ | — |
| 8 | Fuzzy finder (Ctrl+P, overlay bubbles/list) | ⬜ | — |
| 9 | Wikilinks: Ctrl+] segue, Ctrl+B volta (pilha), autocomplete popup em `[[`, broken em vermelho | ⬜ | — |
| 10 | Zen mode (Ctrl+E, read-only, centrado ≤80 cols, preserva cursor/scroll ao voltar) | ⬜ | — |
| 11 | Integração e2e (teatest), CI GitHub Actions (3 OS), .golangci.yml, README final | ⬜ | — |

## Contratos entre pacotes (NÃO quebrar)

- `doc`: `Position{Line,Col}`; `Insert/DeleteRange/DeleteBackward/DeleteForward` retornam Position; `Undo/Redo() (Position, bool)`; `Version()` incrementa a cada mutação (chave de cache); `Save() error` (→ `doc.ErrExternalChange`), `SaveForce()`.
- `vault`: `Open(root) (*Vault, error)`; `List() []Note{Name,Path}`; `Resolve(target string) (string, bool)`; `Rescan() error`.
- `mdparse`: `Parse(lines []string) Result{Blocks []Block, Source []byte}`; `Block{Kind, Start, End, Node}`; blocos cobrem [0,n-1] contíguos sem overlap; segmentos do Node indexam `Source` (alinhado); `WikiLinkAt(line string, col int) (string, bool)` col em runes; node `WikiLink{Target, Alias}.Label()`.
- `render`: `Block(res mdparse.Result, i int, ctx Context{Width, Theme, IsBroken}) []string`; toda linha ≤ Width células imprimíveis; `len(out) ≥ 1`; determinístico por (conteúdo, Width); IsBroken nil-safe.
- `theme`: `DefaultDark() Theme` (campos lipgloss por elemento).

## Tasks restantes — o que fazer + critérios de aceite

### Task 5 — REVIEW pendente
Implementação commitada (084878e). Falta: code review (spec compliance + qualidade) e correções se houver Critical/Important. Aceite: review aprovado; invariantes de largura/altura testados.

### Task 6 — `internal/editor` (núcleo do produto)
**Como:** `Model` com doc + `mdparse.Result` (reparse quando `doc.Version()` muda) + cache de render por (hash do texto do bloco, width) + prefix sums de alturas → visível por busca binária. Bloco contendo cursor renderiza CRU (todas as suas linhas, estilo RawBlock, soft-wrap). `View()` materializa só blocos que intersectam a janela. Cursor de tela: prefix sum + linhas wrapped dentro do bloco cru (runewidth por rune). Teclas: runas, enter, backspace/delete, setas, ctrl+←/→, home/end, pgup/pgdn, ctrl+z/y. Emite `FollowLinkMsg{Target}` (Ctrl+]) e `AutocompleteMsg{Query}` (ao digitar `[[`).
**Aceite:**
- Cursor em QUALQUER linha de tabela/fence → bloco INTEIRO cru; sai → renderizado de novo.
- Doc de 100+ blocos: `View()` retorna exatamente `height` linhas; blocos fora da janela nunca são renderizados (virtualização real — cache miss count testável ou instrumentação simples).
- Scroll segue cursor (nunca sai da janela); prefix sums = soma das alturas.
- Digitar reflete na View; undo restaura View + cursor.
- Testes de mapping pesados (maior risco do projeto): setas cruzando bloco wrapped, backspace unindo linhas, edição em bloco multi-linha.
- Commit: `feat(editor): virtualized inline-render editor widget`.

### Task 7 — `internal/ui` + `cmd/mdit`
**Como:** `App` tea.Model raiz (bubbletea v1, WithAltScreen): modos edit|prompt; statusbar (arquivo, dirty `[+]`, linha:col, hints); Ctrl+S (ErrExternalChange → prompt [o]verwrite/[r]eload/[c]ancel); Ctrl+Q (dirty → [s]ave/[d]iscard/[c]ancel); main.go: arg arquivo→vault=dir do arquivo; dir→vault=dir; panic guard: defer recover → grava `<path>.mdit-recover` + restaura terminal.
**Aceite:** teatest: abrir doc → heading renderizado; digitar+Ctrl+S → conteúdo em disco; Ctrl+Q dirty → prompt → `d` sai sem salvar. QA manual: rodar no terminal real (tmux), digitar/salvar/sair. Commit: `feat(ui): runnable editor app with statusbar and prompts`.

### Task 8 — Fuzzy finder
**Como:** estado modeFinder; overlay centrado com `bubbles/list` (filtro fuzzy ativo) sobre `vault.List()`; Enter abre (prompt se dirty), Esc fecha.
**Aceite:** teatest: Ctrl+P lista notas; filtro reduz; Enter troca doc (statusbar); Esc volta intacto. Commit: `feat(ui): fuzzy note finder`.

### Task 9 — Wikilinks completos
**Como:** Ctrl+] → `mdparse.WikiLinkAt(linha crua, col)` → `vault.Resolve` → abre (dirty prompt) e push no histórico; falha → statusbar "broken link: X". Ctrl+B pop histórico. Autocomplete: `[[` digitado → popup ancorado no cursor filtrando notas a cada rune; Enter insere `nome]]`; Esc mantém texto. Broken render: `IsBroken` do editor = `!vault.Resolve`.
**Aceite:** teatest com vault fixture (a.md com `[[b]]`+`[[nope]]`, b.md): Ctrl+] em `[[b]]` abre b.md; Ctrl+B volta; `[[nope]]` → aviso statusbar; `[[` → popup, filtrar `b`, Enter → `[[b]]` no texto. Commit: `feat(wikilinks): follow, back, autocomplete, broken-link highlight`.

### Task 10 — Zen mode
**Como:** modeZen: layout do editor com cursorBlock=-1 (nada cru, reusa cache), coluna centrada `min(width,80)`, scroll ↑/↓/PgUp/PgDn/Home/End, edição ignorada, statusbar mínima (`zen │ ^E back`). Ctrl+E alterna preservando cursor/scroll.
**Aceite:** teatest: Ctrl+E → bloco do cursor renderizado (não cru), linhas ≤80 e centradas; digitar não muda `doc.Version()`; Ctrl+E volta com bloco cru de novo. Commit: `feat(zen): read-only rendered mode`.

### Task 11 — Integração, CI, README
**Como:** teste e2e teatest (abrir vault → editar → autocomplete → seguir link → voltar → zen → salvar+sair); `.github/workflows/ci.yml` (matrix ubuntu/macos/windows, Go 1.26, CGO_ENABLED=0, test + golangci-lint em job linux); `.golangci.yml` (padrão + errcheck + staticcheck); README: features, install `go install github.com/carvalhosauro/mdit/cmd/mdit@latest`, keybindings, licença.
**Aceite:** e2e verde nos 3 OS via CI; README cobre tudo acima. Commit: `chore: integration test, ci, readme`.

## Pós-MVP (fora de escopo agora — NÃO implementar)

Temas configuráveis, busca full-text/incremental, backlinks, criar nota ao seguir link quebrado, tabs, IA, export HTML, vim mode, mouse, line numbers, parse incremental, "lazy raw" de tabelas, SQLite no vault.

## Pendências conhecidas (minors de review, endereçar na review final / Task 11)

- doc: métodos de edição panicam em posição inválida (sem clamp); `Dirty()` monotônico (undo até estado salvo continua dirty — UI pode over-prompt); arquivo novo vazio salva `"\n"`; guard redundante hasPath.
- vault: trim de extensão só `.md`/`.MD`; subdir ilegível aborta scan (fail-fast).
- mdparse: `indexByte` duplica `bytes.IndexByte`; `close` shadowing; indented code vira Kind CodeFence (misnomer); WikiLinkAt regex vs parser inline têm aceitação levemente diferente em edges.
