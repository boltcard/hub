# install

- provision Debian 12 64-bit on a [lunanode (with affiliate link)](https://www.lunanode.com/?r=9026) m1s VM ($3.50 per month in June 2024, they accept bitcoin and are lighting enabled for payments)
- log in to the machine using SSH (Linux) or Putty (Windows)
- [install docker](https://docs.docker.com/engine/install/debian/)
- [enable managing docker as a non root user](https://docs.docker.com/engine/install/linux-postinstall/)

```
$ docker volume create phoenix_data
$ docker volume create caddy_data
$ docker volume create caddy_config
$ docker volume create card_data
$ git clone --branch v0.1 https://github.com/boltcard/hub
$ cd hub
$ ./docker_init.sh
$ docker compose build
$ docker compose up -d
```

# TODO before commit
- test documented install

# TODO
- 2FA code for admin login
- rate limit API & website
- QR code for adding funds, used for opening first channel
- optionally take a fee when topping up funds on cards
- use an optional invite_secret when creating a card (in co-ordination with BoltCardWallet)
- gather a refund address when creating a card
