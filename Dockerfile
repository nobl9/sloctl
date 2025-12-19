FROM golang:1.25-alpine3.23 AS builder

WORKDIR /app

COPY ./go.mod ./go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY ./cmd/sloctl ./cmd/sloctl
COPY ./internal ./internal

ARG LDFLAGS

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build \
    -ldflags "${LDFLAGS}" \
    -o /artifacts/sloctl \
    "${PWD}/cmd/sloctl"

FROM gcr.io/distroless/static-debian12

COPY --from=builder /artifacts/sloctl /usr/bin/sloctl

ENTRYPOINT ["sloctl"]
