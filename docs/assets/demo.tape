# Re-record: from the repository root,
#   VHS_NO_SANDBOX=true timeout 90 xvfb-run -a vhs docs/assets/demo.tape
# (headless Chromium needs VHS_NO_SANDBOX and a virtual display in a
# devcontainer; see /home/vscode/vhs-setup-notes.md for the one-time setup.
# vhs must be v0.11.0 -- v0.12.0 has a known regression that silently
# produces no GIF at all.)
#
# mtqg itself must be built and on PATH (go install ./cmd/mtqg) before
# recording, since this tape only sets up a throwaway demo git repository,
# not mtqg.

Output docs/assets/demo.gif

Set Shell "bash"
Set FontFamily "DejaVu Sans Mono"
Set FontSize 16
Set Width 1200
Set Height 650
Set Padding 20
Set TypingSpeed 45ms
Set Theme "Dracula"

Hide
Type "rm -rf /tmp/mtqg-demo && mkdir -p /tmp/mtqg-demo/sample-parser && cd /tmp/mtqg-demo/sample-parser && git init -q && git config user.name yamada && git config user.email yamada@example.com && echo '# sample-parser' > README.md && git add README.md && git commit -q -m 'Initial commit' && export MTQG_AUTHOR_KIND=human MTQG_AUTHOR_NAME=yamada && export PS1='$ ' && clear"
Enter
Sleep 500ms
Show

Type "mtqg init"
Enter
Sleep 1200ms

Type "mtqg t add Support C syntax"
Enter
Sleep 900ms

Type "q=$(mtqg q add 'Should error positions show both line and column?')"
Enter
Sleep 900ms

Type "mtqg q add $q 'Yes, callers usually want both'"
Enter
Sleep 900ms

Type "mtqg q list"
Enter
Sleep 1500ms

Type "mtqg log"
Enter
Sleep 1500ms

Type "mtqg context"
Enter
Sleep 3s
