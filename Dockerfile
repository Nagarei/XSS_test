FROM golang:latest

WORKDIR /development
COPY air.toml .

#install mariadb-client
RUN apt-get update \
    && apt-get install -y default-mysql-client

WORKDIR /webapp

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz
RUN go install github.com/cosmtrek/air@latest

RUN ln -sf /usr/share/zoneinfo/Asia/Tokyo /etc/localtime

COPY webapp/go.mod .
COPY webapp/go.sum .

RUN go mod download
