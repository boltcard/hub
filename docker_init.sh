#!/bin/bash
echo Enter the domain name excluding the protocol
read domain_name

cp Caddyfile.template Caddyfile

echo writing the domain name as $domain_name ..
sed -i "s/domain_name/https:\/\/$domain_name/" Caddyfile

# create .env file with domain
echo "HOST_DOMAIN=$domain_name" > .env
echo "created .env and Caddyfile"
