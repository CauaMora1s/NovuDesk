FROM golang:1.23-alpine

# Install system dependencies
RUN apk add --no-cache git curl openssl

# Install air for hot reload (pinned to last version supporting Go 1.23)
RUN go install github.com/air-verse/air@v1.61.0

# Install goose for migrations (pinned to last version supporting Go 1.23)
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.22.1

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b $(go env GOPATH)/bin v1.61.0

WORKDIR /app

# Copy go mod files first for layer caching
COPY apps/api/go.mod ./
RUN go mod download

EXPOSE 8080

# go mod tidy generates go.sum and fetches indirect deps into the volume-mounted source dir on first start
CMD ["sh", "-c", "go mod tidy && air -c .air.toml"]
