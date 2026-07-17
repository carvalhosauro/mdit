# Demo GIF for the README.
# Regenerate from the repo root: vhs docs/demo/demo.tape
Output docs/demo/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1000
Set Height 600
Set Padding 12
Set TypingSpeed 60ms

# hidden setup: build the binary and work on a throwaway copy of the demo vault
Hide
Type "go build -o /tmp/mdit-demo-bin ./cmd/mdit && rm -rf /tmp/mdit-demo && cp -r docs/demo/vault /tmp/mdit-demo && cd /tmp/mdit-demo && alias mdit=/tmp/mdit-demo-bin && clear"
Enter
Sleep 5s
Show

Type "mdit ideas.md"
Enter
Sleep 2.5s

# jump to the empty last line; type a wikilink — autocomplete pops on [[
Down@100ms 8
Type "Also read [[roa"
Sleep 1.5s
Enter
Sleep 800ms
Type "."
Sleep 300ms
Ctrl+S
Sleep 700ms

# put the cursor on the link and follow it
Left@50ms 7
Sleep 500ms
Ctrl+]
Sleep 2s

# back to where we came from
Ctrl+B
Sleep 1.5s

# fuzzy finder over the vault ("/" starts the filter)
Ctrl+P
Sleep 1s
Type@120ms "/road"
Sleep 1s
Enter
Sleep 500ms
Enter
Sleep 1.5s

# zen mode
Ctrl+E
Sleep 3s
