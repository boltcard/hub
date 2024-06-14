#!/bin/bash
echo Enter the domain name excluding the protocol
read domain_name

echo Enter the GroundControl URL excluding the protocol
read gc_url

cp docker/card/Dockerfile.template docker/card/Dockerfile

echo writing the domain name as $domain_name ..
sed -i "s/domain_name/https:\/\/$domain_name/" Caddyfile
sed -i "s/domain_name/$domain_name/g" docker/card/Dockerfile

echo writing the gc url as $gc_url ..
sed -i "s/gc_url/$gc_url/g" docker/card/Dockerfile
