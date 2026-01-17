# How I got the screenshot

This was a surprising PITA:

- open a new kitty window
- run `export PS1='$ '` and `export RPROMPT=''`
- run `kitten @ set-window-title ' '` which clears the window title; an empty string doesn't work, you need the space
- run `aver` in the fixtures dir
