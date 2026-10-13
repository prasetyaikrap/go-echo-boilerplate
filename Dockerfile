# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Download dependencies first, so this layer stays cached until go.mod / go.sum change
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Static binary (no cgo) so it runs on a distroless image. -trimpath and -ldflags="-s -w" make it smaller and reproducible.
# TARGETOS / TARGETARCH are set by `docker buildx build --platform ...` for multi-arch images
ARG TARGETOS=linux
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/app .

# Writable directory for local storage (LOCAL_STORAGE_BASE_DIR default), and the prompts directory in case the
# checkout has none. Created here because distroless has no shell
RUN mkdir -p /out/storage /src/files/agents/prompts

# ---- Runtime stage ----
# distroless/static: no shell or package manager, includes CA certificates and timezone data.
# The :nonroot tag runs as UID 65532, which Kubernetes `runAsNonRoot: true` accepts
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/app /app/app
# LLM prompt templates read at runtime
COPY --from=build /src/files /app/files
COPY --from=build --chown=nonroot:nonroot /out/storage /app/storage

USER nonroot:nonroot

# Documentation only — the app listens on PORT, which must match
EXPOSE 3005

ENTRYPOINT ["/app/app"]
