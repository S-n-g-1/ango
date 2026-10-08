# Tema GUI dan menu

> Bagian dari [dokumentasi Ango](README.md). Berlaku untuk mode `-window`.

Tampilan kotak dialog, plat nama, tombol pilihan, dan menu diatur oleh berkas opsional `assets/gui/theme.json` di dalam folder cerita. **Semua field opsional**: tanpa berkas ini, atau dengan berkas yang rusak, backend memakai nilai bawaan dan permainan tetap berjalan.

## Contoh

Contoh lengkap seluruh field tersedia di `examples/intro/assets/gui/theme.example.json`.

```json
{
  "font_size": 26,
  "colors": { "text": "#FFFFFF", "textbox": "#000000BE", "button_hover": "#46468CF0" },
  "textbox": { "x": 40, "y": 520, "w": 1200, "h": 170 },
  "menu": { "enabled": true, "title": "Judul Game", "start": "Mulai", "quit": "Keluar" }
}
```

## Referensi field

| Field | Fungsi |
| --- | --- |
| `font_size`, `line_height`, `max_lines` | Ukuran huruf, tinggi baris, dan jumlah baris maksimum di kotak dialog |
| `colors.text`, `textbox`, `plate`, `button`, `button_hover` | Warna `#RRGGBB` atau `#RRGGBBAA` |
| `textbox` | Posisi dan ukuran kotak dialog (`x`, `y`, `w`, `h`) |
| `text_pad_x`, `text_pad_y` | Jarak teks dari tepi kotak |
| `plate` | Plat nama: `offset_x`, `height`, `inset` |
| `button` | Tombol pilihan: `w`, `h`, `gap`, `pad_x` |
| `menu` | `enabled`, `title`, `start`, `resume`, `quit`, `pause`, `top`, `title_scale` |

Koordinat memakai ruang layar logis 1280×720, bukan piksel jendela, sehingga tema tetap sama saat jendela diubah ukurannya.

## Gambar opsional

Letakkan di `assets/gui/` dengan ekstensi apa saja yang dikenali (`.webp`, `.png`, `.jpg`, `.jpeg`, `.svg`):

`textbox`, `namebox`, `button`, `button_hover`, `menu_bg`, `title`

Gambar saat ini diregangkan ke ukuran kotaknya (belum ada bingkai sembilan-irisan).

## Menu

Dengan `menu.enabled: true`, menu utama tampil saat permainan dibuka dan menu jeda terbuka lewat `Esc`. Tombol "Lanjut" belum berfungsi karena simpan/muat belum tersedia di v0.1.1.

## Kontrol di window

| Aksi | Input |
| --- | --- |
| Lanjut dialog | Klik, Spasi, atau Enter |
| Memilih opsi | Klik, atau tombol `1`-`9` |
| Menu jeda | `Esc` (bila menu diaktifkan) |
