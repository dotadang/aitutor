.PHONY: build run install clean

VERSION ?= dev

build:
	go build -ldflags "-X main.Version=$(VERSION)" -o aitutor .

run:
	go run .

install:
	go install -ldflags "-X main.Version=$(VERSION)" .

clean:
	rm -f aitutor

vet:
	go vet ./...
