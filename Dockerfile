# Multi-Stage Dockerfile for Awd DriveRouter (Headless Server)

# Stage 1: Build Frontend (React / Vite)
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend (Go Binary)
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app

# Install build dependencies if needed
RUN apk add --no-cache git gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Copy compiled frontend dist into backend embed path
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Build headless server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o driverouter-server .

# Stage 3: Minimal Runtime Container
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata fuse

WORKDIR /app

# Copy binary from backend-builder
COPY --from=backend-builder /app/driverouter-server /app/driverouter-server

# Expose default HTTP Port
EXPOSE 8080

# Environment variables
ENV SERVER_MODE=true
ENV PORT=8080
ENV HOST=0.0.0.0

# Volume for data persistence
VOLUME ["/root/.config/driverouter"]

# Entrypoint
ENTRYPOINT ["/app/driverouter-server", "--server"]
