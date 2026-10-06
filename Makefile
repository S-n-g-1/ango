# Ango - Makefile
#   make            tampilkan bantuan
#   make build      bangun ango + ango-launcher ke bin/
#   make dist       paket siap bagi (tar.gz) di dist/

GO        ?= go
VERSION   ?= 0.1.0
BIN_DIR   ?= bin
DIST_DIR  ?= dist
STORY     ?= examples/intro
PROJECTS  ?= examples

GOOS      := $(shell $(GO) env GOOS)
GOARCH    := $(shell $(GO) env GOARCH)
PKG_NAME  := ango-$(VERSION)-$(GOOS)-$(GOARCH)
PKG_DIR   := $(DIST_DIR)/$(PKG_NAME)
BUILD_FLAGS := -trimpath -ldflags "-s -w"

.DEFAULT_GOAL := help
.PHONY: help all build ango launcher run play story-check launcher-run test vet fmt check dist clean

help: ## Tampilkan daftar target
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  %-14s %s\n", $$1, $$2}'

all: build ## Sama dengan build

build: ango launcher ## Bangun ango dan ango-launcher ke $(BIN_DIR)/

ango: ## Bangun binary ango (engine + CLI)
	@mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/ango ./cmd/ango

launcher: ## Bangun binary ango-launcher (launcher grafis)
	@mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/ango-launcher ./cmd/ango-launcher

run: ango ## Jalankan STORY di window (default: examples/intro)
	$(BIN_DIR)/ango -window $(STORY)

play: run ## Alias dari run

story-check: ango ## Compile STORY saja dan laporkan error
	$(BIN_DIR)/ango -check $(STORY)

launcher-run: build ## Buka launcher dengan folder PROJECTS (default: examples)
	$(BIN_DIR)/ango-launcher -ango $(BIN_DIR)/ango $(PROJECTS)

test: ## Jalankan semua tes
	$(GO) test ./...

vet: ## go vet
	$(GO) vet ./...

fmt: ## Format semua kode Go
	$(GO) fmt ./...

check: vet test ## vet + test

dist: build ## Paket tar.gz: ango, ango-launcher, ango.sh, projects/, docs
	rm -rf $(PKG_DIR)
	mkdir -p $(PKG_DIR)/projects
	cp $(BIN_DIR)/ango $(BIN_DIR)/ango-launcher $(PKG_DIR)/
	install -m 755 scripts/ango.sh $(PKG_DIR)/ango.sh
	cp -r $(STORY) $(PKG_DIR)/projects/$(notdir $(STORY))
	cp README.md $(PKG_DIR)/
	if [ -d docs ]; then cp -r docs $(PKG_DIR)/docs; fi
	if [ -f LICENSE ]; then cp LICENSE $(PKG_DIR)/; fi
	tar -C $(DIST_DIR) -czf $(DIST_DIR)/$(PKG_NAME).tar.gz $(PKG_NAME)
	ls -lh $(DIST_DIR)/$(PKG_NAME).tar.gz

clean: ## Hapus bin/ dan dist/
	rm -rf $(BIN_DIR) $(DIST_DIR)