.PHONY: help build run test test-race test-coverage fmt vet tidy clean

BINARY_NAME ?= treego
BIN_DIR ?= bin
CMD_DIR ?= ./cmd/treego

ifeq ($(OS),Windows_NT)
EXT ?= .exe
else
EXT ?=
endif

OUT ?= $(BIN_DIR)/$(BINARY_NAME)$(EXT)

help:
	@echo "Targets:"
	@echo "  build   Build $(OUT)"
	@echo "  run     Run: make run ARGS='.'"
	@echo "  test    Run tests"
	@echo "  fmt     Go fmt"
	@echo "  vet     Go vet"
	@echo "  tidy    Go mod tidy"
	@echo "  clean   Remove ./$(BIN_DIR)"

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(OUT) $(CMD_DIR)

run: build
	$(OUT) $(ARGS)

test:
	go test ./...

test-race:
	go test -v -race ./tests

test-coverage:
	go test -race -coverprofile=coverage.out ./tests

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)

