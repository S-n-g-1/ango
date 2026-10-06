#!/bin/sh
# Membuka Ango Launcher. Bisa dipanggil dari folder mana saja, juga lewat
# symlink; argumen (folder proyek) diteruskan apa adanya.
DIR="$(dirname "$(readlink -f "$0")")"
exec "$DIR/ango-launcher" "$@"