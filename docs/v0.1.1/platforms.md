# Platform: Linux dan Windows

> Bagian dari [dokumentasi Ango](README.md).

Ango diuji dan dirilis untuk **Linux (amd64)** dan **Windows (amd64)**. macOS dan Android adalah target berikutnya dan belum didukung secara resmi.

## Ringkasan

| | Linux | Windows |
| --- | --- | --- |
| Binary | `ango`, `ango-launcher` | `ango.exe`, `ango-launcher.exe` |
| Paket rilis | `.tar.gz` | `.zip` |
| Skrip bantu | `ango.sh`, `play.sh` | `ango.bat`, `play.bat` |
| Library sistem saat dijalankan | OpenGL, X11 | tidak ada |
| Compiler C (cgo) | tidak perlu | tidak perlu |
| Lokasi konfigurasi launcher | `~/.config/ango` | `%AppData%\ango` |
| Pemisah `ANGO_PROJECTS` | `:` | `;` |
| Pembuka folder | `xdg-open` | `explorer` |
| Pemilih folder | `zenity` / `kdialog` | dialog Windows (PowerShell) |

Build memakai `CGO_ENABLED=0`. Ebitengine memuat library grafis saat dijalankan, sehingga **tidak diperlukan compiler C** dan lintas-kompilasi antar kedua OS bisa dilakukan dari mesin mana pun.

## Linux

Library yang dibutuhkan saat dijalankan (nama paket Debian/Ubuntu):

```bash
sudo apt install libgl1 libx11-6 libxcursor1 libxrandr2 libxinerama1 libxi6 libxxf86vm1
```

Untuk pemilih folder di launcher: `sudo apt install zenity` (atau `kdialog` di KDE).

Build dan jalankan:

```bash
make build
./bin/ango -window examples/intro
```

## Windows

Tidak ada dependensi tambahan. Dengan `make` (Git Bash, MSYS2, atau WSL) perintahnya sama seperti Linux. Tanpa `make`:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\build.ps1            # bin\ango.exe, bin\ango-launcher.exe
powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -Dist -Version 0.1.1   # dist\ango-0.1.1-windows-amd64.zip
```

`ango-launcher.exe` dibangun sebagai program GUI (`-H=windowsgui`) sehingga tidak membuka jendela konsol. `ango.exe` tetap program konsol agar mode terminal dan `-check` dapat menulis ke terminal.

## Lintas-kompilasi

```bash
make build GOOS=windows GOARCH=amd64     # bin/ango.exe, bin/ango-launcher.exe
make dist  GOOS=windows                  # dist/ango-<versi>-windows-amd64.zip   (butuh `zip`)
make dist  GOOS=linux                    # dist/ango-<versi>-linux-amd64.tar.gz
make dist-all                            # keduanya
make cross                               # compile + vet untuk kedua OS, tanpa menghasilkan paket
```

Target `test` selalu berjalan di OS Anda sendiri; binary hasil lintas-kompilasi tidak dapat dijalankan di OS lain.

## Menulis cerita yang portabel

Aturan berikut menjaga satu cerita berjalan sama di kedua OS:

1. **Nama aset peka huruf besar-kecil.** `show asep "calm"` mungkin berjalan di Windows tetapi gagal di Linux bila berkasnya `Calm.png`. Tes `TestAssetNamesMatchCaseExactly` menangkapnya.
2. **Pemisah path di skrip selalu `/`**, mis. `scene "kelas/pagi"`.
3. **Akhir baris bebas.** `\n`, `\r\n`, dan `\r` menghasilkan program yang sama; BOM UTF-8 diabaikan. `.gitattributes` menormalkan akhir baris di repositori.
4. **Hindari karakter terlarang Windows** pada nama berkas aset: `< > : " \ | ? *`, serta nama cadangan seperti `CON` dan `NUL`.
5. **Hindari dua berkas yang hanya beda huruf besar-kecil** (`Calm.png` dan `calm.png`); tidak bisa berdampingan di Windows.

## Pemecahan masalah

| Gejala | Penyebab dan solusi |
| --- | --- |
| Linux: `libGL.so.1` atau `libX11.so.6` tidak ditemukan | Pasang library di bagian [Linux](#linux) |
| Linux: window tidak muncul lewat SSH | Tidak ada display; jalankan di sesi desktop atau pakai mode terminal (`ango -auto`) |
| Aset tampil sebagai blok placeholder hanya di Linux | Beda huruf besar-kecil nama aset; jalankan `go test ./test` |
| Windows: SmartScreen memblokir `ango.exe` | Binary belum ditandatangani; pilih "More info" lalu "Run anyway" |
| Windows: launcher tidak menemukan `ango.exe` | Taruh di folder yang sama, atau pakai `-ango <path>` / `ANGO_BIN` |
| Tombol "Tambah Folder" memberi pesan tidak ada pemilih | Linux tanpa `zenity`/`kdialog`: pasang salah satunya, atau berikan folder lewat argumen |
