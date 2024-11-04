# Bolt Card Hub - Phoenix Edition

## features

- host bolt cards
- web admin
- low resource use

## technologies

- Phoenix Server
- SQLite database
- docker deployment

## install

- provision an m.1s VPS on [lunanode](https://www.lunanode.com/?r=9026) using the `Debian 12 64-bit` template  
  ($3.50 per month in June 2024, they accept bitcoin and are lighting enabled for payments)
- log in to the machine using SSH (Linux) or Putty (Windows)
- [install docker](https://docs.docker.com/engine/install/debian/)
- [enable managing docker as a non root user](https://docs.docker.com/engine/install/linux-postinstall/)
- initialise the system as below
- enter hub.yourdomain.com and set up an A record pointing to the VPS

```bash
docker volume create phoenix_data
docker volume create caddy_data
docker volume create caddy_config
docker volume create card_data
git clone https://github.com/boltcard/hub
cd hub
./docker_init.sh
```

- for a full local build from source and start with phoenix

```bash
docker compose --profile phoenix build
docker compose --profile phoenix up
```

- for development, it may be useful to start without phoenix for faster development iteration

```bash
docker compose down
docker compose build
docker compose up
```

- wait for a few minutes for the TLS certificate to be installed
- access the admin web interface at https://domain-name/admin/ to set a password and login

### to keep the service running

```bash
docker compose --profile phoenix up -d
docker compose logs -f
```

### to get the phoenix server seed words

```bash
sudo cat /var/lib/docker/volumes/hub_phoenix_data/_data/seed.dat ; echo
```

### to access the database

```bash
sudo apt install sqlite3
sudo sqlite3 /var/lib/docker/volumes/hub_card_data/_data/cards.db
```

### misc SQLite
```bash
sqlite> .tables
sqlite> .schema settings
sqlite> select * from settings;
sqlite> update settings set value='' where name='admin_password_hash';
sqlite> .quit
```

### to delete the database

```bash
sudo rm /var/lib/docker/volumes/hub_card_data/_data/cards.db
```

### after updating - check that docker disk use is reasonable

```bash
df -h
docker system df
docker system prune
```
