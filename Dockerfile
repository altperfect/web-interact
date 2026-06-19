FROM node:24-alpine AS frontend

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend

RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/webhook-site ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -h /nonexistent -s /sbin/nologin -u 10001 -G app app
WORKDIR /app
COPY --from=backend --chown=10001:10001 /out/webhook-site /app/webhook-site
COPY --from=frontend --chown=10001:10001 /src/frontend/dist /app/static
RUN chmod 0555 /app/webhook-site && chmod -R a=rX /app/static

ENV ADDR=:8080 \
    STATIC_DIR=/app/static \
    COOKIE_SECURE=true \
    TRUST_PROXY=true \
    RETENTION_DAYS=30 \
    MAX_BODY_BYTES=2097152

USER 10001:10001
EXPOSE 8080
CMD ["/app/webhook-site"]
