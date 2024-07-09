# Bolt Card Hub - Phoenix Edition

Alpha version - early code for further development

## features

- host accounts for the Bolt Card Wallet app
- lightweight
- Phoenix Server
- SQLite database
- docker deployment
- web admin

## install

- provision an m1s VM on [lunanode](https://www.lunanode.com/?r=9026) using the `Debian 12 64-bit` template  
  ($3.50 per month in June 2024, they accept bitcoin and are lighting enabled for payments)
- log in to the machine using SSH (Linux) or Putty (Windows)
- [install docker](https://docs.docker.com/engine/install/debian/)
- [enable managing docker as a non root user](https://docs.docker.com/engine/install/linux-postinstall/)

```bash
docker volume create phoenix_data
docker volume create caddy_data
docker volume create caddy_config
docker volume create card_data
git clone https://github.com/boltcard/hub
cd hub
./docker_init.sh
```

- the domain name could be `your Hostname from the lunanode VM rDNS tab`
- the domain name could also be a custom domain or subdomain where you have set an A record to your VPS external IP
- the GroundControl URL could be `gc.boltcardwallet.com`

```bash
docker compose build
docker compose up
```

- set an admin password and access the admin web interface at https://domain-name/admin/

### to keep the service running

```bash
docker compose up -d
```

### to get the phoenix server seed words

```bash
sudo cat /var/lib/docker/volumes/hub_phoenix_data/_data/seed.dat ; echo
```

### to delete the database

```bash
sudo rm /var/lib/docker/volumes/hub_card_data/_data/cards.db
```
