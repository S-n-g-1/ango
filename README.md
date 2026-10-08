# Ango

**Engine visual novel berbasis Go dengan bahasa skrip sendiri.**

Ceritanya ditulis di berkas teks `.ango` yang ringkas, dikompilasi menjadi bytecode, lalu dimainkan oleh VM di terminal atau di window grafis ([Ebitengine](https://ebitengine.org)). Berjalan di **Linux** dan **Windows** dari satu kode sumber, tanpa compiler C.

> **Status: v0.1.1, tahap awal.** Bahasa dan API masih bisa berubah. Lihat [Keterbatasan](#keterbatasan-saat-ini) dan [CHANGELOG](CHANGELOG.md).

## Daftar isi

- [Fitur](#fitur)
- [Mulai cepat](#mulai-cepat)
- [Contoh skrip](#contoh-skrip)
- [Struktur game](#struktur-game)
- [Menjalankan](#menjalankan)
- [Launcher](#launcher)
- [Platform yang didukung](#platform-yang-didukung)
- [Membangun dan menguji](#membangun-dan-menguji)
- [Struktur repositori](#struktur-repositori)
- [Dokumentasi](#dokumentasi)
- [Keterbatasan saat ini](#keterbatasan-saat-ini)
- [Berkontribusi](#berkontribusi)
- [Lisensi](#lisensi)

## Fitur

- **Bahasa skrip `.ango`**: dialog, narasi, variabel, percabangan, pilihan bersyarat, dan lompatan antar label (`jump`, `call`, `return`).
- **Cerita banyak berkas**: satu folder adalah satu game; `namespace` opsional per berkas mencegah bentrok nama label.
- **Compiler dengan error berposisi**: `ango -check` melaporkan semua kesalahan sekaligus beserta berkas, baris, dan kolom.
- **Dua mode main**: terminal (pilihan bernomor) dan window grafis.
- **Latar dan karakter**: `scene`, `show`, `hide`, posisi `left`/`center`/`right`, `flip` untuk membalik sprite, dan transisi `fade`. Beberapa karakter dapat tampil bersamaan.
- **Tema GUI** lewat `gui/theme.json` untuk kotak dialog, plat nama, tombol, serta menu utama dan menu jeda.
- **Launcher grafis** (`ango-launcher`) untuk memilih proyek lalu menjalankan, memeriksa, atau menguji otomatis.
- **Format gambar**: WebP, PNG, JPEG, dan SVG. Aset yang belum ada diganti placeholder sehingga skrip bisa dijalankan sebelum seninya siap.
- **Lintas-platform**: Linux dan Windows, dengan build lintas-kompilasi dan paket rilis untuk keduanya.

## Mulai cepat

### Dari paket rilis

| Platform | Langkah |
| --- | --- |
| Linux | `tar -xzf ango-<versi>-linux-amd64.tar.gz`, lalu `./play.sh` (contoh) atau `./ango.sh` (launcher) |
| Windows | Ekstrak `ango-<versi>-windows-amd64.zip`, lalu klik dua kali `play.bat` atau `ango.bat` |

### Dari kode sumber

Butuh Go sesuai `go.mod`. Di Linux juga butuh library OpenGL dan X11 ([detail](docs/v0.1.1/platforms.md#linux)).

```bash
make build                          # bin/ango dan bin/ango-launcher
./bin/ango -window examples/intro   # Windows: bin\ango.exe -window examples\intro
```

Tanpa `make`:

```bash
go run ./cmd/ango -window examples/intro
```

```powershell
# Windows tanpa make
powershell -ExecutionPolicy Bypass -File scripts\build.ps1
```

## Contoh skrip

```ango
default player_name = "Asep"
default courage = 0

label start
scene "Kelas" with fade 0.8
show asep "Calm" at left with fade 0.4

"Hujan belum berhenti sejak subuh."
Asep: "Halo, aku [player_name]."

choice:
    "Periksa laci":
        set courage = courage + 1
        Asep: "Ada selembar kertas lipat."
        jump finale
    "Pulang saja":
        jump finale

label finale
hide asep with fade 0.4
if courage >= 1:
    "Kamu pulang dengan kepala tegak."
else:
    "Kamu pulang sambil menoleh ke belakang."
end
```

Panduan lengkap bahasa: [Referensi bahasa](docs/v0.1.1/language-reference.md).

## Struktur game

Satu game adalah satu folder. Semua berkas `.ango` di dalamnya (termasuk subfolder) dimuat sekaligus, dan cerita dimulai dari label `start` yang harus ada dan unik.

```
examples/intro/
├── main.ango          # tanpa namespace: berisi `start` dan variabel global
├── chapter1.ango      # namespace chapter1
├── chapter2.ango      # namespace chapter2
└── assets/
    ├── backgrounds/   # Kelas.webp, ...
    ├── characters/
    │   ├── asep/      # Calm.png, Smile.png, ...
    │   └── dina/
    └── gui/           # opsional: theme.json, textbox.png, ...
```

Letak dan aturan penamaan aset ada di [Latar dan karakter](docs/v0.1.1/scenes-and-sprites.md#letak-berkas). Nama aset **peka huruf besar-kecil** (penting di Linux) dan memakai `/` sebagai pemisah di semua platform.

## Menjalankan

```
ango [-version] [-check] [-auto] [-window] [-assets dir] script.ango|folder-game
```

| Opsi | Fungsi |
| --- | --- |
| `-check` | Compile saja, laporkan error, lalu keluar (kode keluar 1 bila ada error) |
| `-auto` | Tanpa prompt: lanjut otomatis dan pilih opsi pertama |
| `-window` | Main di window grafis (bawaan: terminal) |
| `-assets dir` | Folder aset (bawaan: `<folder game>/assets`) |
| `-version` | Cetak versi dan platform |

Kontrol di window: klik atau Spasi/Enter untuk lanjut, klik atau tombol `1`-`9` untuk memilih, `Esc` membuka menu jeda bila menu diaktifkan. Tampilan diatur lewat [tema GUI](docs/v0.1.1/gui-theme.md).

## Launcher

`ango-launcher` menampilkan daftar proyek di kiri dan tombol **Jalankan**, **Periksa Skrip**, **Uji Otomatis**, **Buka Folder**, **Tambah Folder**, serta **Segarkan** di kanan, dengan panel log untuk keluaran semua perintah.

| Platform | Cara membuka |
| --- | --- |
| Linux | `./ango.sh [folder ...]` |
| Windows | `ango.bat [folder ...]` |

Folder proyek tidak terkunci pada satu tempat; urutan pencariannya: argumen, `ANGO_PROJECTS`, konfigurasi tersimpan, lalu folder `projects/` di samping launcher. Detail, lokasi konfigurasi per OS, dan pencarian binary `ango` ada di [Launcher](docs/v0.1.1/launcher.md).

## Platform yang didukung

| | Linux (amd64) | Windows (amd64) |
| --- | --- | --- |
| Terminal dan `-check` | ✅ | ✅ |
| Window Ebitengine | ✅ | ✅ |
| Launcher | ✅ | ✅ |
| Paket rilis | `.tar.gz` | `.zip` |
| Tes otomatis di CI | ✅ | ✅ |

Build memakai `CGO_ENABLED=0`, sehingga satu mesin dapat membuat paket untuk kedua OS. macOS, Android, dan Web adalah target berikutnya dan belum didukung. Persyaratan, lintas-kompilasi, aturan menulis cerita yang portabel, dan pemecahan masalah: [Platform](docs/v0.1.1/platforms.md).

## Membangun dan menguji

```bash
make help                 # daftar target
make build                # bin/ango dan bin/ango-launcher
make test                 # semua tes
make check                # gofmt + go vet + go test
make cross                # compile + vet untuk Linux dan Windows
make dist                 # paket rilis untuk OS saat ini
make dist GOOS=windows    # paket .zip Windows, dari OS apa pun
make dist-all             # Linux dan Windows sekaligus
```

Variabel yang dapat diubah: `VERSION`, `GOOS`, `GOARCH`, `STORY` (contoh yang dibundel, bawaan `examples/intro`), `PROJECTS`. Contoh: `make dist VERSION=0.1.2 STORY=examples/basic`.

Pengujian berlapis: tes unit per paket, tes integrasi lintas paket dan lintas platform di [`test/`](test), dan setiap folder di `examples/` otomatis dimainkan sampai tamat. Rinciannya di [Pengujian](docs/v0.1.1/testing.md).

## Struktur repositori

| Folder | Isi |
| --- | --- |
| `engine/script` | lexer, parser, dan AST |
| `compiler` | AST menjadi bytecode (`Program`) |
| `engine/game` | VM dan state permainan |
| `engine/value` | tipe nilai |
| `backend/ebitengine` | tampilan window: gambar, animasi, tema GUI, menu |
| `cmd/ango` | CLI dan adapter VM ke backend |
| `cmd/ango-launcher` | launcher grafis dan lapisan platform |
| `test` | tes integrasi lintas paket dan platform |
| `examples` | cerita contoh (`basic`, `intro`) |
| `scripts` | skrip pembungkus Linux dan Windows |
| `docs` | dokumentasi |

Compiler tidak mengenal backend, dan backend tidak mengenal compiler atau VM; adapter di `cmd/ango` menjembatani keduanya. Lihat [Arsitektur](docs/v0.1.1/architecture.md).

## Dokumentasi

Mulai dari [docs/v0.1.1/README.md](docs/v0.1.1/README.md):
[Memulai](docs/v0.1.1/getting-started.md) ·
[Referensi bahasa](docs/v0.1.1/language-reference.md) ·
[Namespace](docs/v0.1.1/namespaces.md) ·
[Latar dan karakter](docs/v0.1.1/scenes-and-sprites.md) ·
[Transisi](docs/v0.1.1/transitions.md) ·
[Tema GUI](docs/v0.1.1/gui-theme.md) ·
[Launcher](docs/v0.1.1/launcher.md) ·
[Platform](docs/v0.1.1/platforms.md) ·
[Arsitektur](docs/v0.1.1/architecture.md) ·
[Pengujian](docs/v0.1.1/testing.md) ·
[Manajemen proyek](docs/v0.1.1/project-management.md)

## Keterbatasan saat ini

- Audio belum ada (folder `audio/` belum dipakai).
- Simpan dan muat permainan belum tersedia; tombol "Lanjut" di menu belum berfungsi. (State VM sudah dapat disalin dan dipulihkan; antarmukanya yang belum.)
- Layar pengaturan, serta "Menu Utama" dan "Mulai ulang" dari menu jeda.
- Transisi selain `fade`; perpindahan posisi belum menggeser.
- Bingkai sembilan-irisan untuk gambar GUI.
- Mode debug (`ango debug`, overlay variabel) dan berkas bytecode terkompilasi.
- Pemeriksaan tipe (mis. `if 1:`) terjadi saat dijalankan, bukan oleh `-check`.
- Paket rilis Windows belum ditandatangani sehingga SmartScreen dapat memberi peringatan.
- macOS, Android, dan Web belum didukung.

## Berkontribusi

Lihat [CONTRIBUTING.md](CONTRIBUTING.md) untuk standar kode, aturan lintas-platform, dan alur perubahan. Pekerjaan dikelola di papan kanban GitHub Projects ([cara kerja](docs/v0.1.1/project-management.md)).

## Lisensi

[MIT](LICENSE) © 2026 SUGI
