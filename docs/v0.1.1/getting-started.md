# Memulai

Panduan ini membawa Anda dari nol sampai menjalankan cerita pertama di terminal dan di window.

## Prasyarat

| | Linux | Windows |
| --- | --- | --- |
| Go | sesuai `go.mod` | sesuai `go.mod` |
| Library sistem | OpenGL dan X11 (lihat [Platform](platforms.md#linux)) | tidak ada |
| `make` | opsional | opsional, ada `scripts/build.ps1` |

Pengguna paket rilis (`.tar.gz` / `.zip`) tidak perlu Go sama sekali.

## Dari paket rilis

Linux:

```bash
tar -xzf ango-0.1.1-linux-amd64.tar.gz
cd ango-0.1.1-linux-amd64
./play.sh          # memainkan contoh di window
./ango.sh          # membuka launcher
```

Windows: ekstrak `ango-0.1.1-windows-amd64.zip`, lalu klik dua kali `play.bat` (contoh) atau `ango.bat` (launcher).

## Dari kode sumber

```bash
make build                       # bin/ango dan bin/ango-launcher
./bin/ango -window examples/intro
```

Tanpa `make`:

```bash
go run ./cmd/ango -window examples/intro
```

Di Windows PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\build.ps1
.\bin\ango.exe -window examples\intro
```

## Cerita pertama

Buat folder `halo/` berisi satu berkas `main.ango`:

```ango
default nama = "Asep"

label start
"Pagi yang tenang."
Asep: "Halo, aku [nama]."

choice:
    "Sapa balik":
        Asep: "Senang bertemu denganmu!"
    "Diam saja":
        "Asep mengangkat bahu."
end
```

Periksa tanpa menjalankan, lalu mainkan:

```bash
ango -check halo        # melaporkan semua error beserta berkas dan baris
ango halo               # mode terminal, pilihan bernomor
ango -auto halo         # tanpa prompt, selalu memilih opsi pertama
ango -window halo       # window grafis; gambar yang belum ada diganti placeholder
```

Tambahkan seni kapan saja ke `halo/assets/` ([letak aset](scenes-and-sprites.md#letak-berkas)); skrip tidak perlu diubah.

## Langkah berikutnya

- [Referensi bahasa](language-reference.md) untuk seluruh sintaks.
- [Namespace](namespaces.md) saat cerita mulai dipecah per bab.
- [Launcher](launcher.md) untuk mengelola beberapa proyek.
