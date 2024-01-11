FROM bats/bats:v1.10.0

WORKDIR /sloctl

COPY ./go.mod ./go.sum ./
COPY ./cmd/sloctl ./cmd/sloctl
COPY ./internal ./internal

RUN apk --no-cache --update add \
  gettext go jq git python3 py3-pip
RUN pip install yq

COPY ./test ./test
