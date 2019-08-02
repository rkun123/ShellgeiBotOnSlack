FROM golang:1.12-alpine

RUN apk add git gcc musl-dev

RUN mkdir -p /go/src/ShellgeiBotOnSlack

COPY . /go/src/ShellgeiBotOnSlack

ENV GOPATH /go

RUN go get -u github.com/golang/dep/cmd/dep

WORKDIR /go/src/ShellgeiBotOnSlack

RUN ../../bin/dep ensure

RUN go build -o main

CMD ./main slackKey.json botConfig.json
