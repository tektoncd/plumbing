FROM docker.io/library/golang:1.17.6@sha256:0fa6504d3f1613f554c42131b8bf2dd1b2346fb69c2fc24a312e7cba6c87a71e as build

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
