# Nabat cascade demo — arg resolution waterfall
# Render with: nix run .#demo
# Or manually: build the binary first, then run vhs docs/tapes/cascade.tape
#   go build -o bin/basic ./examples/basic && export PATH="$PWD/bin:$PATH"
Output docs/assets/cascade.gif

Set FontSize 18
Set Margin 20
Set Padding 20
Set BorderRadius 15
Set Width 900
Set Height 380
Set Theme "Dracula"
Set TypingSpeed 80ms

# ── Run 1: no args → interactive prompt fires ─────────────────────────────────
Type "basic hello"
Enter
Sleep 800ms

# Fill the "Your name" prompt.
Type "alice"
Enter
Sleep 1500ms

# ── Run 2: env var resolves, prompt skipped entirely ──────────────────────────
Type "BASIC_NAME=carol basic hello"
Enter
Sleep 1500ms

# ── Run 3: CLI arg wins, nothing asked ────────────────────────────────────────
Type "basic hello bob"
Enter
Sleep 1500ms
