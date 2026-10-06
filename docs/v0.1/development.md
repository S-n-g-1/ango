
# Jalankan di terminal
go run ./cmd/ango examples/basic/demo.ango

# Jalankan otomatis untuk testing
go run ./cmd/ango -auto examples/basic/demo.ango

# Validasi script tanpa menjalankan game
go run ./cmd/ango -check examples/basic/demo.ango

# Jalankan window Ebitengine
go run ./cmd/ango -window examples/basic/demo.ango


##  nanti

ango check game.ango
ango run game.ango
ango build game.ango
ango play game.ango