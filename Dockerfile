# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

ARG VERSION=0.1.0-dev
ARG COMMIT=unknown
ARG BUILDDATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILDDATE}" \
    -o /launcher ./cmd/launcher

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 1000 qolauncher \
    && mkdir -p /var/log/qolauncher /app \
    && chown -R qolauncher:qolauncher /var/log/qolauncher /app

COPY --from=builder /launcher /usr/local/bin/launcher

USER qolauncher

EXPOSE 8080 8081

ENTRYPOINT ["/usr/local/bin/launcher"]
