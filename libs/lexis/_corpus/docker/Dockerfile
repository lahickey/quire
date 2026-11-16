# syntax=docker/dockerfile:1.9
# The ledger service: build with Go, run on distroless.

ARG GO_VERSION=1.27
ARG ALPINE_VERSION=3.21

# ---- dependencies: cached until go.mod changes ----
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS deps
WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata \
    && update-ca-certificates
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# ---- build ----
FROM deps AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOFLAGS="-trimpath -mod=readonly"
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/ledgerd ./cmd/ledgerd \
    && /out/ledgerd --version

# ---- test: `docker build --target test .` ----
FROM build AS test
RUN echo "$HOME" && go vet ./... && go test -count=1 -short ./...

# ---- assets ----
FROM node:22-slim AS assets
WORKDIR /web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm run build

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

LABEL org.opencontainers.image.title="ledgerd" \
      org.opencontainers.image.description="A ledger service" \
      org.opencontainers.image.source="https://example.com/acme/ledgerd" \
      org.opencontainers.image.licenses="MIT"

ENV TZ=UTC \
    LEDGER_ADDR=":8080" \
    LEDGER_DB="/data/ledger.db" \
    LEDGER_LOG=json

WORKDIR /app
COPY --from=deps /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build --chown=nonroot:nonroot /out/ledgerd /app/ledgerd
COPY --from=assets --chown=nonroot:nonroot /web/dist /app/static
COPY --chmod=0644 config/default.toml /etc/ledgerd/config.toml
ADD https://example.com/ca/internal.pem /etc/ssl/certs/internal.pem

VOLUME ["/data"]
EXPOSE 8080/tcp 9090/tcp
USER nonroot:nonroot
STOPSIGNAL SIGTERM

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/ledgerd", "healthcheck", "--addr", "http://127.0.0.1:8080/healthz"]

ENTRYPOINT ["/app/ledgerd"]
CMD ["serve", "--config", "/etc/ledgerd/config.toml"]

# ---- debug: a shell and the binary, never shipped ----
FROM alpine:${ALPINE_VERSION} AS debug
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]
RUN apk add --no-cache curl jq sqlite strace \
    && adduser -D -u 10001 ledger \
    && echo 'alias ll="ls -la"' >> /home/ledger/.profile
COPY --from=build /out/ledgerd /usr/local/bin/ledgerd
ONBUILD RUN echo "built from the debug image" >&2
USER ledger
CMD ["sh", "-c", "ledgerd --version && exec sh"]
