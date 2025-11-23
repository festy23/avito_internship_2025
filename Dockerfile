
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./

ENV GOTOOLCHAIN=auto

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /build/server .

COPY --from=builder /build/migrations ./migrations

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./server"]
