# Dokumentasi Ango v0.1.1

Ango adalah engine visual novel berbasis Go dengan bahasa skrip sendiri (`.ango`). Dokumen ini adalah titik awal untuk semua panduan dan referensi.

## Untuk pemain dan penulis cerita

| Dokumen | Isi |
| --- | --- |
| [Memulai](getting-started.md) | Pasang, jalankan contoh, dan buat cerita pertama |
| [Referensi bahasa](language-reference.md) | Sintaks, tipe, operator, alur kontrol, dan pesan error |
| [Namespace](namespaces.md) | Memecah cerita ke banyak berkas |
| [Latar dan karakter](scenes-and-sprites.md) | `scene`, `show`, `hide`, posisi, `flip`, dan letak aset |
| [Transisi](transitions.md) | Klausa `with` dan efek yang tersedia |
| [Tema GUI](gui-theme.md) | `theme.json`, kotak dialog, tombol, dan menu |
| [Launcher](launcher.md) | Peluncur grafis dan pencarian folder proyek |

## Untuk pengembang

| Dokumen | Isi |
| --- | --- |
| [Platform](platforms.md) | Linux dan Windows: kebutuhan, build, lintas-kompilasi, pemecahan masalah |
| [Arsitektur](architecture.md) | Alur data dari sumber hingga layar, batas antarpaket |
| [Pengujian](testing.md) | Jenis tes, cara menjalankan, dan cara menambah tes |
| [Manajemen proyek](project-management.md) | Papan kanban GitHub Projects, kolom, WIP, dan Definition of Done |
| [CONTRIBUTING](../../CONTRIBUTING.md) | Alur kerja kontribusi dan standar kode |
| [CHANGELOG](../../CHANGELOG.md) | Riwayat perubahan |

## Konvensi dokumen

- Contoh perintah memakai `make`; setiap target punya padanan `go` langsung yang dijelaskan di [Platform](platforms.md).
- Blok kode ` ```ango ` adalah skrip Ango yang valid kecuali ditandai sebagai contoh error.
- Jalur berkas ditulis dengan `/`. Di Windows, `\` juga diterima oleh shell, tetapi nama aset di dalam skrip selalu memakai `/`.
