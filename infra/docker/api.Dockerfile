# ─── Build stage ────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /bin/novudesk-api ./cmd/server

# ─── Final stage (distroless) ───────────────────────────────────
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /bin/novudesk-api /novudesk-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/novudesk-api"]
