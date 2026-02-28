# Install Bolt Card Hub

## Prerequisites

- Ubuntu 24.04 LTS 64-bit VPS (2GB+ RAM recommended)
- DNS A record pointing your domain to the server's IP address
- Port 443 open for HTTPS

## Quick Install

```bash
HOST_DOMAIN=hub.yourdomain.com curl -fsSL https://raw.githubusercontent.com/boltcard/hub/main/install.sh | bash
```

Replace `hub.yourdomain.com` with your actual domain.

## What the Script Does

1. Removes snap-based Docker if present
2. Installs Docker from the official Docker apt repository (if not already installed)
3. Adds your user to the `docker` group
4. Clones this repository to `~/hub` (or pulls latest changes if it already exists)
5. Creates the `.env` file with your `HOST_DOMAIN`
6. Builds and starts all containers with `docker compose`

The script is idempotent — safe to run multiple times.

## Manual Install

```bash
# Install Docker
# https://docs.docker.com/engine/install/ubuntu/

# Clone the repo
git clone https://github.com/boltcard/hub.git ~/hub
cd ~/hub

# Configure
cp .env.example .env
# Edit .env and set HOST_DOMAIN=hub.yourdomain.com

# Build and run
docker compose build
docker compose up -d
```

## Post-Install

Visit `https://hub.yourdomain.com/admin/` to set your admin password and configure the hub.
