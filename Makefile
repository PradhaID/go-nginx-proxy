.PHONY: all build web test vet clean install run

BINARY := go-nginx-proxy
GO ?= go

all: build

# build the Svelte frontend into internal/web/dist (embedded)
web:
	cd web && npm install && npm run build
	touch internal/web/dist/.gitkeep

build: web
	$(GO) build -o bin/$(BINARY) ./cmd/server

# build without rebuilding the frontend
build-fast:
	$(GO) build -o bin/$(BINARY) ./cmd/server

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

run: build
	./bin/$(BINARY)

install:
	install -m 0755 bin/$(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -rf bin internal/web/dist/*
	touch internal/web/dist/.gitkeep
