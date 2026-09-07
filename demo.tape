# VHS demo tape for tmux-peeker
# Install VHS: https://github.com/charmbracelet/vhs
# Run: vhs demo.tape

Output assets/demo.gif

Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"

Type "tmux-peeker"
Enter
Sleep 2s

# Navigate sessions
Down
Sleep 500ms
Down
Sleep 500ms
Up
Sleep 1s

# Filter
Type "/"
Sleep 300ms
Type "dev"
Sleep 1s
Escape
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
