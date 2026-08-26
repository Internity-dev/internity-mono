# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/api /out/worker ./
USER appuser
EXPOSE 8080
# The compose file picks the binary via `entrypoint:` (api vs worker) since
# both share this one image — no domain logic is duplicated between the two.
# No HEALTHCHECK here: worker has no HTTP listener and would report
# permanently unhealthy on a shared image-level check. Kept at the
# compose/orchestrator level, api service only.
ENTRYPOINT ["/app/api"]
