# Namespace

> Berlaku sejak v0.1. Bagian dari [dokumentasi Ango](README.md).

Namespace memungkinkan cerita yang dipecah ke banyak file memakai nama label yang sama tanpa bentrok, misalnya `opening` di setiap bab.

Fitur ini opsional dan eksplisit. File tanpa `namespace` berperilaku persis seperti sebelumnya.

## Sintaks

```ango
namespace chapter1

label opening
"Ini bab 1."
jump ruang_kelas

label ruang_kelas
"Bel berbunyi. Kelas dimulai."
jump chapter2.opening
```

`namespace <nama>` adalah direktif tingkat file:

- Harus muncul **sebelum** `default` dan `label` pertama.
- Hanya boleh **sekali** per file.
- Nama berupa satu identifier tanpa titik (`chapter1`, bukan `part.one`).
- `namespace` adalah kata cadangan, jadi tidak bisa dipakai sebagai nama variabel atau label.

Satu file berisi satu namespace. Semua label di file itu menjadi milik namespace tersebut.

## Aturan resolusi

| Penulisan                | Artinya                                                             |
| ------------------------ | ------------------------------------------------------------------- |
| `jump ns.label`          | Selalu merujuk ke `label` di namespace `ns`, dari file mana pun.    |
| `jump label` (bare)      | Dicari dulu di namespace file saat ini, lalu di label global.       |

Label ber-namespace **tidak** terdaftar sebagai nama bare di global. Dari luar namespace, `jump opening` tidak akan menemukan `chapter1.opening`; harus ditulis lengkap. Aturan yang sama berlaku untuk `call`.

## Label `start`

`start` (titik masuk) selalu global dan harus unik. Taruh di file root yang **tidak** memakai `namespace`:

```ango
# main.ango
label start
jump chapter1.opening
```

`start` di dalam file ber-namespace terdaftar sebagai `ns.start`, sehingga compiler tetap melaporkan `missing entry label "start"`.

## Contoh folder cerita

```
examples/intro/
├── main.ango       # tanpa namespace, berisi `start`
├── chapter1.ango   # namespace chapter1
└── chapter2.ango   # namespace chapter2
```

```ango
# chapter2.ango
namespace chapter2

label opening
"Ini bab 2."
end
```

Jalankan:

```bash
ango -check examples/intro
ango -auto examples/intro
```

## Yang tidak dipengaruhi namespace

- **Variabel** (`default`) tetap global dan harus unik di seluruh cerita.
- **Label global** (dari file tanpa `namespace`) tetap harus unik secara global.
- Label yang sama di namespace berbeda (`a.intro` dan `b.intro`) boleh. Label yang sama di namespace yang sama, walau dari file berbeda, adalah error duplikat.

## Pesan error umum

| Pesan                                                              | Penyebab                                                    |
| ------------------------------------------------------------------ | ----------------------------------------------------------- |
| `"namespace" must be the first declaration in the file, and may appear only once` | `namespace` ganda, atau muncul setelah `default`/`label`. |
| `expected namespace name (identifier)`                             | `namespace` tanpa nama, atau nama berupa kata cadangan.     |
| `expected label name after "."`                                    | Referensi seperti `jump chapter1.`.                         |
| `label "chapter1.opening" already defined at file:line`            | Dua file memakai namespace dan nama label yang sama.        |
| `undefined label "opening"`                                        | Memanggil label ber-namespace tanpa prefix dari luar namespace-nya. |

## Alasan desain

Namespace sengaja diambil dari direktif di dalam file, bukan dari lokasi file. Dengan begitu memindahkan atau mengganti nama file tidak diam-diam mengubah referensi yang memakainya.
