# ============================================================
# Stage 1: Build frontend assets
# ============================================================
FROM node:22-alpine AS frontend

RUN npm install -g pnpm

WORKDIR /src/web/frontend
COPY web/frontend/package.json web/frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/frontend/ ./
RUN pnpm build:backend

# ============================================================
# Stage 2: Build Go binaries (picoclaw + launcher)
# ============================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy built frontend into the embed directory
COPY --from=frontend /src/web/backend/dist ./web/backend/dist

# Build the main picoclaw binary
RUN make build

# Build the launcher (web console) binary
RUN CGO_ENABLED=0 go build -v -tags stdjson -o build/picoclaw-launcher ./web/backend

# ============================================================
# Stage 3: Minimal runtime
# ============================================================
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

# Copy both binaries to the same directory so the launcher finds picoclaw
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw
COPY --from=builder /src/build/picoclaw-launcher /usr/local/bin/picoclaw-launcher

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
