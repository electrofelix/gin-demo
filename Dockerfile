# syntax = docker/dockerfile:1.0-experimental
FROM golangci/golangci-lint:v1.39 as golangci-lint
FROM golang:1.15-buster as build

COPY --from=golangci-lint /usr/bin/golangci-lint /usr/bin

WORKDIR /go/src
COPY . .

RUN --mount=type=cache,target=/go/pkg make all

FROM scratch

COPY --from=build /go/src/bin/gin-demo /

ENTRYPOINT ["/gin-demo"]
