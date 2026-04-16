Output demo.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1200
Set Height 600
Set Padding 10
Set Theme "Catppuccin Mocha"

# Launch hustle
Type "hustle /tmp/demo.log"
Enter
Sleep 2s

# Scroll down through the logs
Down 5
Sleep 800ms
Down 5
Sleep 800ms
Down 5
Sleep 1.5s

# Scroll back up
Up 15
Sleep 1s

# Open quick filter and search for errors
Type "/"
Sleep 500ms
Type "error"
Sleep 2s

# Clear filter and try another search
Backspace 5
Sleep 500ms
Type "payment"
Sleep 2s

# Clear filter
Escape
Sleep 1s

# Select a row and view details
Down 3
Sleep 500ms
Enter
Sleep 2.5s

# Go back to grid
Escape
Sleep 1.5s

# Quit
Type "q"
Sleep 500ms
