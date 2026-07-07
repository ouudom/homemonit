OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
SKIP_WEB ?= false
EXE_EXT := $(if $(filter windows,$(OS)),.exe,)

.PHONY: tidy build-agent build-hub build-hub-dev build clean test dev-server dev-agent dev-hub dev
.DEFAULT_GOAL := build

clean:
	go clean
	rm -rf ./build

test:
	go test -tags=testing ./...

tidy:
	go mod tidy

build-web-ui:
	@if command -v bun >/dev/null 2>&1; then \
		bun install --cwd ./internal/site && \
		bun run --cwd ./internal/site build; \
	else \
		npm install --prefix ./internal/site && \
		npm run --prefix ./internal/site build; \
	fi

build-agent: tidy
	GOOS=$(OS) GOARCH=$(ARCH) go build -o ./build/homemonit-agent_$(OS)_$(ARCH)$(EXE_EXT) -ldflags "-w -s" ./internal/cmd/agent

build-hub: tidy $(if $(filter false,$(SKIP_WEB)),build-web-ui)
	GOOS=$(OS) GOARCH=$(ARCH) go build -o ./build/homemonit_$(OS)_$(ARCH)$(EXE_EXT) -ldflags "-w -s" ./internal/cmd/hub

build-hub-dev: tidy
	mkdir -p ./internal/site/dist && touch ./internal/site/dist/index.html
	GOOS=$(OS) GOARCH=$(ARCH) go build -tags development -o ./build/homemonit-dev_$(OS)_$(ARCH)$(EXE_EXT) -ldflags "-w -s" ./internal/cmd/hub

build: build-agent build-hub

dev-server:
	cd ./internal/site
	@if command -v bun >/dev/null 2>&1; then \
		cd ./internal/site && bun run dev --host 0.0.0.0; \
	else \
		cd ./internal/site && npm run dev --host 0.0.0.0; \
	fi

dev-hub: export ENV=dev
dev-hub:
	mkdir -p ./internal/site/dist && touch ./internal/site/dist/index.html
	@if command -v entr >/dev/null 2>&1; then \
		find ./internal -type f -name '*.go' | entr -r -s "cd ./internal/cmd/hub && go run -tags development . serve --http 0.0.0.0:8090"; \
	else \
		cd ./internal/cmd/hub && go run -tags development . serve --http 0.0.0.0:8090; \
	fi

dev-agent:
	@if command -v entr >/dev/null 2>&1; then \
		find ./internal/cmd/agent/*.go ./agent/*.go | entr -r go run github.com/henrygd/beszel/internal/cmd/agent; \
	else \
		go run github.com/henrygd/beszel/internal/cmd/agent; \
	fi

dev: dev-server dev-hub dev-agent
