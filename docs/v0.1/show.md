# Latar dan Karakter: `scene`, `show`, `hide`

Tiga perintah ini mengatur apa yang tampil di layar. Di mode terminal mereka dibaca tapi tidak menggambar apa pun; efeknya hanya terlihat di `-window`.

## Ringkasan sintaks

```ango
scene "<latar>" [with <transisi> <durasi>]
show <karakter> "<ekspresi>" [at <posisi>] [flip] [with <transisi> <durasi>]
hide <karakter> [with <transisi> <durasi>]
```

Urutan klausa di `show` tetap: `at`, lalu `flip`, lalu `with`. Semua klausa opsional.

## `scene`

Mengganti latar. Nama latar ditulis tanpa ekstensi:

```ango
scene "Kelas" with fade 0.8     # backgrounds/Kelas.svg
```

Kalau berkas latar tidak ditemukan, backend menggambar blok gelap berlabel `[background: <nama>]` supaya skrip tetap bisa dijalankan.

## `show`

Menampilkan karakter dengan ekspresi tertentu. Nama karakter adalah identifier, ekspresi adalah string:

```ango
show asep "Calm" at left with fade 0.4
```

### Posisi (`at`)

| Posisi   | Letak horizontal         |
| -------- | ------------------------ |
| `left`   | 20% lebar layar          |
| `center` | 50% (bawaan bila `at` dihilangkan) |
| `right`  | 80% lebar layar          |

Sprite selalu berpijak di tepi bawah layar dan berpusat pada titik horizontal itu. Nama posisi lain tidak ditolak oleh parser, tapi diperlakukan seperti `center`.

### Mencerminkan sprite (`flip`)

`flip` mencerminkan gambar secara horizontal:

```ango
show asep "Smile" at right flip with fade 0.4
```

- **Konvensi aset:** semua sprite digambar menghadap **kanan**. Karakter di sisi kanan layar diberi `flip` supaya menghadap ke tengah. Karakter di kiri tidak perlu.
- **Tidak menempel:** setiap `show` menyebut ulang semua atributnya. `show` berikutnya tanpa `flip` mengembalikan arah normal.
- **Kata cadangan:** `flip` tidak bisa dipakai sebagai nama variabel atau label.
- Flip ikut berlaku saat sprite memudar keluar lewat `hide`.

### Beberapa karakter sekaligus

Setiap karakter punya satu entri di layar. Dua `show` berurutan untuk karakter berbeda menampilkan keduanya:

```ango
show asep "Astonishment" at right flip with fade 0.4
show dina "Smile" at left with fade 0.4
Dina: "Pagi, [player_name]!"
```

Urutan `show` menentukan tumpukan gambar: karakter yang baru muncul digambar di atas yang lama.

### Memindahkan atau mengganti ekspresi

`show` pada karakter yang sudah tampil **menimpa** entri lamanya, tidak membuat salinan kedua. Jadi memindahkan Asep dari kiri ke kanan, mengganti ekspresi, atau menyalakan `flip` semuanya cukup dengan `show` ulang:

```ango
show asep "Calm" at left with fade 0.4
show asep "Fear" at right flip with fade 0.4
```

Efek perpindahannya saat ini berupa fade, bukan geser.

## `hide`

Menyembunyikan karakter berdasarkan nama:

```ango
hide dina with fade 0.4
```

## Letak berkas karakter

Backend mencari gambar sprite dalam folder `characters/` pada `assets`, dengan urutan:

1. `characters/<karakter>_<ekspresi>`
2. `characters/<karakter>/<ekspresi>`
3. `characters/<karakter>`

Setiap pola dicoba apa adanya, lalu dengan tiap ekstensi gambar yang dikenali. Pola pertama yang berhasil dipakai.

```
characters/
├── asep/
│   ├── Calm.png         # show asep "Calm"
│   └── Eye rolling.png  # show asep "Eye rolling"
└── dina.png             # satu gambar untuk semua ekspresi dina
```

Catatan:

- Nama ekspresi di skrip harus sama persis dengan nama berkas tanpa ekstensi, termasuk huruf besar-kecil. Di sistem berkas yang membedakan kapitalisasi (Linux), `"calm"` tidak sama dengan `Calm.png`.
- Spasi di nama berkas boleh, tapi nama tanpa spasi lebih mudah dikelola.
- Pola ke-3 berguna untuk karakter yang baru punya satu gambar.
- Kalau tidak ada berkas yang cocok, backend menggambar blok warna berlabel nama karakter dan ekspresi, sehingga skrip tetap jalan sebelum asetnya siap.

Folder aset diatur dengan `-assets <dir>`. Bawaannya `<folder cerita>/assets`.

## Transisi

Klausa `with` dijelaskan di `transitions.md`. Saat ini hanya `fade` yang dikenali.
