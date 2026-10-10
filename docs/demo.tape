# Records docs/demo.gif, the demo at the top of README.md, with VHS
# (https://github.com/charmbracelet/vhs).
#
# Run `vhs docs/demo.tape` from the home directory of a regular user, on a
# machine with the nginxlings package installed, nano, and no ~/nginxlings
# yet; it writes demo.gif there. The recording solves exercise 01 on camera.
# VHS cannot press F2, so a hidden command moves back to nginxlings.

Output demo.gif

Set Shell bash
Set FontSize 15
Set Width 1200
Set Height 780
Set Padding 14
Set TypingSpeed 70ms
Set WaitTimeout 60s

Type "nginxlings"
Sleep 600ms
Enter
Wait+Screen /Files:/
Sleep 2.5s

# Browse the list; the instructions follow the selection.
Type "s"
Sleep 900ms
Type "s"
Sleep 1.2s
Type "w"
Sleep 500ms
Type "w"
Sleep 1.5s

# The starting files do not pass. A checked exercise is no longer dimmed.
Type "c"
Wait+Screen /FAIL/
Sleep 3.5s
Type "s"
Sleep 1.5s
Type "w"
Sleep 800ms

Type "h"
Sleep 3.5s
Type "h"
Sleep 800ms

# t moves to the terminal below, in the exercise's folder.
Type "t"
Sleep 1.5s

Type "nano nginx.conf"
Sleep 400ms
Enter
Sleep 1.2s
Ctrl+W
Type "missing"
Enter
Sleep 900ms
Delete 7
Type "."
Sleep 900ms
Ctrl+W
Enter
Sleep 700ms
Delete 7
Type "index"
Sleep 1.2s
Ctrl+O
Enter
Sleep 500ms
Ctrl+X
Sleep 700ms

Type "nano index.html"
Sleep 400ms
Enter
Sleep 1.2s
Ctrl+W
Type "Static server ex"
Enter
Sleep 900ms
Delete 22
Type "Merhaba Nginxlings"
Sleep 1.2s
Ctrl+O
Enter
Sleep 500ms
Ctrl+X
Sleep 700ms

Hide
Type "tmux select-pane -t :.0 && clear"
Enter
Sleep 1s
Show

Sleep 800ms
Type "c"
Wait+Screen /PASS/
Sleep 3.5s
Type "s"
Sleep 4s
