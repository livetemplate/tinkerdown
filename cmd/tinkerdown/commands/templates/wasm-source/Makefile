# Makefile for building the WASM source module

# Output file name
OUTPUT = [[.ProjectName]].wasm

# Build the WASM module
.PHONY: build
build:
	tinygo build -o $(OUTPUT) -target wasi source.go

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(OUTPUT)

# Build and copy to an example project
.PHONY: install
install: build
	@echo "Built $(OUTPUT)"
	@echo "Copy to your tinkerdown project's sources/ directory"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build   - Build the WASM module"
	@echo "  clean   - Remove build artifacts"
	@echo "  install - Build and show installation instructions"
