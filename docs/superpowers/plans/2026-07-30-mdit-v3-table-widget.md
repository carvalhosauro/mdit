# mdit v3 / S1 — Table Widget Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bloco `Kind Table` sob o cursor abre um widget estruturado (células, grade, alinhamento) no seam do lazy-raw; commit atômico via `doc.ReplaceLines`; Esc cancela sem mutar o doc.

**Architecture:** Editor-centric (ROADMAP opção B). `editor.Model` segura `active blockedit.Widget` (nil no normal). Seam em `layout.go`: rendered | raw | widget. Lógica de tabela em `internal/blockedit` (pura). Durante o widget nada escreve no `doc`; ao Commit, um `ReplaceLines` = um undo.

**Tech Stack:** Go ≥1.26, bubbletea v1, lipgloss, goldmark GFM (`east.Table` só no render existente), go-runewidth, teatest onde couber. `CGO_ENABLED=0`.

**Spec:** `docs/superpowers/specs/2026-07-30-mdit-v3-table-widget-design.md` — leia antes de qualquer task.

**Handoff:** `TASKS.md` seção **v3 / S1**.

**Release train:** PRs `feat/*` → `stable` (`docs/RELEASE_FLOW.md`).

## Global Constraints

- Module: `github.com/carvalhosauro/mdit`. `CGO_ENABLED=0`.
- Todo commit: `gofmt` limpo, `go vet ./...` limpo, `go test ./...` verde.
- Conventional commits (`feat:`, `test:`, `fix:`, `chore:`, `docs:`).
- Cursor SEMPRE em coordenadas cruas `(linha, col-em-runes)` **fora** do widget; dentro do widget o foco é `(row,col)` de célula — reconciliar no exit.
- UI strings em **inglês**.
- **Não** implementar S2/S3 neste plan.
- **Não** reabrir arquitetura App-centric.
- TDD: teste → FAIL → implementa → PASS → commit por task (ou sub-commit se a task pedir).
- Atualizar testes L1 de **tabela** que esperam raw após Enter (passam a esperar widget).

## Critérios de aceite (S1 — todos testáveis)

1. Cursor entra em tabela → permanece **renderizada** (lazy) até intent.
2. `Enter` em tabela → widget ativo (`active != nil`); **não** raw; **não** insere `\n` no doc.
3. 1º caractere em tabela renderizada → abre widget e aplica o rune na célula focada (doc ainda intocado até Commit).
4. Tab / setas movem o foco de célula; editar muda só a célula focada no modelo em memória.
5. Add coluna (Ctrl+Shift+→) reflete no markdown **após** sair com Commit.
6. `Esc` → widget some, tabela renderizada, `doc.Version()` **igual** ao pré-open, cursor permanece no bloco.
7. Sair do bloco por ↑/↓ → Commit + `ReplaceLines` uma vez; um `^Z` restaura o bloco anterior inteiro.
8. Code fence continua lazy-raw (Enter → raw source).
9. Malformed table (`OpenTable` falha) → fallback lazy-raw.
10. Gate: `CGO_ENABLED=0 go test ./... && go vet ./...` ; `gofmt -l .` vazio.

---

### Task S1.1: `doc.ReplaceLines` — mutação atômica de bloco

**Files:**
- Modify: `internal/doc/doc.go` (e `undo.go` se `recordEdit` precisar de flag non-coalesce explícita)
- Test: `internal/doc/doc_test.go` e/ou `internal/doc/replace_lines_test.go`

**Interfaces:**

```go
// ReplaceLines replaces inclusive lines [start, end] with lines.
// A single undo patch; Version increments once. Not coalescible with typing.
func (d *Document) ReplaceLines(start, end int, lines []string) Position
```

- [ ] **Step 1:** Escrever testes:
  - Replace 1 linha por 1 (conteúdo muda; LineCount igual).
  - Replace 3 linhas por 5 (crescimento); por 1 (encolhimento); por `nil`/empty (delete range — documentar: `lines == nil` tratado como empty).
  - Range no fim do doc; `start==end`; unicode nas linhas.
  - `Version` +1 exatamente.
  - Undo restaura `Content()` e cursor do patch; redo reaplica; nova edit limpa redo.
  - Clamp: `start`/`end` fora dos bounds não panicam (espelhar política de `applyReplace` pós-M0 se já houver clamp; senão clamp defensivo).
- [ ] **Step 2:** `go test ./internal/doc/ -run ReplaceLines` → FAIL.
- [ ] **Step 3:** Implementar via um `recordEdit` multi-linha (preferir reusar `applyReplace` cobrindo `[start,0]`…`[end, runeLen(line)]` com `strings.Join(lines, "\n")`, garantindo `coalescible=false`).
- [ ] **Step 4:** PASS. Commit:

```text
feat(doc): add ReplaceLines for atomic block rewrites
```

---

### Task S1.2: `internal/blockedit` — parse/serialize de pipe table

**Files:**
- Create: `internal/blockedit/blockedit.go` (Signal + Widget interface)
- Create: `internal/blockedit/table.go` (modelo + OpenTable + CommitLines)
- Test: `internal/blockedit/table_test.go`

**Interfaces:** (copiar do spec; não exportar tipos de editor)

```go
type Signal int
const ( Continue Signal = iota; Commit; Cancel )

type Widget interface {
    Update(msg tea.Msg) (Widget, tea.Cmd, Signal)
    Lines(width int) []string
    CommitLines() []string
    ExitCursor(sig Signal) doc.Position
}

func OpenTable(rawLines []string, cursor doc.Position, blockStart int) (Widget, bool)
```

Nesta task: **só** parse/serialize + `OpenTable` + `CommitLines` + `ExitCursor` stub; `Update`/`Lines` podem ser mínimos (Lines = join visual simples ou TODO coberto na S1.3). Preferível: nesta task `Update` retorna `Continue` sempre e `Lines` retorna um placeholder — desde que testes de round-trip passem.

- [ ] **Step 1:** Testes table-driven de round-trip:
  - Header + separator + 2 body rows; aligns left/center/right/none.
  - Células com espaços trimados na borda.
  - `OpenTable` em input sem separator → `ok=false`.
  - `CommitLines` auto-resize: coluna com "hello" → separator com ≥5 `-` (respeitando markers `:`).
- [ ] **Step 2:** FAIL → implementar parser/serializer GFM pipe (sem goldmark obrigatório no pacote; regex/split por `|` é ok se golden cases cobrirem).
- [ ] **Step 3:** PASS. Commit:

```text
feat(blockedit): pipe-table model with parse/serialize round-trip
```

---

### Task S1.3: Table widget — navegação e edição de célula

**Files:**
- Modify: `internal/blockedit/table.go` (+ `table_view.go` se preferir separar View)
- Test: `internal/blockedit/table_keys_test.go`

**Comportamento (spec):**
- Tab / Shift+Tab; setas com borda→célula vizinha; runas editam; Backspace/Delete; Enter → célula abaixo.
- Esc → `Signal=Cancel`; não há “Commit key” dedicada no widget (Commit vem do editor ao sair).
- Opcional nesta task: expor método de teste `Focus() (row, col int)` ou inspecionar via `CommitLines` após edits.

- [ ] **Step 1:** Testes com `tea.KeyMsg`:
  - Open fixture 2×2 → focus inicia em célula sob `cursor` (best-effort) ou (0,0).
  - Tab cicla 4 células; Shift+Tab volta.
  - Digitar `"Hi"` na célula → `CommitLines` contém `Hi` só naquela célula.
  - Esc → `Cancel`.
  - Enter na row 0 → focus body row mesma col.
- [ ] **Step 2:** FAIL → implementar `Update` + estado `focus`/`cellCol`.
- [ ] **Step 3:** Testes de `Lines(width)`: contém texto das células; célula focada distinta (após `ansi.Strip`, ainda deve aparecer o texto; opcionalmente assertar sequência ANSI de reverse se estável).
- [ ] **Step 4:** PASS. Commit:

```text
feat(blockedit): table cell navigation and in-memory editing
```

---

### Task S1.4: Operações de grade (row/col/align/resize)

**Files:**
- Modify: `internal/blockedit/table.go`
- Test: `internal/blockedit/table_grid_test.go`

- [ ] **Step 1:** Testes:
  - Ctrl+Shift+Down → +1 body row (células vazias); CommitLines tem +1 linha `| … |`.
  - Ctrl+Shift+Up em body → remove row (ou limpa se última); header nunca removido.
  - Ctrl+Shift+Right → +1 coluna em header/sep/body; Left apaga (mín. 1 col).
  - Ctrl+L cicla align da coluna; CommitLines separator reflete `:---` / `:---:` / `---:`.
  - Auto-resize já coberto em S1.2 — acrescentar caso pós-insert coluna.
- [ ] **Step 2:** FAIL → implementar.
- [ ] **Step 3:** PASS. Commit:

```text
feat(blockedit): table row/column ops and alignment cycle
```

---

### Task S1.5: Editor seam — `active` widget + ativação Table

**Files:**
- Modify: `internal/editor/editor.go` (campo `active blockedit.Widget`)
- Modify: `internal/editor/layout.go` (branch widget no seam)
- Modify: `internal/editor/keys.go` / `lazy_raw.go` (abrir widget em Table; fence inalterado)
- Modify: `internal/editor/lazy_raw_test.go` (expectations de tabela)
- Test: `internal/editor/table_widget_test.go` (novo)

**Contratos:**

```go
// editor.Model gains (private):
//   active blockedit.Widget // nil = no structured edit

// SetDoc / leaving zen clears active.
// BlockEditHint() string  // may land in S1.7; stub ok here
```

Lógica de abertura:
- `shouldOpenTableWidget()` ≈ `shouldLazyActivate() && kind==Table`
- Enter: se table → `OpenTable(raw lines, cursor, start)`; se !ok → `activateEditing()`; senão `active=w`
- 1º rune: se table → open widget **depois** aplicar rune via `Update(KeyRunes)` (doc intocado)
- Se `active != nil`, `handleKey` delega 100% ao widget (exceto atalhos globais listados no spec — podem ficar para S1.6/S1.7)

- [ ] **Step 1:** Atualizar `TestLazyRaw_TableRenderedUntilEnter` e `FirstRune*` / `Esc*` de tabela:
  - Enter → **não** `layouts[i].raw`; assertar via helper `m.active != nil` (exportar `testing` helper no pacote editor: `func (m Model) testActive() bool` ou inspecionar Lines do layout sem pipes crus `| A | B |` se a View do widget não usa pipes).
  - Preferir helper de teste no mesmo pacote: `func (m Model) hasBlockEdit() bool { return m.active != nil }`.
- [ ] **Step 2:** Novos testes:
  - Fence: Enter ainda raw (regression L1).
  - Tabela: Enter → hasBlockEdit; doc.Version inalterado; Lines do bloco não são source pipes.
- [ ] **Step 3:** FAIL → wiring seam + keys → PASS.
- [ ] **Step 4:** Commit:

```text
feat(editor): open table widget on edit intent via lazy-raw seam
```

---

### Task S1.6: Commit / Cancel / leave-block + undo atômico

**Files:**
- Modify: `internal/editor/keys.go`, `layout.go` (leave block)
- Test: `internal/editor/table_widget_test.go`

- [ ] **Step 1:** Testes:
  - Open → edit célula → Esc → `Version` igual; Content igual; `!hasBlockEdit`; tabela renderizada.
  - Open → edit → ↑ para fora do bloco → `Version+1`; Content com nova célula; um Undo restaura Content pré-edit; `!hasBlockEdit`.
  - Open → edit → ↓ para fora → idem Commit.
  - ReplaceLines chamado uma vez por saída (assert Version +1, não +N por tecla).
- [ ] **Step 2:** Implementar `finishBlockEdit(sig)`:
  - Cancel: `active=nil`; cursor = `ExitCursor(Cancel)` (pré-open salvo no editor ao abrir).
  - Commit: capturar `start,end` do bloco **antes**; `lines := active.CommitLines()`; `active=nil`; `cursor = doc.ReplaceLines(...)` ajustado com `ExitCursor(Commit)`; `recompute`.
  - Ao mudar `cursorBlock` por motion **com** active: Commit antes de aplicar motion (ou motion que cruzaria borda).
- [ ] **Step 3:** Guardar `blockStart/blockEnd` e `cursorBeforeOpen` no editor ao abrir (campos privados).
- [ ] **Step 4:** PASS. Commit:

```text
feat(editor): atomic table commit via ReplaceLines and Esc cancel
```

---

### Task S1.7: Statusbar hint + teatest smoke + help drift check

**Files:**
- Modify: `internal/editor/editor.go` (`BlockEditHint() string`)
- Modify: `internal/ui/statusbar.go` (ou equivalente) para mostrar hint
- Test: `internal/ui/*_test.go` e/ou `internal/editor` teatest; smoke em pacote ui se já houver padrão

- [ ] **Step 1:** `BlockEditHint()` retorna string inglesa não vazia só com table widget ativo.
- [ ] **Step 2:** teatest (ou unit View): open table widget → statusbar contém `table` / `esc cancel`.
- [ ] **Step 3:** Smoke: fixture doc com tabela → Enter → type → leave → releitura de `Doc().Lines()` com coluna nova se o smoke cobrir add-col (mínimo: edit célula + leave).
- [ ] **Step 4:** `go test ./...` verde. Commit:

```text
feat(ui): table-widget status hint and smoke teatest
```

---

### Task S1.8: QA gate + handoff

**Files:**
- Modify: `TASKS.md` (marcar S1.x ✅ + hashes de commit)
- Opcional: nota curta em `docs/ROADMAP.md` que S1 está implementado (só se o autor pedir; default = só TASKS)

- [ ] **Step 1:** Rodar gate completo:

```bash
CGO_ENABLED=0 go test ./...
go vet ./...
test -z "$(gofmt -l .)"
```

- [ ] **Step 2:** Atualizar tabela de status em `TASKS.md`.
- [ ] **Step 3:** Commit se houver só docs:

```text
docs(tasks): mark v3/S1 table widget complete
```

---

## Ordem de execução (resumo)

1. **S1.1** `ReplaceLines` — fundação (começar aqui)
2. **S1.2** parse/serialize `blockedit`
3. **S1.3** keys de célula
4. **S1.4** grade/align
5. **S1.5** seam editor
6. **S1.6** commit/cancel/undo
7. **S1.7** hint + teatest
8. **S1.8** gate + TASKS

## Fora deste plan

- S2 widgets (fence language, links, callouts)
- S3 in-doc find
- Raw escape hatch para tabelas bem-formadas
- Mudança de release train / CGO / seleção OSC 52
