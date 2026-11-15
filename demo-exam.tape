# File used to generate misc/demo-exam.gif, the demo of `tflings exam`.
#
# Needs vhs (https://github.com/charmbracelet/vhs) and a tflings on the PATH, then:
#   vhs demo-exam.tape
# --seed 5 asks the same three questions every time, as long as the question bank
# is the one of v0.9.0; the answers typed below match them. If the bank changes,
# look at the questions again and adjust the answers.

Output misc/demo-exam.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 700
Set Padding 15
Set Framerate 20
Set TypingSpeed 60ms

Hide
Type `export TFLINGS_HOME=$(mktemp -d) && cd $(mktemp -d) && tflings init > /dev/null && cd tflings-exercises`
Enter
Type "clear"
Enter
Show

Type "tflings exam --practice -n 3 --seed 5"
Sleep 500ms
Enter
Sleep 3s

# A question with two answers.
Type "c,d"
Sleep 600ms
Enter
Sleep 3s

Type "c"
Sleep 600ms
Enter
Sleep 3s

# A question with a code snippet.
Type "b"
Sleep 600ms
Enter
Sleep 4s

# What is left to learn, per exam objective.
Type "clear"
Enter
Type "tflings progress | head -22"
Sleep 500ms
Enter
Sleep 5s
