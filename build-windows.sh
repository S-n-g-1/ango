#!/bin/sh
set -e
OUT=dist/ango-windows-amd64
rm -rf "$OUT"
mkdir -p "$OUT"
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$OUT/ango.exe" ./cmd/ango
cp -r examples/intro "$OUT/story"
printf '@echo off\ncd /d %%~dp0\nango.exe -window story\npause\n' > "$OUT/play.bat"
cd dist && zip -r ango-windows-amd64.zip ango-windows-amd64
ls -lh dist/ango-windows-amd64.zip