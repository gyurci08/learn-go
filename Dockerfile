# Stage 1: Build Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app-server ./main.go
  
# Stage 2: Create minimal runtime image
FROM alpine:latest

WORKDIR /app
  
# Make non-root user (best practice)
RUN adduser -D appuser

COPY --from=builder /app/app-server .
COPY ./internal ./internal

USER appuser
  
# Expose your service port (change if not 8080)
EXPOSE 8080

ENV PORT=8080

CMD ["./app-server"]
