#!/bin/sh
# Plays the bundled story in a window.
DIR="$(dirname "$(readlink -f "$0")")"
cd "$DIR" && exec ./ango -window projects/intro
