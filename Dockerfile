# Multi-stage build for minimal production image
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/hotel-server ./cmd/server/main.go

# Minimal runtime image using distroless / alpine non-root
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /home/appuser

COPY --from=builder /app/bin/hotel-server /usr/local/bin/hotel-server

USER appuser

EXPOSE 8080

CMD ["hotel-server"]
