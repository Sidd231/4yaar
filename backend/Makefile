.PHONY: all build run tidy migrate clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOMOD=$(GOCMD) mod
BINARY_NAME=bin/server
MAIN_PATH=./cmd/server/main.go

all: build

build:
	@echo "Building binary..."
	@$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

run:
	@echo "Starting 4Yaar server..."
	@$(GORUN) $(MAIN_PATH)

tidy:
	@echo "Tidying go modules..."
	@$(GOMOD) tidy

migrate:
	@echo "Running database migrations via Go auto-migration or SQL files..."
	@$(GORUN) $(MAIN_PATH) -migrate

clean:
	@echo "Cleaning artifacts..."
	@rm -rf bin/