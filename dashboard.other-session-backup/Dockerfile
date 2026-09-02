FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/dashboard ./cmd/server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/dashboard /app/dashboard
COPY web /app/web

EXPOSE 8080
ENV PORT=8080
ENV WEB_DIR=/app/web

ENTRYPOINT ["/app/dashboard"]
