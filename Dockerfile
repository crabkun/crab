# syntax=docker/dockerfile:1.7

# Go 1.26.6 is the current stable release (https://go.dev/dl/, 2026-08-18).
# Alpine 3.24 is the current Alpine stable series used by the official image.
FROM golang:1.26.6-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=2.0.0
ARG COMMIT=unknown
ARG DATE=unknown

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w \
        -X github.com/crabkun/crab/internal/version.Version=${VERSION} \
        -X github.com/crabkun/crab/internal/version.Commit=${COMMIT} \
        -X github.com/crabkun/crab/internal/version.Date=${DATE}" \
    -o /out/crab ./cmd/crab

# Distroless static (Debian 13) is the current minimal image for static Go binaries.
# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /out/crab /crab

USER nonroot:nonroot
EXPOSE 51324
ENTRYPOINT ["/crab"]
CMD ["--config", "/etc/crab/config.json"]
