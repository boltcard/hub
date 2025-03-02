FROM ubuntu:22.04

ARG PHOENIXD_VERSION=0.5.0

RUN apt-get update
RUN apt-get install -y wget
RUN apt-get install -y unzip
RUN apt-get install -y sqlite3
RUN apt-get install -y curl
RUN wget https://github.com/ACINQ/phoenixd/releases/download/v${PHOENIXD_VERSION}/phoenix-${PHOENIXD_VERSION}-linux-x64.zip
RUN unzip -j phoenix-${PHOENIXD_VERSION}-linux-x64.zip

EXPOSE 9740

ENTRYPOINT ["./phoenixd", "--http-bind-ip=0.0.0.0"]

