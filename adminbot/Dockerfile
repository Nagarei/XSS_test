FROM golang:latest

RUN apt update  && apt upgrade -y

# install chrome
RUN wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
RUN apt install -y ./google-chrome-stable_current_amd64.deb

WORKDIR /adminbot

COPY adminbot/go.mod .
COPY adminbot/go.sum .
RUN go mod download

COPY adminbot/*.go .
RUN go build -o adminbot
CMD ["/adminbot/adminbot"] 
