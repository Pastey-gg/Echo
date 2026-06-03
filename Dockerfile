FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG COMMIT=""
ARG COMMIT_TIME=""
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/EvieePy/Echo/state.commitHash=${COMMIT} -X github.com/EvieePy/Echo/state.commitTime=${COMMIT_TIME}" \
    -o echo .

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/echo .
COPY --from=builder /build/VERSION .

CMD ["./echo"]
