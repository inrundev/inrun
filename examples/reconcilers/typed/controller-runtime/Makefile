# ── Typed Orkestra Operator — Hooks ───────────────────────────────────────────
BINARY_NAME ?= ork
DEV_OUTPUT_DIR  ?= $(HOME)/.orkestra/bin
PROD_OUTPUT_DIR ?= $(HOME)/.orkestra/bin/runtime
KATALOG     ?= katalog.yaml

IMAGE_REPO  ?= myorg/my-operator
IMAGE_TAG   ?= latest
IMAGE       ?= $(IMAGE_REPO):$(IMAGE_TAG)
BUILD_TAGS	?=

GOOS        ?= linux
GOARCH      ?= amd64
CGO_ENABLED  = 0

ORK_LDFLAGS := -X github.com/orkspace/orkestra/pkg/version.Version=$(GIT_VERSION) \
               -X github.com/orkspace/orkestra/pkg/version.Commit=$(GIT_COMMIT) \
               -X github.com/orkspace/orkestra/pkg/version.Date=$(GIT_DATE)

# ── registry ──────────────────────────────────────────────────────────────────
# Generates pkg/typeregistry/zz_generated_typeregistry.go.
# Re-run whenever you change apiTypes fields in katalog.yaml.
.PHONY: registry
registry:
	@[ -f go.mod.txt ] && mv go.mod.txt go.mod || true
	@[ -f go.sum.txt ] && mv go.sum.txt go.sum || true
	@find . -name "*.go" | xargs grep -l "^//go:build ignore$$" 2>/dev/null | while read f; do tail -n +3 "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; done || true
	ork generate registry --file $(KATALOG)

# ── build (development) ───────────────────────────────────────────────────────
# Builds the full CLI with all commands (validate, e2e, simulate, run, etc.)
.PHONY: build
build:
	@[ -f go.mod.txt ] && mv go.mod.txt go.mod || true
	@[ -f go.sum.txt ] && mv go.sum.txt go.sum || true
	@find . -name "*.go" | xargs grep -l "^//go:build ignore$$" 2>/dev/null | while read f; do tail -n +3 "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; done || true
	@mkdir -p $(DEV_OUTPUT_DIR)
	go mod tidy
	gofmt -w .
	go build \
		-ldflags "$(ORK_LDFLAGS)" \
		-o $(DEV_OUTPUT_DIR)/$(BINARY_NAME) ./cmd/orkestra
	@echo "✅ Development build: $(DEV_OUTPUT_DIR)/$(BINARY_NAME)"

# ── build-runtime (production) ────────────────────────────────────────────────
# Builds a production binary with ONLY the `run` command (no validate, e2e, etc.)
# This is what runs in your cluster — smaller attack surface, focused responsibility.
.PHONY: build-runtime
build-runtime:
	@[ -f go.mod.txt ] && mv go.mod.txt go.mod || true
	@[ -f go.sum.txt ] && mv go.sum.txt go.sum || true
	@find . -name "*.go" | xargs grep -l "^//go:build ignore$$" 2>/dev/null | while read f; do tail -n +3 "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; done || true
	@mkdir -p $(PROD_OUTPUT_DIR)
	go mod tidy
	gofmt -w .
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-tags "runtime" \
		-ldflags "$(ORK_LDFLAGS)" \
		-o $(PROD_OUTPUT_DIR)/$(BINARY_NAME) ./cmd/orkestra
	@echo "✅ Production runtime build: $(PROD_OUTPUT_DIR)/$(BINARY_NAME)"
	@echo "   This binary only supports 'ork run' — perfect for production."

# ── validate ──────────────────────────────────────────────────────────────────
.PHONY: validate
validate:
	$(DEV_OUTPUT_DIR)/$(BINARY_NAME) validate -f $(KATALOG)

# ── simulate ───────────────────────────────────────────────────────────────────────
.PHONY: simulate
simulate:
	$(DEV_OUTPUT_DIR)/$(BINARY_NAME) simulate

# ── e2e ───────────────────────────────────────────────────────────────────────
.PHONY: e2e
e2e:
	$(DEV_OUTPUT_DIR)/$(BINARY_NAME) e2e

# ── docker / push / release ───────────────────────────────────────────────────
# Copies the production runtime binary into the current directory for docker build,
# then removes it. The Docker image contains ONLY the production binary (runtime tag).
.PHONY: docker
docker: build-runtime
	@cp $(PROD_OUTPUT_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)
	docker build -t $(IMAGE) .
	@rm -f ./$(BINARY_NAME)
	@echo "✅ Docker image built: $(IMAGE)"

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: release
release: docker push

# ── clean ─────────────────────────────────────────────────────────────────────
.PHONY: clean
clean:
	@rm -f $(DEV_OUTPUT_DIR)/$(BINARY_NAME)
	@rm -rf $(PROD_OUTPUT_DIR)
	@echo "✅ Removed all local builds"

# ── help ──────────────────────────────────────────────────────────────────────
.PHONY: help
help:
	@echo "  registry       generate type registry from Katalog"
	@echo "  build          compile full development CLI (all commands)"
	@echo "  build-runtime  compile production binary (only 'ork run')"
	@echo "  validate       run katalog validation (using development binary)"
	@echo "  e2e            run end‑to‑end tests (using development binary)"
	@echo "  simulate       run simulation tests (using development binary)"
	@echo "  docker         build-runtime + copy into Docker image"
	@echo "  push           push Docker image to registry"
	@echo "  release        docker + push"
	@echo "  clean          remove all local builds"