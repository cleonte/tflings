# File used to generate misc/demo-tui.gif, the demo of `tflings watch`.
#
# Needs vhs (https://github.com/charmbracelet/vhs) and a tflings on the PATH
# (`go install ./cmd/tflings`, or brew), then:  vhs demo-tui.tape
# It uses a scratch home and folder, so nothing of yours is touched.

Output misc/demo-tui.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 640
Set Padding 15
Set Framerate 20
Set TypingSpeed 70ms

# Set up a fresh workspace, out of sight. A background job saves the answer of the
# first exercise (a quiz) after a while, the way an editor saves a fix.
Hide
Type `export TFLINGS_HOME=$(mktemp -d) && cd $(mktemp -d) && tflings init > /dev/null && cd tflings-exercises`
Enter
Type `(sleep 11; sed -i.bak 's/^answer = ""$/answer = "b"/' exercises/iac/iac1/quiz.toml) & disown`
Enter
Type "clear"
Enter
Show

Type "tflings watch"
Sleep 600ms
Enter
Sleep 4s

# The hint.
Type "h"
Sleep 3s
Escape
Sleep 1s

# The answer is saved by the background job: the view checks the exercise again.
Sleep 5s

# n takes the marker out of the exercise that works and moves on.
Type "n"
Sleep 3s

# l lists every exercise: filter by name or topic, enter opens one.
Type "l"
Sleep 2s
Type "/"
Sleep 400ms
Type "workspaces"
Sleep 1s
Enter
Sleep 1s
Enter
Sleep 3s

# c checks all the exercises, without changing any file.
Type "c"
Sleep 3s
Escape
Sleep 1s

Type "q"
Sleep 1s
