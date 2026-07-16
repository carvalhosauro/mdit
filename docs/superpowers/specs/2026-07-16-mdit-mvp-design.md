# mdit — Design do MVP

**Data:** 2026-07-16
**Status:** Aprovado pelo autor
**Licença do projeto:** MIT (OSS)

## Visão geral

`mdit` é um editor de markdown para terminal (TUI), escrito em Go, inspirado no
[pixdeo/editxr](https://github.com/pixdeo/editxr). O diferencial é o **live render
inline**: o documento é exibido renderizado (headings grandes, negrito real, bullets,
tasks) diretamente no buffer de edição — apenas o bloco sob o cursor aparece como
markdown cru. Você edita sempre o arquivo real, nunca uma cópia.

Complementam o MVP: **wikilinks** estilo Obsidian resolvidos contra um vault (pasta),
**zen mode** (leitura renderizada, read-only) e **fuzzy finder** de notas.

Uso: `mdit arquivo.md` ou `mdit pasta/` (a pasta — ou a pasta do arquivo — vira o vault).

## Objetivos do MVP

1. Editar markdown com live render inline (bloco sob cursor = cru; resto = renderizado).
2. Wikilinks: renderizar `[[nota]]`, seguir com Enter/Ctrl+], autocomplete ao digitar
   `[[`, destacar link quebrado.
3. Zen mode: documento inteiro renderizado, read-only, coluna centralizada (~80 cols).
4. Fuzzy finder (Ctrl+P) sobre as notas do vault.
5. Salvar com segurança (detecção de modificação externa, indicador dirty, prompt no quit).
6. Binário único, cross-platform (linux/mac/windows), `go install` + releases.

## Não-objetivos (pós-MVP)

- Temas configuráveis / detecção de background (MVP: 1 tema dark embutido).
- Busca full-text no vault e busca incremental no arquivo.
- Backlinks, criação de nota ao seguir link quebrado.
- Tabs / múltiplos arquivos abertos (MVP: 1 doc por vez + histórico de navegação).
- Edição assistida por IA (diferencial do editxr; avaliar depois).
- Export HTML, vim mode, mouse, line numbers.
- Parse incremental (MVP reparseia o documento por edit — suficiente, ver Performance).
- "Lazy raw" em tabelas: tabela permanece renderizada com cursor em cima; vira raw
  apenas ao editar; re-renderiza só quando o cursor sai. (Ideia do autor, registrada
  para pós-MVP; no MVP, entrar no bloco já o torna cru.)
- SQLite para índice do vault — só se backlinks + full-text em vaults gigantes
  justificarem. `vault` fica atrás de interface para permitir a troca.

## Decisões de escopo (com alternativas rejeitadas)

| Decisão | Escolha | Rejeitado |
|---|---|---|
| Plataforma | TUI (terminal) | GUI nativa (Fyne/Gio), local web app |
| Live render | Inline estilo Obsidian, 1 pane | Split pane, toggle cru/rendered |
| Wikilinks | Navegar + autocomplete | Só navegar; backlinks completos |
| Teclas | Modeless (estilo nano/micro) | Modal vim-like; híbrido |
| Markdown | CommonMark + GFM básico | CommonMark puro; GFM completo + callouts/math |
| Extras MVP | Fuzzy finder | Busca full-text, temas |
| Stack | Bubble Tea + goldmark | tcell puro (2.5x esforço); lexer linha-a-linha (incorreto em blocos multi-linha, 2 implementações de markdown) |
| Índice vault | Map em memória, rescan no startup | SQLite (CGO/migração/invalidação sem benefício nesta escala) |

## Markdown suportado (MVP)

Headings, ênfase (negrito/itálico/código inline), listas (ordenadas, não ordenadas,
aninhadas), task lists (`- [ ]`), code blocks com syntax highlight (chroma), blockquote,
links `[t](url)`, tabelas (render-only), réguas horizontais, frontmatter YAML
(renderizado como bloco discreto) e wikilinks `[[nota]]` / `[[nota|alias]]`.

## Arquitetura

Stack: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (arquitetura Elm),
lipgloss (estilos), bubbles (fuzzy finder/inputs/viewport),
[goldmark](https://github.com/yuin/goldmark) (parser CommonMark/GFM, AST extensível —
wikilink implementado como extensão do parser), chroma (highlight de code fence),
go-runewidth (largura de células unicode).

### Pacotes

```
cmd/mdit/         main + flags (stdlib flag; mdit [arquivo|pasta])
internal/doc/     buffer de linhas, operações de edição, undo/redo, dirty, save
internal/vault/   scan de *.md, índice nome→path, resolve/autocomplete de wikilink
internal/mdparse/ goldmark + extensão wikilink → []Block (range de linhas cruas + AST)
internal/render/  Block → linhas estilizadas (lipgloss), cache, width-aware
internal/editor/  widget: viewport virtualizado, cursor, bloco-sob-cursor cru
internal/ui/      model raiz Bubble Tea: editor | zen | fuzzy finder, statusbar, prompts
internal/theme/   estilos centralizados (1 tema dark)
```

### Fluxo de dados (edição)

```
keypress → ui (roteia por modo) → doc (aplica op + registra patch inverso)
        → mdparse (reparse full-document → []Block)
        → render (re-renderiza apenas blocos visíveis/invalidados)
        → editor (mapping cursor cru→tela, scroll) → View()
```

Zen mode reusa o mesmo pipeline de render (full-width centralizado, nenhum bloco cru,
edits bloqueados, navegação = scroll). Fuzzy finder é overlay sobre o índice do vault.

### Modelo do editor

- **Cursor vive em coordenadas cruas** (linha, coluna em runes). A tela é sempre
  derivada; nunca é fonte de verdade.
- **Unidade de "cru" = bloco, não linha.** Cursor dentro do bloco → bloco inteiro exibe
  markdown cru; ao sair, re-renderiza. Para parágrafo de 1 linha equivale ao
  comportamento do Obsidian; para code fence/tabela (multi-linha) é a única semântica
  consistente — não existe mapping honesto para editar uma linha interna de uma tabela
  renderizada.
- **Edits são operações no doc** (inserir/apagar rune, quebrar/unir linha). Cada
  operação gera patch inverso → undo/redo = pilhas de patches, coalescidos por pausa de
  digitação. Redo é invalidado por novo edit.
- **Soft-wrap em tudo** (blocos crus e renderizados). Mapping coluna crua→coluna de tela
  usa largura real de rune via go-runewidth (CJK/emoji = 2 células).
- **Autocomplete de wikilink:** digitar `[[` abre popup fuzzy sobre o índice do vault;
  Enter completa `nota]]`; Esc cancela.
- **Seguir link:** Ctrl+] com cursor sobre wikilink → vault resolve por nome → troca de
  documento (prompt se dirty). Enter não serve para seguir link: com o cursor sobre o
  link o bloco está cru (modo edição) e Enter insere quebra de linha. Ctrl+B volta
  (pilha de histórico de navegação; Ctrl+[ é descartado por ser o mesmo byte que Esc).
  Link quebrado: estilizado em vermelho; seguir exibe aviso na statusbar.

### Virtualização (day-1)

O render é virtualizado desde o início — cai naturalmente do design em blocos:

1. `mdparse` produz `[]Block`; cada bloco conhece seu range de linhas cruas.
2. Cada bloco expõe `Height(width) int` e `Render(width) []styledLine`, ambos com cache
   invalidado por edit no range ou resize.
3. O viewport guarda offset em linhas-de-tela + prefix sum das alturas → blocos visíveis
   localizados por busca binária (O(log n)).
4. `View()` materializa apenas blocos que intersectam a janela; o resto do arquivo nunca
   é renderizado.

### Performance

Parse é full-document por keystroke no MVP: goldmark processa dezenas/centenas de KB em
~1ms — notas markdown são KBs, não MBs. O custo real é o render (estilização + wrap), e
esse é virtualizado + cacheado. Parse incremental por bloco é otimização pós-MVP, sem
mudança de arquitetura (a interface `[]Block` permanece).

### Vault

- Ao abrir, walk da pasta raiz coletando `*.md` → map nome-normalizado → path
  (+ lista para o fuzzy finder).
- Resolução de `[[nota]]`: match por nome de arquivo sem extensão, case-insensitive;
  ambiguidade resolvida pelo path mais curto (regra simples e documentada).
- Índice em memória; reconstruído no startup. Sem persistência em disco no MVP.
- Interface pequena (`Resolve`, `Complete`, `List`) para permitir trocar a
  implementação (ex.: índice persistente) sem tocar no resto.

## Keybindings (MVP)

| Tecla | Ação |
|---|---|
| Ctrl+S | Salvar |
| Ctrl+Q | Sair (prompt se dirty) |
| Ctrl+P | Fuzzy finder de notas |
| Ctrl+E | Toggle zen mode |
| Ctrl+] | Seguir wikilink sob cursor |
| Ctrl+B | Voltar (histórico de navegação) |
| Ctrl+Z / Ctrl+Y | Undo / Redo |
| Setas, PgUp/PgDn, Home/End | Navegação |
| Ctrl+←/→ | Mover por palavra |

## Tratamento de erros

- **Save falha:** erro na statusbar; doc permanece dirty. O buffer nunca é perdido.
- **Modificação externa:** no save, se o mtime difere do registrado no load → prompt
  sobrescrever / recarregar / cancelar.
- **Parse:** markdown não tem erro de sintaxe; goldmark sempre produz árvore. Não existe
  estado de erro de parse.
- **Panic:** Bubble Tea restaura o terminal; defer grava o buffer em
  `<arquivo>.mdit-recover` antes de encerrar.
- **Quit com unsaved:** prompt salvar / descartar / cancelar.

## Testes

- `doc`: unit + property-based (sequência aleatória de edits seguida dos undos ⇒
  documento idêntico ao original).
- `mdparse`: golden tests markdown → blocos (incluindo wikilink, fence, tabela,
  frontmatter, listas aninhadas).
- `render`: golden tests bloco → linhas estilizadas em largura fixa.
- `editor`: unit de mapping cursor/scroll com blocos wrapped — maior risco do projeto,
  cobertura pesada aqui.
- Integração: `teatest` (Charm) — abrir arquivo, digitar, seguir link, zen toggle,
  fuzzy finder, salvar.
- CI: GitHub Actions — golangci-lint + testes em linux/mac/windows.

## Distribuição

`go install`, binários em GitHub Releases (goreleaser), sem CGO. Homebrew tap e AUR
pós-MVP.

## Referência competitiva

editxr (Swift, MIT, ~64 stars, ativo): inline WYSIWYG + wikilinks + tabs + IA com diff
inline + 12 temas. mdit se diferencia com vault Obsidian real (autocomplete +
resolução por nome; editxr apenas segue links) e zen mode read-only dedicado.
Pesquisa completa: notas da sessão de 2026-07-16.
