# Manajemen proyek (papan kanban)

> Bagian dari [dokumentasi Ango](README.md).

Pekerjaan Ango dikelola dengan **GitHub Projects** memakai teknik kanban ala Trello: satu papan, kartu bergerak dari kiri ke kanan, dan jumlah pekerjaan yang berjalan dibatasi.

## Membuat papan

Sekali saja, dengan [GitHub CLI](https://cli.github.com) dan `jq`:

```bash
gh auth login
gh auth refresh -s project            # izin untuk Projects
scripts/github/setup-project.sh OWNER/REPO --dry-run   # lihat dulu apa yang akan dibuat
scripts/github/setup-project.sh OWNER/REPO
```

Skrip membuat label, milestone `v0.2` sampai `v0.5`, project "Ango" yang ditautkan ke repositori, kolom, field, dan mengisi backlog awal dari [`scripts/github/backlog.tsv`](../../scripts/github/backlog.tsv) sebagai issue. Opsi: `--owner` (bila project milik organisasi atau akun berbeda) dan `--no-issues` (papan saja, tanpa backlog).

Dua langkah tetap manual karena `gh` belum bisa mengaturnya:

1. Di project, **New view > Board**, dikelompokkan menurut **Status**.
2. **Settings > Workflows**: aktifkan *Item added to project* (Status = Backlog), *Item closed* dan *Pull request merged* (Status = Done).

## Kolom

| Kolom | Arti | Aturan |
| --- | --- | --- |
| **Backlog** | Ide dan tugas belum terjadwal | Boleh kasar; urutkan dari atas = paling penting |
| **To Do** | Dipilih untuk dikerjakan berikutnya | Maksimal sekitar 5 kartu, sudah jelas kriteria selesainya |
| **In Progress** | Sedang dikerjakan | **Batas WIP: 3.** Selesaikan sebelum mengambil kartu baru |
| **Review** | Menunggu tes, tinjauan, atau uji manual | Termasuk uji window di Windows dan Linux |
| **Done** | Memenuhi Definition of Done | Ditutup otomatis saat issue ditutup atau PR digabung |

## Field dan label

| Field / label | Nilai |
| --- | --- |
| Priority | **P0** penghalang rilis, **P1** tinggi, **P2** normal, **P3** suatu hari nanti |
| Area | engine, language, backend, gui, launcher, docs, tooling, platform |
| Label `type:` | `bug`, `feature`, `chore` |
| Label `blocked` | Menunggu hal lain; tulis penyebabnya di komentar |
| Milestone | `v0.2`, `v0.3`, `v0.4`, `v0.5` |

## Definition of Done

Kartu pindah ke Done bila: `make check` dan `make cross` lolos, tes ditambahkan, dokumentasi dan `CHANGELOG.md` diperbarui, dan perubahan tidak merusak salah satu dari Linux atau Windows. Daftar ini juga ada di template PR.

## Kebiasaan kerja

- **Tarik, jangan dorong.** Ambil dari To Do paling atas; jangan mulai kartu baru saat In Progress penuh.
- **Kartu kecil.** Satu kartu sebaiknya selesai dalam beberapa hari; pecah bila lebih besar.
- **Tinjau papan mingguan** (15 menit): pindahkan yang basi ke Backlog, pilih isi To Do, cek yang `blocked`.
- **Rilis** mengikuti milestone: saat semua kartu milestone Done, perbarui `CHANGELOG.md`, beri tag, dan jalankan `make dist-all`.
- **Satu issue, satu cabang, satu PR** dengan `Menutup #nomor` di deskripsi.

## Menambah backlog

Isi cepat lewat tab Issues (template Bug atau Fitur), atau tambahkan baris ke `backlog.tsv` dan jalankan skrip dengan papan baru. Format baris: `judul|area|prioritas|milestone|kolom|deskripsi`.
