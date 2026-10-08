#!/bin/sh
# Opens the Ango launcher. Works from any directory and through symlinks;
# arguments (project folders) are passed through unchanged.
DIR="$(dirname "$(readlink -f "$0")")"
exec "$DIR/ango-launcher" "$@"
