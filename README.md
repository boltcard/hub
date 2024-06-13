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

- provision an m1s VM on [lunanode (with affiliate link)](https://www.lunanode.com/?r=9026) using the `Debian 12 64-bit` template  
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
- the GroundControl URL could be `gc.boltcardwallet.com`

```bash
docker compose build
docker compose up
```

- monitor the logs
- access the web interface at <https://domain-name-from-init/>
- note that the first sats you send will appear as 'fee credit' rather than 'balance' until there is enough available to open a channel, as described [on the Phoenix Server website under Auto Liquidity](https://phoenix.acinq.co/server/auto-liquidity)

### to keep the service running

```bash
docker compose up -d
```

### to get the phoenix server seed words

```bash
sudo cat /var/lib/docker/volumes/hub_phoenix_data/_data/seed.dat
```

### to delete the database

```bash
sudo rm /var/lib/docker/volumes/hub_card_data/_data/cards.db
```
