# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

# ============================================================
# Stage 2: Minimal runtime with Caddy reverse proxy
# ============================================================
FROM caddy:2-alpine AS caddy

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw
COPY --from=caddy /usr/bin/caddy /usr/local/bin/caddy

COPY Caddyfile.trapiche /etc/caddy/Caddyfile
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN addgroup -g 1000 picoclaw && \
    adduser -D -u 1000 -G picoclaw picoclaw

USER picoclaw

# Run onboard to create initial config
RUN /usr/local/bin/picoclaw onboard

ENV PICOCLAW_GATEWAY_HOST=0.0.0.0
ENV PICOCLAW_GATEWAY_PORT=18790

EXPOSE 3000

ENTRYPOINT ["/entrypoint.sh"]
