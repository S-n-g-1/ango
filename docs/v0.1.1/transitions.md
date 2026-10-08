# Transisi

> Bagian dari [dokumentasi Ango](README.md).

Transisi adalah efek visual opsional saat latar atau sprite berganti. Tanpa transisi, perubahan terjadi seketika (hard cut).

## Sintaks

```
with <nama> <durasi>
```

Klausa ini ditaruh di akhir baris `scene`, `show`, atau `hide`:

```ango
scene "Kelas" with fade 0.8
show asep "Smile" at right flip with fade 0.4
hide asep with fade 0.4
```

- `<nama>` adalah satu identifier (`fade`). Kata cadangan seperti `at`, `flip`, atau `with` tidak bisa dipakai sebagai nama.
- `<durasi>` adalah angka dalam detik, bulat atau desimal (`1` dan `0.5` sama-sama sah).
- Durasi negatif ditolak compiler: `transition duration must not be negative`.

## Transisi yang tersedia

| Nama   | Berlaku di              | Efek                                        |
| ------ | ----------------------- | ------------------------------------------- |
| `fade` | `scene`, `show`, `hide` | Memudar masuk atau keluar selama durasinya. |

Hanya `fade` yang dikenali backend saat ini. Pada `show` ulang untuk karakter yang sudah tampil (pindah posisi atau ganti ekspresi), `fade` memudarkan sprite di keadaan barunya; belum ada efek meluncur.

## Nama yang tidak dikenal

Compiler dan VM sengaja tidak memvalidasi nama transisi. Nama diteruskan apa adanya ke backend, dan backend yang menafsirkannya. Akibatnya:

- `with dissolve 0.5` atau salah ketik seperti `with fdae 0.5` **lolos `-check`**.
- Saat dijalankan di `-window`, backend mencatat peringatan satu kali per nama, dan perubahan terjadi tanpa efek.

Kalau transisi tidak terlihat, cek dulu ejaan namanya dan lihat log peringatan.

## Mode terminal dan mode window

Transisi hanya punya arti di `-window`. Mode terminal tidak menggambar apa pun, sehingga klausa `with` dibaca tapi tidak berefek.

## Aset

Aturan nama dan letak berkas latar dan sprite ada di [Latar dan karakter](scenes-and-sprites.md#letak-berkas).

## Menambah transisi baru

Karena bahasanya tidak perlu diubah, transisi baru cukup ditambahkan di backend:

1. Daftarkan nama baru di `backend/ebitengine/game.go`, di samping `transitionFade`.
2. Perluas `fadeDuration` (atau tambahkan fungsi serupa) supaya nama itu dikenali dan tidak memicu `warnUnknownTransition`.
3. Gambar efeknya di tempat transisi `fade` digambar sekarang: `bgAnim` untuk latar, `entering` untuk sprite yang baru tampil, dan `exiting` untuk sprite yang disembunyikan.

Tipe `Transition` di `backend/ebitengine/session.go` hanya membawa `Name` dan `Duration`, jadi tidak perlu berubah. Kalau sebuah transisi butuh parameter tambahan (arah geser, misalnya), bentuk klausanya yang perlu dirancang ulang, dan itu menyentuh parser.
