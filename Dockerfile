
# Stage 1: Build Vue frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY Portifolio_front/energy-controller/package*.json ./

# Install dependencies
RUN npm install

# Copy frontend source
COPY Portifolio_front/energy-controller/ ./

# Build frontend for production
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app/backend

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY Portifolio_back/go.mod Portifolio_back/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY Portifolio_back/ ./

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Stage 3: Final runtime image
FROM alpine:latest

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Go binary from backend builder
COPY --from=backend-builder /app/backend/main .

# Copy the built frontend from frontend builder
COPY --from=frontend-builder /app/frontend/dist ./static

# Expose port (Azure App Service will use PORT environment variable)
EXPOSE 8080

# Run the application
CMD ["./main"]
