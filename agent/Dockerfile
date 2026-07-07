FROM golang:1.26-alpine AS builder
RUN apk add --no-cache make git
WORKDIR /app
COPY . .
RUN go build -ldflags="-w -s" -o homemonit-agent ./internal/cmd/agent

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/homemonit-agent .
CMD ["./homemonit-agent"]
