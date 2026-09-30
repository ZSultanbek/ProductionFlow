# syntax=docker/dockerfile:1.7

# --- build ------------------------------------------------------------------
# Pinned on BUILDPLATFORM and cross-compiled to TARGETARCH, so building an
# arm64 image on an amd64 CI machine (or the reverse) never runs an emulator.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

WORKDIR /src

# Module files on their own layer: dependencies are only re-downloaded when
# go.mod or go.sum actually change, not on every source edit.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILT_AT=unknown
ARG TARGETOS
ARG TARGETARCH

# The two cache mounts are what make a rebuild after a one-line change take
# seconds instead of a minute. They live in the builder, not the image.
#   CGO_ENABLED=0  static binary, so the runtime stage needs no libc
#   -trimpath      keeps absolute build paths out of the binary
#   -s -w          drops the symbol table and DWARF data
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT}" \
      -o /out/api ./cmd/api

# --- run --------------------------------------------------------------------
# distroless static: CA certificates, timezone data, /etc/passwd, and nothing
# else. No shell, no package manager, no busybox — a few MB, and nothing for a
# process to exec if it ever gets the chance.
#
# The trade-off is that `docker compose` cannot health-check this container:
# there is no curl, wget or sh inside it. The API's own /api/health endpoint is
# the check; call it from outside.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/api /api

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/api"]
