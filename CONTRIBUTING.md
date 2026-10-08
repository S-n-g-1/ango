# Kontribusi

Terima kasih sudah mau berkontribusi pada Ango. Dokumen ini merangkum cara menyiapkan lingkungan, standar kode, dan alur perubahan.

## Menyiapkan

```bash
git clone <url-repositori> ango && cd ango
make build        # bin/ango dan bin/ango-launcher
make check        # gofmt + go vet + go test
```

Versi Go mengikuti `go.mod`. Kebutuhan per platform ada di [docs/v0.1.1/platforms.md](docs/v0.1.1/platforms.md). Di Windows tanpa `make`, gunakan `go build`, `go vet`, dan `go test ./...` secara langsung, atau `scripts/build.ps1`.

## Alur perubahan

1. Buat cabang dari `main`.
2. Tulis perubahan beserta tesnya. Bug diperbaiki dengan tes yang gagal lebih dulu.
3. Jalankan `make check` dan `make cross`. Perubahan yang menyentuh path, proses, atau berkas harus lolos di **kedua** OS.
4. Perbarui dokumentasi di `docs/` dan tambahkan entri di `CHANGELOG.md` bagian atas.
5. Buka pull request. CI menjalankan seluruh tes di Linux dan Windows.

## Standar kode

- Format dengan `gofmt` (`make fmt`); CI menolak kode yang belum diformat.
- `go vet` harus bersih.
- Komentar dokumentasi menjelaskan **mengapa**, bukan mengulang apa yang sudah jelas dari kode.
- Pesan error untuk penulis cerita memuat berkas, baris, dan petunjuk perbaikan.

### Aturan lintas-platform

- Path di sistem berkas: `path/filepath`. Path di dalam `fs.FS` dan nama aset di skrip: `path`, dengan `/`.
- Jangan memanggil program yang hanya ada di satu OS di luar `cmd/ango-launcher/platform*.go`.
- Fungsi yang berbeda per OS menerima nama OS sebagai parameter agar teruji dari satu mesin.
- Di tes, jangan mengandalkan `HOME`, `XDG_CONFIG_HOME`, atau pemisah `:`; atur juga `USERPROFILE`, `APPDATA`, dan `os.PathListSeparator`.
- Jangan memakai `cgo`. Build rilis memakai `CGO_ENABLED=0`.

## Pengujian

Panduan lengkap: [docs/v0.1.1/testing.md](docs/v0.1.1/testing.md). Singkatnya: tes unit di samping kode, tes lintas paket di `test/`, dan cerita contoh di `examples/` otomatis ikut diuji.

## Perubahan bahasa

Lexer v0.1 dibekukan untuk fitur baru. Perubahan sintaks perlu diskusi lebih dulu dan mengikuti langkah di [arsitektur](docs/v0.1.1/architecture.md#menambah-fitur-bahasa).

## Lisensi

Dengan berkontribusi, Anda setuju kontribusi Anda dilisensikan di bawah [MIT](LICENSE).
