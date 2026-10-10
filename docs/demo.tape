# Records docs/demo.gif, the demo at the top of README.md, with VHS
# (https://github.com/charmbracelet/vhs).
#
# Run `vhs docs/demo.tape` from the home directory of a regular user, on a
# machine with the nginxlings package installed, nano, and no ~/nginxlings
# yet; it writes demo.gif there. /solution holds a solved copy of exercise 01:
# the solution is applied off camera, so the recording does not give it away.
# VHS cannot press F2, so the hidden part also moves back to nginxlings.

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
Sleep 900ms
Type "s"
Sleep 1.4s
Type "w"
Sleep 500ms
Type "w"
Sleep 500ms
Type "w"
Sleep 1.5s

# The starting files do not pass.
Type "c"
Wait+Screen /FAIL/
Sleep 4s

Type "h"
Sleep 4s
Type "h"
Sleep 1s

# t moves to the terminal below, in the exercise's folder.
Type "t"
Sleep 1.5s
Type "nano nginx.conf"
Sleep 500ms
Enter
Sleep 3.5s

Hide
Ctrl+X
Sleep 500ms
Type "cp /solution/* . && tmux select-pane -t :.0 && clear"
Enter
Sleep 1s
Show

Sleep 1s
Type "c"
Wait+Screen /PASS/
Sleep 5s
