BIN := sitecrate
VERSION := $(shell sed -n 's/^const version = "\(.*\)"/\1/p' cmd/sitecrate/main.go)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: all build install test test-go test-js test-browser check version dist clean

all: build

build:
	go build -o $(BIN) ./cmd/sitecrate

install:
	go install ./cmd/sitecrate

test: test-go test-js test-browser

test-go:
	go test ./...

test-js:
	node --test test/runtime.test.js

# BROWSERS=chromium make test-browser runs a single engine
test-browser: test/node_modules
	cd test && node --test browser.test.js

test/node_modules: test/package.json test/package-lock.json
	cd test && npm ci && npx playwright install
	touch $@

check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	go vet ./...

version:
	@echo $(VERSION)

# dist builds release archives for PLATFORMS, with checksums, into dist/
dist:
	rm -rf dist && mkdir dist
	for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; name=sitecrate-$(VERSION)-$$os-$$arch; ext=; \
		if [ $$os = windows ]; then ext=.exe; fi; \
		mkdir dist/$$name && \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags='-s -w' -o dist/$$name/sitecrate$$ext ./cmd/sitecrate && \
		cp README.md LICENSE dist/$$name/ && \
		if [ $$os = windows ]; then (cd dist && zip -qr $$name.zip $$name); else tar -C dist -czf dist/$$name.tar.gz $$name; fi && \
		rm -r dist/$$name || exit 1; \
	done
	cd dist && sha256sum * > SHA256SUMS

clean:
	rm -f $(BIN)
	rm -rf dist
	rm -rf test/node_modules
