# Referensi bahasa `.ango` (v0.1)

Dokumen ini adalah spesifikasi sintaks dan semantik yang berlaku untuk v0.1.x.

## Struktur berkas

- Encoding UTF-8. BOM di awal berkas diabaikan.
- Akhir baris `\n`, `\r\n`, dan `\r` diperlakukan sama, sehingga berkas yang disunting di Windows dan Linux menghasilkan program identik.
- Kolom pada pesan error dihitung per karakter (rune), bukan per byte.
- Komentar dimulai dengan `#` sampai akhir baris.
- Satu cerita adalah satu berkas atau satu folder berisi banyak berkas `.ango` (rekursif). Titik masuknya adalah label `start`, yang harus ada tepat satu kali.

## Indentasi

Indentasi hanya bermakna di dalam blok `if`, `else`, dan `choice`, dan harus **kelipatan 4 spasi**. Tab ditolak dengan pesan `tabs are not allowed in indentation; use 4 spaces per level`. Isi sebuah `label` **tidak** diberi indentasi.

## Tipe nilai dan literal

| Tipe | Literal | Catatan |
| --- | --- | --- |
| Bilangan bulat | `0`, `42` | |
| Desimal | `0.5`, `1.25` | Hasil operator `/` selalu desimal |
| String | `"teks"` | Boleh memuat `[variabel]` untuk interpolasi |
| Boolean | `true`, `false` | |

Nilai awal sebuah `default` harus berupa literal.

## Variabel

```ango
default courage = 0          # deklarasi global; wajib sebelum dipakai
set courage = courage + 1    # mengubah nilai
```

- Setiap variabel harus dideklarasikan dengan `default`. `set` pada variabel yang tidak dideklarasikan adalah error compile (`undeclared variable "x" (declare it with `default`)`).
- Variabel selalu global dan namanya unik di seluruh cerita, juga lintas [namespace](namespaces.md).
- Interpolasi: `"Keberanianku [courage]."` menyisipkan nilai variabel ke dalam teks dialog dan teks pilihan.

## Operator

| Kelompok | Operator |
| --- | --- |
| Aritmetika | `+ - * /` |
| Perbandingan | `== != < <= > >=` |
| Logika | `and or not` |

- `/` selalu menghasilkan desimal, termasuk `4 / 2`.
- `and`, `or`, `not`, dan kondisi `if` hanya menerima boolean; nilai lain bukan "truthy".
- Ketidakcocokan tipe (misalnya `"a" + 1` atau `if 1:`) dilaporkan **saat dijalankan**, bukan oleh `-check`. Uji semua jalur cerita dengan [tes integrasi](testing.md).

## Dialog dan narasi

```ango
"Narasi tanpa pembicara."
Asep: "Dialog oleh pembicara bertipe identifier."
"Bu Rina": "Nama dengan spasi atau kata cadangan ditulis sebagai string."
@id("intro_001")
Asep: "Anotasi @id memberi ID stabil pada baris berikutnya."
```

Pembicara kosong berarti narasi. Pembicara berupa identifier atau string.

## Percabangan

```ango
if courage > 1 and not sudah_tahu:
    Asep: "Aku berani."
else:
    "Masih ragu."
end
```

Blok `if` diakhiri `end` (atau `else:` lalu `end`).

## Pilihan

```ango
choice:
    "Ambil kunci" if not found_key:
        set found_key = true
        jump hallway_loop
    "Pulang saja":
        jump finale
```

Setiap opsi punya teks (boleh berinterpolasi), syarat `if` opsional, dan blok isi. Opsi yang syaratnya salah tidak ditampilkan; indeks yang diterima `VM.Choose` mengacu pada opsi yang aktif.

## Alur

| Perintah | Fungsi |
| --- | --- |
| `label nama` | Mendefinisikan label |
| `jump nama` | Lompat tanpa kembali |
| `call nama` | Lompat, lalu kembali setelah `return` |
| `return` | Kembali ke pemanggil `call` |
| `end` | Mengakhiri cerita |

Label yang tidak diakhiri `jump`, `return`, atau `end` berakhir otomatis dengan `end`. Referensi lintas berkas memakai `namespace.label`; lihat [Namespace](namespaces.md).

## Kata cadangan

`and or not label scene show hide choice jump call return set if else default namespace flip end at with true false`

Kata ini tidak boleh menjadi nama variabel atau label. Pembicara yang namanya kata cadangan ditulis sebagai string.

## Perintah tampilan

`scene`, `show`, `hide`, dan klausa `with` dijelaskan di [Latar dan karakter](scenes-and-sprites.md) dan [Transisi](transitions.md). Di mode terminal perintah ini dibaca tetapi tidak menggambar apa pun.

## Pesan error

Semua error memuat berkas, baris, dan kolom, dan `-check` melaporkan semua error compile sekaligus.

| Pesan | Penyebab |
| --- | --- |
| `tabs are not allowed in indentation; use 4 spaces per level` | Tab di indentasi |
| `unterminated string` | String tidak ditutup |
| `expected a statement, found "..."` | Kata di tempat yang salah |
| `undeclared variable "x" (declare it with `default`)` | `set` atau pemakaian variabel tanpa `default` |
| `undefined label "x"` | `jump` atau `call` ke label yang tidak ada |
| `label "x" already defined at file:line` | Label ganda |
| `missing entry label "start"` | Cerita tidak punya titik masuk |
| `expected transition duration (number), found "..."` | Durasi transisi bukan angka atau negatif |
