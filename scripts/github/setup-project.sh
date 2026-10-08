#!/usr/bin/env bash
# Menyiapkan papan GitHub Projects bergaya Trello untuk Ango.
#
# Prasyarat:
#   - GitHub CLI: https://cli.github.com   (gh --version)
#   - jq
#   - login dengan izin project:   gh auth login   lalu   gh auth refresh -s project
#
# Pemakaian:
#   scripts/github/setup-project.sh OWNER/REPO [--owner OWNER] [--dry-run] [--no-issues]
#
#   OWNER/REPO   repositori tujuan, mis. sugi/ango
#   --owner      pemilik project (user atau organisasi); bawaan: pemilik repo
#   --dry-run    hanya cetak apa yang akan dilakukan
#   --no-issues  buat papan, label, dan milestone tanpa mengisi backlog
#
# Skrip aman dijalankan ulang: label dan milestone yang sudah ada dilewati.
# Isi backlog hanya dimasukkan sekali, jadi jangan jalankan ulang tanpa --no-issues.
set -euo pipefail

REPO="${1:-}"
[ -n "$REPO" ] || { sed -n '2,16p' "$0"; exit 2; }
shift
OWNER="${REPO%%/*}"
DRY=0
ISSUES=1
while [ $# -gt 0 ]; do
  case "$1" in
    --owner) OWNER="$2"; shift ;;
    --dry-run) DRY=1 ;;
    --no-issues) ISSUES=0 ;;
    *) echo "opsi tidak dikenal: $1" >&2; exit 2 ;;
  esac
  shift
done

HERE="$(cd "$(dirname "$0")" && pwd)"
BACKLOG="$HERE/backlog.tsv"
TITLE="Ango"

run() { if [ "$DRY" = 1 ]; then echo "[dry-run] $*"; else "$@"; fi; }
need() { command -v "$1" >/dev/null || { echo "perlu '$1' terpasang" >&2; exit 1; }; }
[ "$DRY" = 1 ] || { need gh; need jq; }

# ---------- 1. Label ----------
echo "== Label"
mk_label() { # nama warna deskripsi
  run gh label create "$1" --repo "$REPO" --color "$2" --description "$3" --force >/dev/null
}
mk_label "area:engine"   "1d76db" "VM, state, nilai"
mk_label "area:language" "5319e7" "Lexer, parser, compiler, sintaks .ango"
mk_label "area:backend"  "0e8a16" "Tampilan Ebitengine, aset, audio"
mk_label "area:gui"      "fbca04" "Tema, menu, dialog"
mk_label "area:launcher" "c5def5" "ango-launcher"
mk_label "area:docs"     "0075ca" "Dokumentasi"
mk_label "area:tooling"  "bfdadc" "Makefile, CI, CLI, rilis"
mk_label "area:platform" "d4c5f9" "Linux, Windows, Android, Web"
mk_label "type:bug"      "d73a4a" "Sesuatu tidak berfungsi"
mk_label "type:feature"  "a2eeef" "Fitur baru"
mk_label "type:chore"    "ededed" "Perawatan, refactor, dependensi"
mk_label "blocked"       "b60205" "Menunggu hal lain"

# ---------- 2. Milestone ----------
echo "== Milestone"
for m in v0.2 v0.3 v0.4 v0.5; do
  if [ "$DRY" = 1 ]; then echo "[dry-run] milestone $m"; continue; fi
  gh api "repos/$REPO/milestones" -f title="$m" >/dev/null 2>&1 || echo "  $m sudah ada"
done

# ---------- 3. Project ----------
echo "== Project"
if [ "$DRY" = 1 ]; then
  echo "[dry-run] gh project create --owner $OWNER --title $TITLE"
  NUM=0; PID=dry
else
  OUT="$(gh project create --owner "$OWNER" --title "$TITLE" --format json)"
  NUM="$(jq -r .number <<<"$OUT")"
  PID="$(jq -r .id <<<"$OUT")"
  gh project edit "$NUM" --owner "$OWNER" \
    --description "Papan kerja Ango (kanban bergaya Trello). Lihat docs/v0.1.1/project-management.md" >/dev/null
  gh project link "$NUM" --owner "$OWNER" --repo "$REPO" >/dev/null
  echo "  project #$NUM dibuat dan ditautkan ke $REPO"
fi

# Kolom: ganti opsi bawaan field Status (Todo / In Progress / Done).
echo "== Kolom (field Status)"
if [ "$DRY" = 0 ]; then
  STATUS_ID="$(gh project field-list "$NUM" --owner "$OWNER" --format json \
    | jq -r '.fields[] | select(.name=="Status") | .id')"
  gh api graphql -f query='
    mutation($field:ID!){
      updateProjectV2Field(input:{
        fieldId:$field,
        singleSelectOptions:[
          {name:"Backlog",     color:GRAY,   description:"Ide dan tugas belum terjadwal"},
          {name:"To Do",       color:BLUE,   description:"Dipilih untuk dikerjakan berikutnya"},
          {name:"In Progress", color:YELLOW, description:"Sedang dikerjakan (batas WIP: 3)"},
          {name:"Review",      color:ORANGE, description:"Menunggu tes, tinjauan, atau uji manual"},
          {name:"Done",        color:GREEN,  description:"Memenuhi Definition of Done"}
        ]
      }){ projectV2Field{ ... on ProjectV2SingleSelectField{ id } } }
    }' -f field="$STATUS_ID" >/dev/null
else
  echo "[dry-run] ubah opsi Status menjadi Backlog, To Do, In Progress, Review, Done"
fi

# Field tambahan.
echo "== Field Priority dan Area"
run gh project field-create "$NUM" --owner "$OWNER" --name "Priority" \
  --data-type SINGLE_SELECT --single-select-options "P0,P1,P2,P3" >/dev/null
run gh project field-create "$NUM" --owner "$OWNER" --name "Area" \
  --data-type SINGLE_SELECT \
  --single-select-options "engine,language,backend,gui,launcher,docs,tooling,platform" >/dev/null

# ---------- 4. Backlog ----------
if [ "$ISSUES" = 1 ]; then
  echo "== Backlog"
  if [ "$DRY" = 0 ]; then
    FIELDS="$(gh project field-list "$NUM" --owner "$OWNER" --format json)"
    fid() { jq -r --arg n "$1" '.fields[] | select(.name==$n) | .id' <<<"$FIELDS"; }
    oid() { jq -r --arg n "$1" --arg o "$2" '.fields[] | select(.name==$n) | .options[] | select(.name==$o) | .id' <<<"$FIELDS"; }
  fi
  while IFS='|' read -r title area prio mile stage body; do
    case "$title" in ''|\#*) continue ;; esac
    type="feature"
    case "$title" in Uji*|Verifikasi*|Ganti*) type="chore" ;; esac
    if [ "$DRY" = 1 ]; then
      echo "[dry-run] issue: $title [$area/$prio/$mile] -> $stage"
      continue
    fi
    URL="$(gh issue create --repo "$REPO" --title "$title" --body "$body" \
      --label "area:$area" --label "type:$type" --milestone "$mile")"
    ITEM="$(gh project item-add "$NUM" --owner "$OWNER" --url "$URL" --format json | jq -r .id)"
    set_opt() { gh project item-edit --id "$ITEM" --project-id "$PID" \
      --field-id "$(fid "$1")" --single-select-option-id "$(oid "$1" "$2")" >/dev/null; }
    set_opt Status "$stage"
    set_opt Priority "$prio"
    set_opt Area "$area"
    echo "  $URL -> $stage"
  done < "$BACKLOG"
fi

cat <<MSG

Selesai. Satu langkah manual di web (gh belum bisa mengatur tampilan):
  1. Buka project:  https://github.com/users/$OWNER/projects/${NUM}  (organisasi: /orgs/$OWNER/projects/${NUM})
  2. Tambah tampilan "+ New view" > Board, dikelompokkan menurut Status, lalu simpan.
  3. Settings > Workflows: aktifkan "Item added to project" (Status = Backlog),
     "Item closed" (Done), dan "Pull request merged" (Done).
MSG
