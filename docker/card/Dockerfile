# Stage 1: Build React SPA (parallel with Stage 2)
FROM node:22-slim AS frontend
WORKDIR /app
COPY admin-ui/package.json admin-ui/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY admin-ui/ .
RUN npm run build

# Stage 2: Build Go binary (parallel with Stage 1)
FROM golang:1.25.7 AS builder
WORKDIR /app

# Copy dependency files first
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" \
    go build -o app -ldflags="-X 'card/build.Date=$(date +"%a %d %b %Y")' -X 'card/build.Time=$(date +"%H:%M:%S %Z")'"

# Stage 3: Runtime (waits for both)
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates sqlite3 && rm -rf /var/lib/apt/lists/*
RUN mkdir /app
WORKDIR /app
COPY --from=builder /app/web-content/ /web-content/
COPY --from=frontend /app/dist/ /web-content/admin/spa/
COPY --from=builder /app/app .
ENTRYPOINT ["./app"]
