# ----------- BUILD STAGE -----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for go mod if needed
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# ----------- FINAL STAGE -----------
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Copy server binary
COPY --from=builder /app/server /app/server

USER nonroot:nonroot

EXPOSE 8080

CMD ["/app/server"]
