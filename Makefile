BINARY := triage
PREFIX ?= $(HOME)/.local/bin

.PHONY: build run test fmt vet clean install

build:
	go build -o $(BINARY) ./cmd/triage

run: build
	./$(BINARY)

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

# Installs to $(PREFIX) (default ~/.local/bin). Override with:
#   make install PREFIX=/usr/local/bin
install: build
	install -Dm755 $(BINARY) $(PREFIX)/$(BINARY)

clean:
	rm -f $(BINARY)
