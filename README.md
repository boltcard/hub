# Bolt Card Hub - Phoenix Edition

Alpha version - early code for further development

## features

- will host accounts for the Bolt Card Wallet app
- lightweight - to run on a low cost server (lunanode $3.50 per month)
- Phoenix Server - so a LND / CLN node is not needed - further reducing cost and complexity
- SQLite database
- docker deployment
- web based admin interface

## install

- provision Debian 12 64-bit on a [lunanode (with affiliate link)](https://www.lunanode.com/?r=9026) m1s VM ($3.50 per month in June 2024, they accept bitcoin and are lighting enabled for payments)
- log in to the machine using SSH (Linux) or Putty (Windows)
- [install docker](https://docs.docker.com/engine/install/debian/)
- [enable managing docker as a non root user](https://docs.docker.com/engine/install/linux-postinstall/)

```bash
$ docker volume create phoenix_data
$ docker volume create caddy_data
$ docker volume create caddy_config
$ docker volume create card_data
$ git clone https://github.com/boltcard/hub
$ cd hub
$ ./docker_init.sh
    # on lunanode the domain name could be your Hostname from the rDNS tab
    # the GroundControl URL could be "gc.boltcardwallet.com"
$ docker compose build
$ docker compose up
    #monitor the logs
```

- access the web interface at <https://domain-name-from-init/>
- the admin web page will auto update every few seconds
- use the QR code to connect a BoltCardWallet account
- note that the first sats you send will appear as 'fee credit' rather than 'balance' until there is enough to open a channel
- this behaviour is described [on the Phoenix Server website under Auto Liquidity](https://phoenix.acinq.co/server/auto-liquidity)

- to keep the service running

```bash
docker compose up -d
```

- to get the phoenix server seed words

```bash
$ sudo su -

# cat /var/lib/docker/volumes/hub_phoenix_data/_data/seed.dat

```

- to delete the database

```bash
$ sudo su -
# cd /var/lib/docker/volumes/hub_card_data/_data/
```

## TODO

- 2FA code for admin login
- show the phoenix server seed words for the admin
- rate limit API & website
- optionally take a fee when topping up funds on cards
- use an optional invite_secret when creating a card (in co-ordination with BoltCardWallet)
- gather a refund address when creating a card
