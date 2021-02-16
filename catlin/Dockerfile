FROM golang:1.15-alpine3.12 as BUILD

COPY . /build
WORKDIR /build
RUN GOOS=linux GARCH=amd64 CGO_ENABLED=0 \
    go build -o catlin ./cmd/catlin

FROM alpine:3.12

RUN apk --no-cache add bash

WORKDIR /data

COPY --from=BUILD /build/catlin /usr/bin/catlin

CMD [ "catlin" ]
