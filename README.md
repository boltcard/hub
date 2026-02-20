# Bolt Card Hub - Phoenix Edition

A lightweight, self-hosted service for hosting NFC Bolt Cards on the Lightning Network, powered by phoenixd.

## Features

- host bolt cards
- web admin
- low resource use

## Technologies

- Phoenix Server
- SQLite database
- docker deployment

## Install

- provision an m.1s VPS on [lunanode](https://www.lunanode.com/?r=9026) using the `Debian 12 64-bit` template
  ($3.50 per month, LunaNode accept bitcoin and are lightning enabled for payments)
- log in to the machine using SSH (Linux) or Putty (Windows)
- [install docker](https://docs.docker.com/engine/install/debian/)
- [enable managing docker as a non root user](https://docs.docker.com/engine/install/linux-postinstall/)
- add swap space (the Go build needs more memory than the 1GB VPS provides)

```bash
sudo fallocate -l 1G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

- set up a DNS A record pointing `hub.yourdomain.com` to the VPS IP address

```bash
git clone https://github.com/boltcard/hub
cd hub
cp .env.example .env
# Edit .env to set HOST_DOMAIN=hub.yourdomain.com
```

- build and start the services

```bash
docker compose build
docker compose up -d
```

- wait for a few minutes for the TLS certificate to be installed
- access the admin web interface at https://hub.yourdomain.com/admin/ to set a password and login

## Operations

### View the Logs

```bash
docker compose logs
```

### Access the Database

```bash
docker compose exec card sqlite3 /card_data/cards.db
```

### Get a Shell on a Container

```bash
docker exec -it phoenix bash
./phoenix-cli
```

```bash
docker exec -it card bash
./app
```

### Update

```bash
git pull
docker compose down
docker compose build
docker compose up -d
docker system prune
```
