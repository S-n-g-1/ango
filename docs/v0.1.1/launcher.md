# Launcher

> Bagian dari [dokumentasi Ango](README.md).

`ango-launcher` adalah peluncur grafis bergaya Ren'Py. Ia mendaftar proyek dari satu atau beberapa folder lalu menjalankan binary `ango` pada proyek terpilih, dengan keluarannya ditampilkan di panel log.

## Membuka

| Platform | Perintah |
| --- | --- |
| Linux | `./ango.sh` (paket rilis) atau `./bin/ango-launcher` |
| Windows | `ango.bat` (paket rilis) atau `bin\ango-launcher.exe` |

Argumen berupa folder proyek: `./ango.sh ~/games`. Dari kode sumber: `make launcher-run`.

## Tombol

| Tombol | Fungsi |
| --- | --- |
| Jalankan | Memainkan proyek di window (`ango -window`) |
| Periksa Skrip | Menjalankan `ango -check` |
| Uji Otomatis | Menjalankan `ango -auto` |
| Buka Folder | Membuka folder proyek di pengelola berkas sistem |
| Tambah Folder | Membuka pemilih folder dan menyimpan pilihan |
| Segarkan | Memindai ulang folder |

Klik dua kali sebuah proyek untuk menjalankannya.

## Folder proyek

Urutan pencarian (yang pertama tersedia dipakai):

1. Folder pada argumen baris perintah.
2. Variabel lingkungan `ANGO_PROJECTS`, dipisahkan `:` di Linux/macOS atau `;` di Windows.
3. Daftar tersimpan di berkas konfigurasi.
4. Folder `projects/` di samping launcher.

Sebuah folder boleh berisi banyak game (setiap subfolder yang memuat `.ango` jadi satu proyek) atau menjadi satu game langsung. Subfolder tersembunyi (diawali `.`) dilewati; tautan simbolik diikuti.

## Konfigurasi

| Platform | Lokasi `launcher.json` |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/ango/` atau `~/.config/ango/` |
| Windows | `%AppData%\ango\` |

```json
{ "roots": ["/home/kamu/games", "/mnt/data/vn"], "ango": "/opt/ango/ango" }
```

## Mencari binary `ango`

Urutan: flag `-ango <path>`, variabel `ANGO_BIN`, field `ango` di konfigurasi, binary di folder yang sama dengan launcher (`ango` atau `ango.exe`), lalu `PATH`.

## Dialog pemilih folder

| Platform | Mekanisme |
| --- | --- |
| Linux | `zenity`, lalu `kdialog` (pasang salah satu); tanpa keduanya gunakan argumen atau edit konfigurasi |
| Windows | Dialog folder bawaan Windows lewat PowerShell |
| macOS | `osascript` (belum diuji) |
