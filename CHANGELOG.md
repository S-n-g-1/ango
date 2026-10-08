# Changelog

Format mengikuti [Keep a Changelog](https://keepachangelog.com/id-ID/1.1.0/) dan versi mengikuti [Semantic Versioning](https://semver.org/lang/id/).

## [0.1.1]

### Ditambahkan
- Dukungan Windows (amd64) sejajar dengan Linux: launcher memakai `ango.exe`, `explorer`, dan dialog folder Windows; proses anak tidak memunculkan jendela konsol.
- Build lintas-platform tanpa cgo: `make build GOOS=windows`, `make dist GOOS=...`, `make dist-all`, dan `scripts/build.ps1` untuk Windows tanpa `make`.
- Skrip pembungkus per OS: `scripts/ango.sh`, `scripts/play.sh`, `scripts/ango.bat`, `scripts/play.bat`.
- Direktori `test/` berisi tes integrasi lintas paket dan lintas platform (akhir baris CRLF/LF/CR, BOM, nama aset peka huruf besar-kecil, seluruh contoh dengan dua strategi pilihan, simpan dan pulihkan state).
- Flag `ango -version`.
- GitHub Actions: tes di Linux dan Windows, lalu build dan paket kedua platform.
- `.gitattributes` untuk normalisasi akhir baris, dan target `make fmt-check`, `cover`, `cross`.
- Dokumentasi baru di `docs/v0.1.1/`: memulai, referensi bahasa, platform, arsitektur, pengujian, launcher, tema GUI. `CONTRIBUTING.md` dan `CHANGELOG.md`.

### Diubah
- `go.mod`: dependensi langsung tidak lagi ditandai `// indirect`.
- Makefile ditulis ulang: variabel `GOOS`/`GOARCH`/`VERSION`, ekstensi `.exe`, arsip `.zip` untuk Windows.
- README dilengkapi: bagian platform, pengujian, dan daftar keterbatasan yang akurat.
- `ANGO_PROJECTS` dipisahkan `:` di Linux/macOS dan `;` di Windows (perilaku `filepath.SplitList`; kini terdokumentasi).

### Diperbaiki
- Target `make dist` merujuk `scripts/ango.sh` yang tidak ada; skrip tersebut kini disertakan (sebelumnya tersimpan salah nama sebagai `cmd/ango-launcher/build-linux.sh`).
- Tes launcher tidak lagi bergantung pada variabel lingkungan khusus Linux (`HOME`, `XDG_CONFIG_HOME`) atau pemisah `:`.

### Dihapus
- `tests/game_test.go` (hanya berisi catatan perintah) digantikan oleh `test/`; `build-linux.sh` dan `build-windows.sh` di akar digantikan target `make dist`.
- `docs/v0.1/` dipindahkan ke `docs/v0.1.1/` dengan isi yang disempurnakan.

## [0.1.0]

Rilis awal: bahasa `.ango` (dialog, variabel, percabangan, pilihan, label, `namespace`), compiler dengan error berposisi, VM, mode terminal, window Ebitengine (`scene`, `show`, `hide`, `flip`, transisi `fade`), tema GUI, dan launcher grafis.
