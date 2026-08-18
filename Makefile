MODULE  := github.com/crabkun/crab
VERSION ?= 2.0.0
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: all build test tidy docker up down demo dist

all: test build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/crab ./cmd/crab

test:
	go test ./...

tidy:
	go mod tidy

docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-t crab:$(VERSION) .

up:
	docker compose up -d --build

down:
	docker compose --profile demo down

demo:
	docker compose --profile demo up -d --build

dist:
	VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) bash scripts/dist.sh
