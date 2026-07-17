# Social preview (OG image) for GitHub, 1280x640 (2:1).
# Regenerate from the repo root: vhs docs/demo/og.tape
# Upload the PNG at: GitHub repo -> Settings -> General -> Social preview
Output docs/demo/og-scratch.gif

Set Shell "bash"
Set FontSize 20
Set Width 1280
Set Height 640
Set Padding 24

Hide
Type "go build -o /tmp/mdit-demo-bin ./cmd/mdit && rm -rf /tmp/mdit-demo && cp -r docs/demo/vault /tmp/mdit-demo && cd /tmp/mdit-demo && alias mdit=/tmp/mdit-demo-bin && clear"
Enter
Sleep 5s
Type "mdit showcase.md"
Enter
Sleep 2.5s
Down@100ms 12
Sleep 1s
Show
Screenshot docs/demo/social-preview.png
Sleep 500ms
