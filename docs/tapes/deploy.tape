# Nabat deploy demo
# Render with: nix run .#demo
# Or manually: build the binary first, then run vhs docs/tapes/deploy.tape
#   go build -o bin/deployctl ./examples/deploy && export PATH="$PWD/bin:$PATH"
Output docs/assets/deploy.gif

Set FontSize 18
Set Margin 20
Set Padding 20
Set BorderRadius 15
Set Width 1000
Set Height 580
# VHS terminal colors: Nabat dark — palette/nabat.json flavors.dark (ansi + roles).
# Set Theme { "name": "Nabat", "black": "#1F1A16", "red": "#E05454", "green": "#7EC87E", "yellow": "#E8A83A", "blue": "#3EB0CC", "magenta": "#C8568C", "cyan": "#5FBFA8", "white": "#EDE4D3", "brightBlack": "#8A7E72", "brightRed": "#F07878", "brightGreen": "#A0DCA0", "brightYellow": "#F5C260", "brightBlue": "#6CC8DC", "brightMagenta": "#DC78A4", "brightCyan": "#88D2BE", "brightWhite": "#FFF8E7", "background": "#1F1A16", "foreground": "#EDE4D3", "selection": "#2A231D", "cursor": "#C89B3C" }
Set Theme "Dracula"
Set TypingSpeed 100ms

# ── Run 1: full interactive path ──────────────────────────────────────────────
# No args → select prompt → confirm → spinner → progress bar → table.

Type "deployctl deploy"
Enter
Sleep 1000ms

# Select prompt: "staging" is highlighted by default — accept it.
Enter
Sleep 500ms

# Confirm "Deploy latest to staging?" — default is No; Left moves to Yes.
Left
Enter
Sleep 300ms

# Spinner 800ms + progress bar 2 pods × 300ms = 1 600ms total.
Sleep 2000ms

# Let the table sit on screen.
Sleep 2500ms

# ── Run 2: non-interactive / CI path ──────────────────────────────────────────
# --yes skips the confirm; production + pinned tag shows the real-world flags.

Type "deployctl deploy production --replicas 3 --tag v1.2.0 --yes"
Enter
Sleep 300ms

# Spinner 800ms + bar 3 × 300ms = 1 700ms total.
Sleep 2800ms

# Three-pod table.
Sleep 2500ms

# ── Run 3: tree output ────────────────────────────────────────────────────────

Type "deployctl deploy staging -o tree --yes"
Enter
Sleep 300ms

# Spinner 800ms + bar 2 × 300ms = 1 400ms total.
Sleep 2500ms

# Tree view — hold a beat so viewers can read it.
Sleep 3000ms
