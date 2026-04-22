# Nabat theme demo — four of the twelve built-in themes
# Render with: nix run .#demo
# Or manually: build the binary first, then run vhs docs/tapes/theme.tape
#   go build -o bin/theme ./examples/theme && export PATH="$PWD/bin:$PATH"
Output docs/assets/theme.gif

Set FontSize 18
Set Margin 20
Set Padding 20
Set BorderRadius 15
Set Width 900
Set Height 500
Set Theme "Dracula"
Set TypingSpeed 60ms

Type "NABAT_THEME=nord theme show"
Enter
Sleep 2500ms

Type "NABAT_THEME=gruvbox theme show"
Enter
Sleep 2500ms

Type "NABAT_THEME=catppuccin-mocha theme show"
Enter
Sleep 2500ms

Type "NABAT_THEME=nabat theme show"
Enter
Sleep 2500ms
