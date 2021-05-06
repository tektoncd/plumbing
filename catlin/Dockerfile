FROM docker.io/library/golang:1.16.3-alpine3.12 as build

COPY . /build
WORKDIR /build
RUN GOOS=linux GARCH=amd64 CGO_ENABLED=0 \
    go build -o catlin ./cmd/catlin

FROM docker.io/library/alpine:3.12

RUN apk --no-cache add bash shellcheck py3-pip

RUN pip3 install pylint

WORKDIR /data

COPY --from=build /build/catlin /usr/bin/catlin

CMD [ "catlin" ]
