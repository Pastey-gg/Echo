# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
ARG COMMIT=""
ARG COMMIT_TIME=""
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go generate ./...
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    MODULE="$(go list -m)" && CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X ${MODULE}/state.commitHash=${COMMIT} -X ${MODULE}/state.commitTime=${COMMIT_TIME}" \
    -o echo .

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/echo .
COPY --from=builder /build/VERSION .
COPY --from=builder /build/docs ./docs

CMD ["./echo"]
