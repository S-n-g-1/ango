#!/bin/sh
set -e
OUT=dist/ango-linux-amd64
rm -rf "$OUT"
mkdir -p "$OUT"
go build -trimpath -ldflags "-s -w" -o "$OUT/ango" ./cmd/ango
cp -r examples/intro "$OUT/story"
printf '#!/bin/sh\ncd "$(dirname "$0")" && exec ./ango -window story\n' > "$OUT/play.sh"
chmod +x "$OUT/play.sh"
tar -C dist -czf dist/ango-linux-amd64.tar.gz ango-linux-amd64
ls -lh dist/ango-linux-amd64.tar.gz
