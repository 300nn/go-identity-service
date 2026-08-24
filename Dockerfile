# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26.7-bookworm AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" \
    -o /out/api ./cmd/api

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" \
    -o /out/migrate ./cmd/migrate

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" \
    -o /out/healthcheck ./cmd/healthcheck

FROM gcr.io/distroless/static-debian13:nonroot AS runtime

WORKDIR /app

COPY --chmod=755 --from=builder /out/api /app/api
COPY --chmod=755 --from=builder /out/migrate /app/migrate
COPY --chmod=755 --from=builder /out/healthcheck /app/healthcheck
COPY migrations /app/migrations

USER nonroot:nonroot

EXPOSE 8080 50051

STOPSIGNAL SIGTERM

ENTRYPOINT ["/app/api"]