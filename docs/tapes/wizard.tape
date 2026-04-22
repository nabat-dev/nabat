# Nabat wizard demo — multi-page form
# Render with: nix run .#demo
# Or manually: build the binary first, then run vhs docs/tapes/wizard.tape
#   go build -o bin/deployctl ./examples/deployctl && export PATH="$PWD/bin:$PATH"
Output docs/assets/wizard.gif

Set FontSize 18
Set Margin 20
Set Padding 20
Set BorderRadius 15
Set Width 1000
Set Height 580
Set Theme "Dracula"
Set TypingSpeed 80ms

Type "deployctl configure"
Enter
Sleep 1000ms

# Page 1 — Identity
# Cursor lands on the Name field (note is read-only display).
Type "alice"
Enter
Sleep 500ms

# Submit page 1 → advance to page 2.
Enter
Sleep 500ms

# Page 2 — Deployment
# Environment select: "staging" is the default — accept it.
Enter
Sleep 300ms

# Confirm field: default is No; Left moves to Yes.
Left
Enter
Sleep 300ms

# Hold the result summary on screen.
Sleep 3000ms
