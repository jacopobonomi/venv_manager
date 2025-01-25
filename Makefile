.PHONY: build test install lint clean

build:
	go build -o bin/venv-manager cmd/venv-manager/main.go

install:
	./scripts/install.sh

lint:
	golangci-lint run

clean:
	rm -rf bin/