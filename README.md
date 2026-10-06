# Ango v0.1

Ango adalah engine visual novel yang ditulis dengan Go. Ceritanya ditulis di berkas teks `.ango` yang ringkas, lalu dikompilasi ke bytecode dan dimainkan oleh VM, baik di terminal maupun di window (Ebitengine).

> **Status: v0.1, tahap awal.** Bahasa dan API masih bisa berubah. Daftar yang belum ada ada di bagian [Belum ada di v0.1](#belum-ada-di-v01).

## Fitur

- **Bahasa skrip `.ango`**: dialog, narasi, variabel, percabangan, pilihan, dan lompatan antar label.
- **Cerita banyak berkas**: satu folder = satu game, dengan `namespace` opsional per berkas supaya nama label tidak bentrok.
- **Compiler dengan pesan error berposisi**: `ango -check` melaporkan semua kesalahan sekaligus beserta berkas dan barisnya.
- **Dua mode main**: terminal dan window grafis.
- **Latar dan karakter**: `scene`, `show`, `hide`, tiga posisi (`left`, `center`, `right`), `flip` untuk membalik sprite, dan transisi `fade`.
- **Banyak karakter sekaligus** di layar, dengan ekspresi per karakter.
- **Tema GUI** (`gui/theme.json`) untuk kotak dialog, plat nama, dan tombol, plus menu utama dan menu jeda yang bisa diaktifkan.
- **Launcher grafis** (`ango-launcher`, dibuka lewat `ango.sh`) untuk memilih proyek lalu menjalankan, memeriksa, atau menguji otomatis.
- **Format gambar**: WebP, PNG, JPEG, dan SVG.

## Mulai cepat

Butuh Go (sesuai `go.mod`) dan pustaka sistem untuk Ebitengine (OpenGL dan X11 di Linux).

```bash
make build                      # bin/ango dan bin/ango-launcher
./bin/ango -window examples/intro
```

Atau lewat launcher:

```bash
make launcher-run               # membuka launcher dengan folder examples/
```

Tanpa `make`:

```bash
go run ./cmd/ango -window examples/intro
go run ./cmd/ango -check examples/intro
```

## Struktur game

Satu game adalah satu folder:

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
    └── gui/           # opsional: theme.json, textbox.png, button.png, ...
```

Semua berkas `.ango` di folder dimuat sekaligus. Cerita dimulai dari label `start`, yang harus ada dan unik.

## Bahasa Ango

### Dasar

```ango
# komentar
default player_name = "Asep"      # variabel global; nilai awal harus literal
default courage = 0
default sudah_tahu = false

label start                        # isi label TIDAK diberi indentasi
scene "Kelas" with fade 0.8
"Narasi tanpa pembicara."
Asep: "Halo, [player_name]."       # [nama] menyisipkan nilai variabel
"Bu Rina": "Nama dengan spasi ditulis sebagai string."
set courage = courage + 1
jump chapter1.opening
```

- Tipe nilai: bilangan bulat, desimal, string, dan boolean (`true`/`false`).
- Operator: `+ - * /`, `== != < <= > >=`, dan `and or not`.
- Indentasi hanya dipakai di dalam blok `if`, `else`, dan `choice` (kelipatan 4 spasi).
- Kata cadangan: `and or not label scene show hide choice jump call return set if else default namespace flip end at with true false`. Kata ini tidak bisa jadi nama variabel atau label, dan pembicara bernama kata cadangan harus ditulis sebagai string.

### Percabangan dan pilihan

```ango
if courage > 1 and not sudah_tahu:
    Asep: "Aku berani."
else:
    "Masih ragu."

choice:
    "Buka kertasnya" if courage >= 0:
        set sudah_tahu = true
        jump surat
    "Biarkan dulu":
        jump pelajaran
```

Opsi pilihan boleh punya syarat (`if ...`). Opsi yang syaratnya salah tidak ditampilkan.

### Alur: `jump`, `call`, `return`, `end`

```ango
call tampilkan_judul     # lompat dan kembali setelah `return`
jump finale              # lompat tanpa kembali
end                      # akhiri cerita
```

Label yang tidak diakhiri `jump`, `return`, atau `end` otomatis berakhir dengan `end`.

### Namespace

Untuk cerita yang dipecah ke banyak berkas, `namespace` di baris pertama sebuah berkas membuat semua labelnya milik namespace itu:

```ango
namespace chapter1

label opening
"Ini bab 1."
jump ruang_kelas            # label di berkas yang sama: cukup nama pendek
...
jump chapter2.opening       # dari berkas lain: tulis lengkap namespace.label
```

Aturan singkat: `namespace` opsional, harus jadi deklarasi pertama dan hanya sekali per berkas, `start` tetap global, dan variabel (`default`) tetap global. Detail ada di [docs/namespace.md](docs/namespace.md).

### Latar, karakter, dan transisi

```ango
scene "Kelas" with fade 0.8
show asep "Calm" at left with fade 0.4
show dina "Smile" at right flip with fade 0.4
hide dina with fade 0.4
```

- `at left | center | right`. Bawaannya `center`.
- `flip` membalik sprite secara horizontal. Konvensi aset: sprite digambar menghadap kanan, jadi karakter di sisi kanan layar diberi `flip`.
- `show` pada karakter yang sudah tampil memindahkannya atau mengganti ekspresinya, tidak membuat salinan.
- Transisi: `with <nama> <detik>`. Saat ini hanya `fade` yang dikenali. Detail di [docs/show.md](docs/show.md) dan [docs/transitions.md](docs/transitions.md).

### Letak aset

| Perintah | Dicari di (`assets/`) |
| --- | --- |
| `scene "Kelas"` | `backgrounds/Kelas.*` |
| `show asep "Calm"` | `characters/asep_Calm.*`, lalu `characters/asep/Calm.*`, lalu `characters/asep.*` |

Ekstensi dicoba berurutan: `.webp`, `.png`, `.jpg`, `.jpeg`, `.svg`. Nama berkas peka huruf besar-kecil di Linux. Aset yang tidak ditemukan diganti placeholder (blok warna berlabel), jadi skrip bisa dijalankan sebelum seni siap.

## Menjalankan

```
ango [-check] [-auto] [-window] [-assets dir] script.ango|folder-game
```

| Opsi | Fungsi |
| --- | --- |
| `-check` | Compile saja, laporkan error, lalu keluar |
| `-auto` | Tanpa prompt: lanjut otomatis dan pilih opsi pertama |
| `-window` | Main di window grafis (bawaan: terminal) |
| `-assets dir` | Folder aset (bawaan: `<folder game>/assets`) |

Kontrol di window: klik atau Spasi/Enter untuk lanjut, klik atau tombol `1`-`9` untuk memilih, `Esc` membuka menu jeda bila menu diaktifkan.

## Tema GUI dan menu utama

Letakkan `assets/gui/theme.json` di game. Semua field opsional; tanpa berkas ini tampilan sama seperti bawaan. Contoh lengkap ada di `theme.example.json`.

```json
{
  "font_size": 26,
  "colors": { "text": "#FFFFFF", "textbox": "#000000BE", "button_hover": "#46468CF0" },
  "textbox": { "x": 40, "y": 520, "w": 1200, "h": 170 },
  "menu": { "enabled": true, "title": "Judul Game", "start": "Mulai", "quit": "Keluar" }
}
```

- Warna: `#RRGGBB` atau `#RRGGBBAA`.
- Gambar opsional di `assets/gui/`: `textbox`, `namebox`, `button`, `button_hover`, `menu_bg`, `title` (ekstensi apa saja yang dikenali). Gambar saat ini diregangkan ke ukuran kotaknya.
- `menu.enabled: true` menampilkan menu utama saat game dibuka dan menu jeda lewat `Esc`.

## Launcher

`ango.sh` membuka jendela launcher: daftar proyek di kiri, tombol **Jalankan**, **Periksa Skrip**, **Uji Otomatis**, **Buka Folder**, **Tambah Folder**, dan **Segarkan** di kanan, serta panel log untuk keluaran semua perintah. Klik dua kali sebuah proyek untuk menjalankannya.

Folder proyek tidak terkunci pada satu tempat. Urutan pencariannya (yang pertama ada yang dipakai):

1. Folder di argumen: `./ango.sh ~/games ~/lain`
2. Variabel lingkungan `ANGO_PROJECTS` (dipisah `:`)
3. Daftar tersimpan di `~/.config/ango/launcher.json`
4. Folder `projects/` di samping launcher

Folder boleh berupa folder yang berisi banyak game (tiap sub-folder yang punya `.ango` jadi satu proyek) atau satu game langsung. Tombol **Tambah Folder** membuka pemilih folder sistem (butuh `zenity` atau `kdialog`) dan menyimpannya ke konfigurasi. Tanpa keduanya, edit konfigurasi atau pakai argumen.

```json
{ "roots": ["/home/kamu/games", "/mnt/data/vn"], "ango": "/opt/ango/ango" }
```

Lokasi binary `ango` dicari lewat `-ango <path>`, env `ANGO_BIN`, field `ango` di konfigurasi, folder yang sama dengan launcher, lalu `PATH`.

## Membangun

```bash
make help        # daftar target
make build       # bin/ango dan bin/ango-launcher
make test        # go test ./...
make check       # go vet + go test
make dist        # dist/ango-0.1.0-linux-amd64.tar.gz
```

Paket `dist` berisi `ango`, `ango-launcher`, `ango.sh`, `projects/` (contoh game), README, dan `docs/` bila ada. Ubah `VERSION`, `STORY`, atau `PROJECTS` lewat variabel make, misalnya `make dist VERSION=0.1.1 STORY=examples/basic`.

## Struktur kode

| Folder | Isi |
| --- | --- |
| `engine/script` | lexer, parser, dan AST |
| `compiler` | AST menjadi bytecode (`Program`) |
| `engine/game` | VM dan state permainan |
| `engine/value` | tipe nilai |
| `backend/ebitengine` | tampilan window: gambar, animasi, tema GUI, menu |
| `cmd/ango` | CLI |
| `cmd/ango-launcher` | launcher grafis |

Compiler tidak mengenal backend, dan backend tidak mengenal compiler atau VM; adapter di `cmd/ango` menjembatani keduanya.

## Belum ada di v0.1

- Audio (folder `audio/` belum dipakai).
- Simpan dan muat permainan; tombol "Lanjut" di menu belum berfungsi.
- Layar pengaturan, dan "Menu Utama"/"Mulai ulang" dari menu jeda.
- Transisi selain `fade` (efek geser untuk perpindahan posisi belum ada).
- Bingkai sembilan-irisan untuk gambar GUI.
- Mode debug (`ango debug`, overlay variabel) dan berkas bytecode terkompilasi.
- Dukungan Windows dan macOS belum diuji.