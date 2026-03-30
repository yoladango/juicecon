# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o juicecon .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN adduser -D -u 1001 appuser

WORKDIR /app

COPY --from=builder /app/juicecon .

USER appuser

EXPOSE 8080

CMD ["./juicecon"]
