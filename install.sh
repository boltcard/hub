#!/usr/bin/env bash
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

# Bolt Card Hub installer
# Usage: export HOST_DOMAIN=hub.yourdomain.com && curl -fsSL https://raw.githubusercontent.com/boltcard/hub/main/install.sh | bash
# Log: /var/log/boltcardhub-install.log (when run via cloud-init startup script)

log() { echo "[$(date -Iseconds)] $*"; }

RAW_URL="https://raw.githubusercontent.com/boltcard/hub/main"
INSTALL_DIR="$HOME/hub"

# --- Check HOST_DOMAIN ---

if [ -z "${HOST_DOMAIN:-}" ]; then
    echo "Error: HOST_DOMAIN is not set."
    echo ""
    echo "Usage:"
    echo "  export HOST_DOMAIN=hub.yourdomain.com && curl -fsSL https://raw.githubusercontent.com/boltcard/hub/main/install.sh | bash"
    exit 1
fi

log "Installing Bolt Card Hub for domain: $HOST_DOMAIN"

# --- Check for root/sudo ---

if [ "$(id -u)" -eq 0 ]; then
    SUDO=""
else
    if ! command -v sudo &>/dev/null; then
        echo "Error: This script requires root privileges or sudo."
        exit 1
    fi
    SUDO="sudo"
fi

# --- Wait for apt lock (fresh VPS may be running unattended-upgrades) ---

log "Waiting for apt lock..."
while $SUDO fuser /var/lib/dpkg/lock-frontend &>/dev/null; do
    sleep 2
done

# --- Remove snap Docker if present ---

if snap list docker &>/dev/null 2>&1; then
    log "Removing snap Docker..."
    $SUDO snap remove docker
fi

# --- Install Docker if not present ---

if ! command -v docker &>/dev/null; then
    log "Installing Docker..."

    $SUDO apt-get update
    $SUDO apt-get install -y ca-certificates curl gnupg

    $SUDO install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    $SUDO chmod a+r /etc/apt/keyrings/docker.gpg

    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
        $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
        $SUDO tee /etc/apt/sources.list.d/docker.list > /dev/null

    $SUDO apt-get update
    $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    log "Docker installed"
else
    log "Docker already installed, skipping"
fi

# --- Add user to docker group if needed ---

if [ "$(id -u)" -ne 0 ] && ! groups | grep -qw docker; then
    log "Adding $USER to docker group..."
    $SUDO usermod -aG docker "$USER"
    echo "    You may need to log out and back in for group changes to take effect."
    echo "    Continuing with sudo for now..."
    DOCKER="$SUDO docker"
else
    DOCKER="docker"
fi

# --- Create install directory ---

mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

# --- Download config files ---

log "Downloading docker-compose.yml..."
curl -fsSL "$RAW_URL/docker-compose.yml" -o docker-compose.yml

log "Downloading Caddyfile..."
curl -fsSL "$RAW_URL/Caddyfile" -o Caddyfile

# --- Create .env ---

log "Writing .env file..."
echo "HOST_DOMAIN=$HOST_DOMAIN" > .env

# --- Pull and start ---

log "Pulling images..."
$DOCKER compose pull

log "Starting containers..."
$DOCKER compose up -d

log "Bolt Card Hub is running!"
echo "    Visit https://$HOST_DOMAIN/admin/ to set your admin password."
echo "    Note: It may take a minute or two for the TLS certificate to be issued."
