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
	@echo "  test-race  Run tests with -race (CGO; Windows needs MinGW toolchain, see CONTRIBUTING.md)"
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

# -race requires CGO on Windows. cgo expects a MinGW-style toolchain; MSVC-target
# Clang (LLVM installer) is rejected here because it fails at link time (see CONTRIBUTING.md).
ifeq ($(OS),Windows_NT)
ifneq ($(strip $(WINDOWS_CGO_CC)),)
ifneq ($(strip $(WINDOWS_CGO_CXX)),)
WINDOWS_RACE_CC := $(WINDOWS_CGO_CC)
WINDOWS_RACE_CXX := $(WINDOWS_CGO_CXX)
else
$(error WINDOWS_CGO_CXX is required when WINDOWS_CGO_CC is set (for example clang++ or g++). See CONTRIBUTING.md)
endif
else
# Avoid "where" probes (they can misbehave under some Make/shell setups); these fail fast if missing.
HAS_GCC := $(shell gcc -dumpversion 2>nul)
ifneq ($(strip $(HAS_GCC)),)
WINDOWS_RACE_CC := gcc
WINDOWS_RACE_CXX := g++
else
HAS_CLANG_TRIPLE := $(shell clang -print-target-triple 2>nul)
ifeq ($(strip $(HAS_CLANG_TRIPLE)),)
$(error Windows go test -race needs CGO. Install MinGW-w64 (put gcc/g++ on PATH), llvm-mingw, or use MSYS2 MINGW64. See CONTRIBUTING.md)
else
ifneq (,$(findstring msvc,$(HAS_CLANG_TRIPLE)))
$(error Go -race+cgo on Windows does not work with MSVC-target Clang (triple: $(HAS_CLANG_TRIPLE)). Install MinGW-w64 gcc, llvm-mingw, or run from MSYS2 MINGW64 where Clang targets MinGW. See CONTRIBUTING.md)
endif
WINDOWS_RACE_CC := clang
WINDOWS_RACE_CXX := clang++
endif
endif
endif
test-race: export CC := $(WINDOWS_RACE_CC)
test-race: export CXX := $(WINDOWS_RACE_CXX)
test-coverage: export CC := $(WINDOWS_RACE_CC)
test-coverage: export CXX := $(WINDOWS_RACE_CXX)
endif
test-race: export CGO_ENABLED := 1
test-race:
	go test -v -race ./tests

test-coverage: export CGO_ENABLED := 1
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

