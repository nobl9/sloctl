FROM golang:1.21-alpine3.18 AS builder

ARG VERSION

WORKDIR /app

COPY ./go.mod ./go.sum ./
COPY ./cmd/sloctl ./cmd/sloctl
COPY ./internal ./internal

ARG LDFLAGS

RUN CGO_ENABLED=0 go build \
  -ldflags "${LDFLAGS}" \
  -o /artifacts/sloctl \
  "${PWD}/cmd/sloctl"

FROM scratch

COPY --from=builder /artifacts/sloctl /usr/bin/sloctl

ENTRYPOINT ["sloctl"]
