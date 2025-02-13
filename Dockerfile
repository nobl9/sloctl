FROM golang:1.24-alpine3.20 AS builder

WORKDIR /app

COPY ./go.mod ./go.sum ./
COPY ./cmd/sloctl ./cmd/sloctl
COPY ./internal ./internal

ARG LDFLAGS

RUN CGO_ENABLED=0 go build \
  -ldflags "${LDFLAGS}" \
  -o /artifacts/sloctl \
  "${PWD}/cmd/sloctl"

FROM gcr.io/distroless/static-debian12

COPY --from=builder /artifacts/sloctl /usr/bin/sloctl

ENTRYPOINT ["sloctl"]
