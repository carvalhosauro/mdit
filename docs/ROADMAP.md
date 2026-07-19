# mdit — ROADMAP (pós-MVP)

> Estado do MVP: `TASKS.md` (tasks 1–11 ✅). Este doc cobre **o que vem depois**:
> polish, breadth de markdown e edição estruturada assistida.
> **DRAFT em refino** — nada aqui é para implementar antes de aprovado.
> Fonte de design do MVP: `docs/superpowers/specs/2026-07-16-mdit-mvp-design.md`.

## Como ler

Três milestones incrementais (**v2 → v2.5 → v3**), ordenados por valor/esforço.
Cada item tem **escopo** + **critérios de aceite** testáveis. Sizes: `S` (horas),
`M` (1–2 dias), `L` (dias / arquitetural).

Gate de qualidade (vale para TODO item, herdado do MVP):
`go build ./...`, `go vet ./...`, `go test ./...` verdes; `gofmt -l .` vazio;
`CGO_ENABLED=0`. Mensagens de UI em inglês. Commits convencionais.

## Princípios (guardrails — todo item é checado contra isto)

1. **WYSIWYG inline, sem preview pane.** Bloco renderizado no lugar; bloco sob o
   cursor é editável. Nada que reintroduza "modo de leitura separado".
2. **Sem clutter.** Cada elemento de chrome (statusbar, gutter, overlay) precisa
   justificar seu espaço. Default limpo.
3. **Modeless.** Sem modos vim (normal/insert). Estado de UI (edit/prompt/finder/
   zen/help) ≠ modos de edição.
4. **Single binary, zero config.** `CGO_ENABLED=0`, sem arquivo de config
   obrigatório. Features novas não podem exigir setup.
5. **Vault-first / Obsidian-flavored.** Wikilinks e o vault são o diferencial;
   breadth de markdown prioriza o que a comunidade Obsidian usa.
6. **Invariante do cursor:** cursor sempre em coordenadas cruas `(linha, col-em-runes)`;
   a tela é derivada (`TASKS.md:8`). Qualquer editor estruturado (v3) precisa de uma
   camada de estado própria — não pode violar isto no buffer.

---

## Visão geral — dois tracks

M0 primeiro (fundação). Depois o trabalho se divide em **dois tracks que quase não
se cruzam**:

**Track A — modelo de cursor/edição** (sequencial; cada etapa constrói sobre a
anterior, todas tocam `doc.Position`/cursor/`DeleteRange`/seam de edição):
```
M0  →  v2 Polish  →  S0 Selection/clipboard  →  Lazy-raw (bridge)  →  v3 Structured editing
```

**Track B — render/parse** (ortogonal; interleava a qualquer momento, até em paralelo):
```
Breadth: callouts, highlight, footnotes, deflist, typographer
```

Track B só toca `mdparse`/`render`/`theme` — não mexe no cursor, então **não compete**
com o Track A por ordem. `selection` foi puxado pra frente (era 1º item do v3): é win
geral não-gated por v3, e toca o mesmo core que lazy-raw/widgets → fazer uma vez, em
ordem, evita retrabalho no modelo de cursor.

Ordem dentro de cada milestone = **criticidade primeiro**.

---

## M0 — Bugfix & Hardening (fundação)

**Meta:** corrigir a base antes de empilhar features. Todos os bugs abaixo foram
**confirmados com probe-tests em ambiente controlado** (evidência na coluna). v3
(edição estruturada) vai chamar as APIs de `doc` com posições calculadas o tempo
todo → BG1 deixa de ser latente e vira crash frequente; por isso M0 vem antes.

Ordenado por criticidade.

| # | Bug | Ref | Sev | Evidência (probe) | Fix |
|---|-----|-----|:---:|-------------------|-----|
| BG1 | Edições panicam em posição inválida (linha E coluna, sem clamp) | `doc.go:191` `applyReplace`, `:154` `DeleteForward` | **Alta** | `Insert{Line:50}` → `index out of range [50] with length 1`; `Insert{Col:999}` → `slice bounds out of range [:999] cap 32` | clamp em `applyReplace`/`normalizeCursor` no pkg doc (contrato seguro). |
| BG4 | 1 subdir ilegível aborta scan do vault inteiro | `vault.go:98` `scan` (`WalkDir` retorna `err`) | **Alta** | `Open` → `open .../secret: permission denied` (vault não abre) | no callback: `IsPermission(err)` → `SkipDir` + coletar aviso, não abortar. |
| BG2 | `Dirty()` monotônico: undo até estado salvo continua dirty | `doc.go:99` | Med | `content == saved` mas `Dirty()=true` (version=2, savedVersion=0) | comparar por conteúdo/hash salvo, não contador de versão. |
| BG8 | `^C`/SIGINT morto (só `^Q` sai) | `app.go:168-189` (sem `KeyCtrlC`) | Med | code-fact (nenhum handler) | rotear `^C` pelo path de `^Q` (dirty→prompt). |
| BG6 | `WikiLinkAt` (regex) segue link dentro de code span | `wikilink.go:140` vs parser inline | Med | `` `[[note]]` `` → AST=false, regex=**true** (Ctrl+] seguiria link que renderiza como código) | validar contexto (não seguir dentro de code span) OU derivar do AST. |
| BG5 | Só `.md` minúsculo indexado; `.markdown`/`.Md` invisíveis | `vault.go:114` | Med | `foo.markdown`=false, `bar.Md`=false, `baz.md`=true | case-insensitive + set de extensões (`.md`, `.markdown`). |
| BG3 | Arquivo novo vazio salva `"\n"` (1 byte) em vez de vazio | `doc.go:106` `Content()` | Low | disco: `"\n"` len=1 | doc vazio → `Content()` vazio. |
| BG7 | Indented code classificado como `Kind CodeFence` (misnomer) | `parse.go:321` `mapKind` | Low | bloco indentado → kind=3 (CodeFence) | Kind próprio `IndentedCode` ou renomear. |

**Recomendações (qualidade/DX), após os bugs:**

| # | Rec | Ref | Nota |
|---|-----|-----|------|
| RC1 | Gate de qualidade não roda local (lint/hooks) | memória `dx-tooling-verification-gaps` | golangci local é go1.25 < go1.26 alvo; `make tools` instala. Pinar versão + documentar em `docs/DEVELOPMENT.md`. |
| RC2 | Keybindings single-source-of-truth | `app.go:168`, `keys.go:14`, `help.go:12` | tabela única → dispatch + help derivam dela. Fix da classe de drift que gerou BG8. |
| RC3 | Dead code: `hasPath` redundante, `indexByte` dup de `bytes.IndexByte`, `close` shadowing | `doc.go:25`, `wikilink.go:107`, `parse.go` | hygiene. |

**Não-bugs verificados (não agir):** undo/redo — boundaries seguros e redo
corretamente invalidado após nova edição (probe `Explore_UndoRedo`).

**Aceite M0:** cada bug ganha teste de regressão que falha antes / passa depois;
gate (`go test/vet`, `gofmt`) verde; nenhum probe descartável commitado.

---

## v2 — Polish (papercuts)

**Meta:** eliminar atritos que fazem o editor "sentir errado" hoje. Barato,
alto ganho percebido. Sem features novas grandes.

> Movidos para **M0**: `^C`/SIGINT (era P1 → BG8) e keybindings single-source-of-truth
> (era P6 → RC2), por serem bug/hardening.

| # | Item | Size | Escopo | Aceite |
|---|------|:----:|--------|--------|
| P2 | Esc dispensa flash de erro | S | Erro só some por timeout de 5s (`app.go:48,214`). Esc deve limpar o flash na hora. | teatest: flashErr ativo + Esc → statusbar limpa antes dos 5s. |
| P3 | Empty states com dica | S | Doc novo/vazio = linha em branco (`layout.go:154`); vault vazio = título "Notes" pelado (`finder.go:36`). Placeholder dim ("start typing…") no buffer vazio; linha de dica no finder sem notas. | Visual/teatest: buffer vazio mostra placeholder; finder com 0 notas mostra dica. Placeholder some ao 1º caractere. |
| P4 | Broken link → criar nota | S/M | `^]` em link quebrado hoje só avisa na statusbar. Oferecer prompt `[c]reate note?` que cria `<target>.md` **na subpasta da nota atual** (não no root), **sem template** (arquivo vazio), e navega. | teatest: `^]` em `[[nope]]` → prompt; `c` cria arquivo na pasta da nota corrente + abre + push no histórico; `Esc` cancela. |
| P5 | Word count na statusbar | S | Sem contagem de palavras/caracteres (`statusbar.go`). Adicionar word count; cair sob largura estreita como os hints já fazem (`statusbar.go:49-62`). | teatest: statusbar mostra contagem; sob largura pequena, contagem é a 1ª a sumir (depois hints, depois trunca nome). |

**Não incluir em v2:** selection/clipboard, find, save-as (ver v3 / non-goals).

---

## Breadth (Track B — paralelo, ortogonal ao Track A)

> Track B não toca cursor/`doc` — só `mdparse`/`render`/`theme`. Pode ser feito em
> paralelo ou interleaved com o Track A, na ordem que fizer sentido de valor.

**Meta:** aumentar cobertura de markdown com o que é barato e Obsidian-relevante.
Padrão por item: add extensão goldmark (ou parse custom) + 1 estilo no `theme` +
1 case em `render.Block`/`styleInline`. Cobertura atual: `internal/mdparse/parse.go:24-35`
(blocos) e `internal/render/inline.go:35-77` (inline).

| # | Item | Size | Escopo | Aceite |
|---|------|:----:|--------|--------|
| B1 | Callouts `> [!note]` | M | Maior ganho Obsidian. Hoje vira Blockquote genérico. **Parse custom sobre Blockquote** (travado — sem dep nova): regex na 1ª linha do blockquote detecta `> [!tipo] título` → render com ícone/cor por tipo (note/warning/tip/…). | render_test: `> [!warning]\n> x` → estilo de callout warning, não blockquote comum; tipo desconhecido cai em blockquote. |
| B2 | Highlight `==texto==` | S | Mark do Obsidian. Extensão inline. Novo estilo `theme.Highlight`. | render_test: `==x==` → estilo highlight; `=x=` (single) não. |
| B3 | Footnotes `[^1]` | S/M | `extension.Footnote` do goldmark. Render de referência + bloco de definição. | render_test: doc com `[^1]` + definição → referência estilizada + seção de notas. |
| B4 | Definition lists | S | `extension.DefinitionList`. Novo Kind ou sub-render. | render_test: `termo\n: definição` → render de deflist. |
| B5 | Typographer | S | `extension.Typographer` (aspas curvas, em-dash, ellipsis). Só render — **não** altera bytes no disco. | render_test: `"x"` renderiza com aspas curvas; `Save` mantém ASCII original. |

**Nota de invariante:** B5 é puramente visual. Nenhum item do Track B pode reescrever
o buffer — render deriva da fonte, fonte intacta.

**Fora (limite de terminal, ver non-goals):** imagens, math/LaTeX.

---

## S0 — Selection & clipboard (Track A)

**Meta:** o maior surface de editor faltando. Win geral (não-gated por v3) e pré-req
do lazy-raw/widgets — por isso puxado pra frente (era 1º item do v3). `doc.Position`
é ponto único hoje; sem shift-keys, sem yank/paste (`DeleteRange` existe, só interno).

| # | Item | Size | Escopo | Aceite |
|---|------|:----:|--------|--------|
| S0a | Seleção interna + copy/paste | M | Âncora de seleção no editor (range, não ponto); shift+setas / shift+word / shift+Home/End; delete de seleção via `DeleteRange`; register interno (yank/paste dentro do app). | teatest: shift+seta marca range; copy+paste round-trip via register; delete de seleção remove o range certo. |
| S0b | Clipboard do SO | S/M | **OSC 52** (travado): cross-platform, zero-dep, funciona sobre SSH. Habilitar bracketed-paste (`main.go:56` hoje não liga). Register interno segue funcionando se OSC 52 for bloqueado. | manual/qa: copy num terminal com OSC 52 → cola noutro app; paste via bracketed-paste entra como texto, não comandos. |

**Invariante:** seleção é estado do **editor**, não do `doc` — `doc` continua com
`Position` ponto. Selection estende o modelo de cursor que lazy-raw e widgets herdam.

---

## Lazy-raw (Track A — bridge p/ v3)

**Meta (ponto 1 travado):** bloco estrutural não vira cru só por ter cursor dentro —
fica renderizado até intenção de editar. É o gate de ativação que o v3 reusa.

**Política:** só **estrutural** (tabela/code/callout) é lazy; **texto** (parágrafo/
heading/lista/quote) fica eager (digitar edita inline = o WYSIWYG). Ativa cru por
`Enter` **ou** 1º caractere; `Esc`/sair do bloco → renderizado.

**Seam:** `layout.go:93` — `raw := !zen && i==cursorBlock` vira
`raw := !zen && i==cursorBlock && (isText(kind) || editing)`. Bit `editing` no editor.

**Não é throwaway:** gate + render-under-cursor sobrevivem 100% no v3 (só troca o
branch cru pelo widget). Descarte real ≈ colocação boba do cursor (~5 linhas).

| # | Item | Size | Escopo | Aceite |
|---|------|:----:|--------|--------|
| L1 | Gate de ativação | S/M | Classificar estrutural; bit `editing`; intenção (Enter/1º-char) → cru; sair/Esc → render; condição no `:93`. | teatest: cursor entra em tabela → renderizada; `Enter` → crua; edita; sair → renderizada; texto continua eager. |

---

## v3 — Assisted structured editing (arquitetural)

**Meta (teu ponto):** bloco-sob-cursor vira **widget estruturado** em vez de raw
text. Extensão natural do modelo atual ("bloco sob cursor = editável"): troca
`raw → widget` para kinds que se beneficiam. Assistência no estilo do autocomplete
de notas, mas para componentes.

**Arquitetura (Q1 travado — opção B, editor-centric):**
- Editor segura `active blockedit.Widget` (nil no normal). O widget é o **3º valor**
  do estado do cursorBlock (`raw` / `rendered` / `widget`) no MESMO seam do lazy-raw
  (`layout.go:93`). Editor delega `Update`/`View` ao widget; a lógica pesada mora em
  pacote novo `internal/blockedit` (interface pura, testável sem TUI).
- **Cache:** widget = branch não-cacheada do cursorBlock, igual `rawBlockLines` já é
  (só `renderedLines` é cacheado, `layout.go:126`). Nada de máquina de cache nova.
- **Commit atômico:** ao sair, `widget.Commit()` → 1 mutação `doc.ReplaceLines(start,
  end, lines)` → bump de Version → recompute reparsa/recacheia. Durante a edição nada
  é escrito no doc → 1 grupo de undo por edição, cancel (Esc) trivial, sem reparse por
  tecla. Cursor cru reposto best-effort ao sair (Princípio 6).
- **Recurso do App** (ex: autocomplete de nota no editor de link) via canal de msg
  editor→App que já existe (`AutocompleteMsg`/`FollowLinkMsg`). Editor fica fino.
- Migração B→C, se um dia o v3 virar App-heavy, é rewiring de posse localizado —
  `internal/blockedit` não muda. Baixo arrependimento.

**Pré-requisitos já entregues no Track A:** selection/clipboard (S0). Falta só o
in-doc find (S3), que fica aqui.

**Artefatos novos:** pacote `internal/blockedit` (interface `Widget` + widgets por
Kind); `doc.ReplaceLines(start, end, []string)` (mutação de bloco atômica, também
usada no commit do lazy-raw); campo `editor.active` + delegação no seam `:93`.

| # | Item | Size | Escopo | Aceite |
|---|------|:----:|--------|--------|
| S1 | Editor de tabela | L | Kind Table sob cursor → widget: navega célula (Tab/setas), edita célula isolada, add/del linha-coluna, alinhamento, auto-resize. Commit reescreve o bloco cru. | teatest: entra na tabela → seleção de célula; editar célula muda só aquela; add coluna reflete no markdown ao sair; `Esc` sai preservando cursor cru. |
| S2 | Padrão reusável p/ outros kinds | M | Generalizar S1: code fence (picker de linguagem), links (editar target/alias), callouts (trocar tipo). | cada kind tem entrada/saída de widget testada; sair sempre restaura cursor cru. |
| S3 | In-doc find (`^F`) | M | Busca dentro do doc (além do note finder). Highlight de matches, next/prev. | teatest: `^F` + query → matches destacados; `Enter`/`n` navega; `Esc` fecha. |

**Marcado como pós-MVP explícito** em `TASKS.md:70` (selection, tabs, etc.) — v3 é
onde parte disso entra, de forma alinhada aos princípios.

---

## Non-goals (fora de escopo — com justificativa)

| Item | Por quê |
|------|---------|
| Preview pane / modo leitura separado | Contradiz o pitch central ("no preview pane", Princípio 1). |
| Line numbers / gutter | Clutter; contra a estética limpa WYSIWYG (Princípio 2). `TASKS.md:70`. |
| Vim modes | Modeless por design (Princípio 3). |
| Mouse | Fora de escopo MVP; reavaliar só se pedido real aparecer. |
| Imagens inline | Precisa protocolo de terminal (kitty/iterm/sixel); frágil, não-portável. |
| Math / LaTeX | Render de LaTeX em terminal é ruim; alto custo, baixo retorno. |
| Temas configuráveis (arquivo de config) | Zero-config é princípio (4). Adaptive light/dark automático já cobre o comum. |
| SQLite / índice no vault | Vault é folder de markdown; sem estado oculto (Princípio 4/5). |

---

## Decisões travadas

- **Lazy-raw + ativação (ex-Q2):** lazy só para blocos **estruturais** (tabela/code/
  callout); texto (parágrafo/heading/lista/quote) fica eager. Ativa raw por `Enter`
  **ou** 1º caractere. É o gate de ativação do v3, entregue cedo (v2.7) como valor
  próprio; scaffold reusado pelo v3, não throwaway (descarte real ≈ colocação boba do
  cursor, ~5 linhas).
- **Camada de estado do widget (ex-Q1): opção B** (editor-centric). Detalhes na seção
  v3. Escolhido por fit com a separação atual (editar buffer = editor) e por ser
  baixo-arrependimento (migra pra C barato se preciso).
- **Ordenação (ex-Q4): dois tracks.** Track A (cursor/edição) sequencial:
  `M0 → v2 → S0 selection → lazy-raw → v3`. Track B (render/parse) ortogonal, paralelo.
  `selection` puxado pra frente (era 1º item do v3): win geral + toca o mesmo core que
  lazy-raw/widgets, então fazer uma vez em ordem evita retrabalho.
- **Clipboard: OSC 52** (não shell-out). Cross-platform, zero-dep, SSH-safe; casa com
  single-binary/zero-config. Register interno funciona mesmo se OSC 52 for bloqueado.
- **Callouts (ex-Q3): parse custom** sobre Blockquote, sem dep nova (regex na 1ª linha).
  `goldmark-callout` economiza pouco e adiciona dependência; só valeria p/ aninhamento
  complexo, fora do caso.
- **Criar nota / P4 (ex-Q5): subpasta da nota atual**, arquivo vazio (sem template).
  Template com frontmatter default = feature separada, pós-v2.

## Questões abertas

Nenhuma — todas resolvidas (ver **Decisões travadas**). Roadmap pronto pra virar
tasks executáveis (estilo `TASKS.md`) quando quiser começar pelo **M0**.
