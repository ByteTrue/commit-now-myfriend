.PHONY: build install clean

BINARY := cnm
OUTPUT := /tmp/$(BINARY)

build:
	go build -o $(OUTPUT) ./cmd/cnm/

install: build
	cp $(OUTPUT) /usr/local/bin/$(BINARY)
	chmod +x /usr/local/bin/$(BINARY)

clean:
	rm -f $(OUTPUT)
