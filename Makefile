.PHONY: build run clean fmt test

BINARY=raportichka

build:
	go build -o $(BINARY) .

run:
	go run main.go

clean:
	rm -f $(BINARY)

fmt:
	go fmt ./...

test:
	go test -v ./...