# Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY internal/site/package*.json ./internal/site/
WORKDIR /app/internal/site
RUN npm install
COPY internal/site/ ./
RUN npm run build

# Build Backend
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache make git
WORKDIR /app
COPY . .
COPY --from=frontend-builder /app/internal/site/dist ./internal/site/dist
RUN go build -ldflags="-w -s" -o homemonit ./internal/cmd/hub

# Final Image
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/homemonit .
EXPOSE 8090
CMD ["./homemonit", "serve", "--http=0.0.0.0:8090"]
