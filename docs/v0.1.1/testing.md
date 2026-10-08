# Pengujian

> Bagian dari [dokumentasi Ango](README.md).

## Menjalankan

```bash
make test               # semua tes
make test-unit          # hanya tes per paket
make test-integration   # hanya folder test/
make cover              # ringkasan cakupan
make check              # gofmt + go vet + go test
make cross              # compile dan vet untuk Linux dan Windows
```

Tanpa `make`: `go test ./...`.

## Lapisan tes

| Lokasi | Cakupan |
| --- | --- |
| `engine/script/*_test.go` | Lexer dan parser: token, indentasi, posisi error, namespace |
| `compiler/*_test.go` | Validasi semantik, emisi kode, resolusi namespace |
| `engine/game/vm_test.go` | Eksekusi VM, pilihan, dan alur dasar |
| `backend/ebitengine/*_test.go` | Pemuatan tema GUI dan perilaku bawaan |
| `cmd/ango/main_test.go` | CLI: mode `-check`, `-auto`, input pilihan, seluruh `examples/` |
| `cmd/ango-launcher/main_test.go` | Pemindaian proyek, konfigurasi, resolusi folder, helper platform |
| `test/` | Integrasi lintas paket dan lintas platform (di bawah) |

## Isi `test/`

| Berkas | Memverifikasi |
| --- | --- |
| `pipeline_test.go` | Akhir baris LF, CRLF, dan CR menghasilkan program identik; BOM diabaikan; berkas tanpa newline akhir; pesan error memuat nama berkas; cabang pilihan saling terpisah; simpan dan pulihkan state |
| `examples_test.go` | Setiap folder `examples/` dikompilasi, tamat dengan pilihan pertama **dan** dengan pilihan terakhir, dan deterministik; nama aset cocok persis termasuk huruf besar-kecil; tidak ada `\` pada nama aset |
| `helpers_test.go` | Helper: kompilasi dari string atau folder, `play` yang menjalankan VM tanpa tampilan |
| `testdata/` | Fixture skrip `.ango` |

Tes ini tidak membuka window sehingga berjalan di CI tanpa display.

## Menambah tes

**Skrip kecil dengan perilaku tertentu**: tambahkan fixture ke `test/testdata/`, lalu tulis tes yang memakai `mustCompile` dan `play`:

```go
func TestKeberanianBertambah(t *testing.T) {
	prog := mustCompile(t, readFixture(t, "keberanian.ango"))
	tr := play(t, game.New(prog), 0) // pilih opsi aktif pertama
	if !tr.has("keberanian naik") {
		t.Errorf("dialog tidak muncul: %v", tr.says())
	}
}
```

**Cerita baru**: cukup taruh folder di `examples/`; tes contoh mengambilnya otomatis.

**Kode khusus platform**: buat fungsi menerima nama OS sebagai parameter (seperti `angoBinaryName(goos)`), lalu uji semua nilainya dari satu mesin.

## Cakupan yang sengaja tidak ada

- Tampilan window tidak diuji otomatis; periksa visual manual dengan `ango -window`.
- Tes berjalan di OS asal. CI menjalankan seluruh rangkaian di Linux dan Windows (`.github/workflows/ci.yml`).
