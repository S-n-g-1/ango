# Ango - Makefile
#
#   make help                      daftar target
#   make build                     bangun ango + ango-launcher ke bin/
#   make test                      jalankan semua tes (unit + test/)
#   make dist                      paket rilis untuk OS saat ini
#   make dist GOOS=windows         paket rilis Windows (.zip), dari OS apa pun
#   make dist GOOS=linux           paket rilis Linux (.tar.gz), dari OS apa pun
#   make dist-all                  paket Linux dan Windows sekaligus
#
# Build memakai CGO_ENABLED=0 (Ebitengine tidak butuh cgo di Linux/Windows),
# sehingga lintas-kompilasi tidak memerlukan compiler C.

GO        ?= go
VERSION   ?= 0.1.1
BIN_DIR   ?= bin
DIST_DIR  ?= dist
STORY     ?= examples/intro
PROJECTS  ?= examples

# Target platform. Timpa dengan: make build GOOS=windows GOARCH=amd64
GOOS      ?= $(shell $(GO) env GOOS)
GOARCH    ?= $(shell $(GO) env GOARCH)
export GOOS
export GOARCH
export CGO_ENABLED = 0

ifeq ($(GOOS),windows)
EXE         := .exe
ARCHIVE_EXT := zip
# Launcher tanpa jendela konsol di Windows.
LAUNCHER_LDFLAGS := -H=windowsgui
else
EXE         :=
ARCHIVE_EXT := tar.gz
LAUNCHER_LDFLAGS :=
endif

PKG_NAME  := ango-$(VERSION)-$(GOOS)-$(GOARCH)
PKG_DIR   := $(DIST_DIR)/$(PKG_NAME)
ARCHIVE   := $(DIST_DIR)/$(PKG_NAME).$(ARCHIVE_EXT)

BUILD_FLAGS    := -trimpath
LDFLAGS        := -s -w -X main.version=$(VERSION)
LAUNCHER_FLAGS := -s -w $(LAUNCHER_LDFLAGS)

.DEFAULT_GOAL := help
.PHONY: help all build ango launcher run play story-check launcher-run test test-unit test-integration cover vet fmt fmt-check check cross dist dist-all clean

help: ## Tampilkan daftar target
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  %-18s %s\n", $$1, $$2}'

all: build ## Sama dengan build

build: ango launcher ## Bangun ango dan ango-launcher ke bin/

ango: ## Bangun binary ango (engine + CLI)
	@mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ango$(EXE) ./cmd/ango

launcher: ## Bangun binary ango-launcher (launcher grafis)
	@mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -ldflags "$(LAUNCHER_FLAGS)" -o $(BIN_DIR)/ango-launcher$(EXE) ./cmd/ango-launcher

run: ango ## Jalankan STORY di window (default: examples/intro)
	$(BIN_DIR)/ango$(EXE) -window $(STORY)

play: run ## Alias dari run

story-check: ango ## Compile STORY saja dan laporkan error
	$(BIN_DIR)/ango$(EXE) -check $(STORY)

launcher-run: build ## Buka launcher dengan folder PROJECTS (default: examples)
	$(BIN_DIR)/ango-launcher$(EXE) -ango $(BIN_DIR)/ango$(EXE) $(PROJECTS)

test: ## Jalankan semua tes (unit + integrasi)
	GOOS= GOARCH= $(GO) test ./...

test-unit: ## Tes unit saja (tanpa folder test/)
	GOOS= GOARCH= $(GO) test $$($(GO) list ./... | grep -v '/test$$')

test-integration: ## Tes integrasi di test/
	GOOS= GOARCH= $(GO) test ./test/...

cover: ## Laporan cakupan tes (coverage.out)
	GOOS= GOARCH= $(GO) test -coverprofile=coverage.out ./...
	GOOS= GOARCH= $(GO) tool cover -func=coverage.out | tail -n 1

vet: ## go vet
	$(GO) vet ./...

fmt: ## Format semua kode Go
	$(GO) fmt ./...

fmt-check: ## Gagal bila ada file yang belum diformat
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "belum diformat:"; echo "$$out"; exit 1; fi

check: fmt-check vet test ## fmt-check + vet + test

cross: ## Pastikan kode ter-compile untuk Linux dan Windows
	GOOS=linux   GOARCH=amd64 $(GO) build ./...
	GOOS=windows GOARCH=amd64 $(GO) build ./...
	GOOS=linux   GOARCH=amd64 $(GO) vet ./...
	GOOS=windows GOARCH=amd64 $(GO) vet ./...

dist: build ## Paket rilis untuk GOOS/GOARCH saat ini
	rm -rf $(PKG_DIR)
	mkdir -p $(PKG_DIR)/projects
	cp $(BIN_DIR)/ango$(EXE) $(BIN_DIR)/ango-launcher$(EXE) $(PKG_DIR)/
ifeq ($(GOOS),windows)
	cp scripts/ango.bat scripts/play.bat $(PKG_DIR)/
else
	install -m 755 scripts/ango.sh scripts/play.sh $(PKG_DIR)/
endif
	cp -r $(STORY) $(PKG_DIR)/projects/$(notdir $(STORY))
	cp README.md LICENSE $(PKG_DIR)/
	cp -r docs $(PKG_DIR)/docs
ifeq ($(GOOS),windows)
	cd $(DIST_DIR) && rm -f $(PKG_NAME).zip && zip -qr $(PKG_NAME).zip $(PKG_NAME)
else
	tar -C $(DIST_DIR) -czf $(ARCHIVE) $(PKG_NAME)
endif
	ls -lh $(ARCHIVE)

dist-all: ## Paket Linux (amd64) dan Windows (amd64)
	$(MAKE) dist GOOS=linux GOARCH=amd64
	$(MAKE) dist GOOS=windows GOARCH=amd64

clean: ## Hapus bin/, dist/ dan coverage.out
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out
