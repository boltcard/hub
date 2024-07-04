FROM ubuntu:22.04

RUN apt update
RUN apt install -y wget
RUN apt install -y unzip
RUN apt install -y sqlite3
RUN apt install -y curl
RUN wget https://github.com/ACINQ/phoenixd/releases/download/v0.2.0/phoenix-0.2.0-linux-x64.zip
RUN unzip -j phoenix-0.2.0-linux-x64.zip

EXPOSE 9740

ENTRYPOINT ["./phoenixd", "--http-bind-ip=0.0.0.0"]

