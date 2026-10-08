# Arsitektur

> Bagian dari [dokumentasi Ango](README.md).

## Alur data

```
 .ango  ──►  engine/script  ──►  compiler  ──►  engine/game (VM)  ──►  Event  ──►  frontend
 sumber      lexer, parser       bytecode        GameState              say/scene/   terminal
             AST                 Program                                show/hide/   atau
                                                                        choice/end   backend/ebitengine
```

| Paket | Tanggung jawab |
| --- | --- |
| `engine/script` | Lexer, parser, AST, dan posisi sumber (`Span`). Tidak mengenal compiler. |
| `compiler` | AST menjadi `Program` (kode, konstanta, tabel say/scene/sprite/choice, label). Menjalankan validasi semantik dan melaporkan semua error sekaligus. |
| `engine/value` | Tipe nilai dinamis (nil, bool, int, float, string). |
| `engine/game` | `VM` yang mengeksekusi `Program` dan menghasilkan `Event`; `GameState` sebagai sumber kebenaran tunggal. |
| `backend/ebitengine` | Window: gambar, animasi transisi, tema GUI, menu. Hanya melihat state presentasi. |
| `cmd/ango` | CLI, adapter VM ke backend, mode terminal. |
| `cmd/ango-launcher` | Launcher grafis dan lapisan platform (`platform*.go`). |
| `test` | Tes integrasi lintas paket dan lintas platform. |

## Prinsip desain

- **`GameState` adalah sumber kebenaran tunggal.** Isinya hanya data biasa (PC, variabel, call stack, latar, sprite, pilihan tertunda) sehingga dapat disalin dalam (`Clone`) dan diserialisasi untuk simpan dan rollback.
- **Backend tidak mengenal compiler maupun VM.** Adapter di `cmd/ango` menerjemahkan `Event` ke perintah backend, sehingga backend lain (web, mobile) dapat ditambahkan tanpa menyentuh bahasa.
- **Aset dibaca lewat `fs.FS`.** Seluruh nama aset memakai `/`; tidak ada `filepath` di jalur pemuatan aset, sehingga perilakunya sama di Linux dan Windows.
- **Error selalu berposisi.** Setiap instruksi menyimpan `Span` sumbernya, jadi error runtime pun menunjuk berkas dan baris.
- **Bahasa dan backend terpisah untuk efek visual.** Nama transisi tidak divalidasi oleh compiler; backend yang menafsirkannya ([Transisi](transitions.md)).

## Kode khusus platform

Semua perbedaan Linux/Windows dikurung di satu tempat:

| Berkas | Isi |
| --- | --- |
| `cmd/ango-launcher/platform.go` | Nama binary, pembuka folder, pemilih folder per OS |
| `cmd/ango-launcher/platform_windows.go` | Menyembunyikan jendela konsol proses anak |
| `cmd/ango-launcher/platform_other.go` | Implementasi kosong untuk OS lain |

Fungsi-fungsi ini menerima nama OS sebagai parameter sehingga dapat diuji untuk semua platform dari satu mesin.

## Menambah fitur bahasa

1. Token baru di `engine/script/token.go` dan lexer bila perlu; ingat bahwa lexer v0.1 dibekukan untuk fitur baru.
2. Node AST di `ast.go`, parsing di `parser.go`, dan `dump.go` untuk keluaran debug.
3. Validasi dan emisi di `compiler/`, dengan tes tabel di `compiler_test.go`.
4. Eksekusi di `engine/game/vm.go`; tambahkan `EventKind` bila frontend perlu tahu.
5. Tampilan di `backend/ebitengine` dan adapter di `cmd/ango`.
6. Dokumentasi di `docs/` dan entri di `CHANGELOG.md`.
