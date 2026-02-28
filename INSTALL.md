# Install Bolt Card Hub

## VPS

We recommend [LunaNode](https://www.lunanode.com/?r=9026) — the **m.1s** plan (1GB RAM) at **$3.50/month** is sufficient. LunaNode accept Bitcoin for payments.

Select **Ubuntu 24.04 LTS 64-bit** as the OS template.

## Prerequisites

- Ubuntu 24.04 LTS 64-bit VPS (1GB+ RAM is sufficient — no build step required)
- A domain pointing to the server — on LunaNode the rDNS hostname works out of the box, or set a DNS A record for your own domain
- Port 443 open for HTTPS (open by default on a new LunaNode VPS)

## Quick Install

```bash
export HOST_DOMAIN=hub.yourdomain.com && curl -fsSL https://raw.githubusercontent.com/boltcard/hub/main/install.sh | bash
```

Replace `hub.yourdomain.com` with your actual domain.

## What the Script Does

1. Removes snap-based Docker if present
2. Installs Docker from the official Docker apt repository (if not already installed)
3. Adds your user to the `docker` group
4. Creates `~/hub/` and downloads `docker-compose.yml` and `Caddyfile`
5. Creates the `.env` file with your `HOST_DOMAIN`
6. Pulls pre-built images from Docker Hub and starts all containers

No git clone or build step required — images are pulled from Docker Hub.

The script is idempotent — safe to run multiple times.

## Manual Install

```bash
# Install Docker
# https://docs.docker.com/engine/install/ubuntu/

# Create directory
mkdir -p ~/hub && cd ~/hub

# Download config files
curl -fsSL https://raw.githubusercontent.com/boltcard/hub/main/docker-compose.yml -o docker-compose.yml
curl -fsSL https://raw.githubusercontent.com/boltcard/hub/main/Caddyfile -o Caddyfile

# Configure
echo "HOST_DOMAIN=hub.yourdomain.com" > .env

# Pull and run
docker compose pull
docker compose up -d
```

## Post-Install

Visit `https://hub.yourdomain.com/admin/` to set your admin password and configure the hub.

Note: It may take a minute or two for the TLS certificate to be issued. If you see a "can't provide a secure connection" error, wait and try again.
