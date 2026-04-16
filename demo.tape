Output demo.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1200
Set Height 600
Set Padding 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

# Launch hustle with the demo log file
Type "hustle testdata/app.json"
Enter
Sleep 2.5s

# Scroll down through the logs
Down 5
Sleep 700ms
Down 5
Sleep 700ms
Down 5
Sleep 1.5s

# Scroll back to top
Up 15
Sleep 1s

# Show help overlay
Type "?"
Sleep 3s
Escape
Sleep 1s

# Quick filter: search for timeout
Type "/"
Sleep 400ms
Type "timeout"
Sleep 2s

# Clear and search for cache
Backspace 7
Sleep 300ms
Type "cache"
Sleep 2s

# Clear filter
Escape
Sleep 1s

# jq filter: find slow requests (duration > 200ms)
Type "j"
Sleep 500ms
Type ".duration_ms > 200"
Sleep 2s
Enter
Sleep 4s

# Clear jq filter
Type "j"
Sleep 500ms
Ctrl+U
Enter
Sleep 1.5s

# Select a row and view details
Down 8
Sleep 500ms
Enter
Sleep 2.5s

# Back to grid
Escape
Sleep 1.5s

# Quit
Type "q"
Sleep 500ms
