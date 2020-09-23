FROM golang:1.14 as BUILD

COPY . /build
WORKDIR /build
RUN GOOS=linux GARCH=amd64 CGO_ENABLED=0 \
    go build -o catlin ./cmd/catlin

FROM scratch

WORKDIR /app
COPY --from=BUILD /build/catlin /app/catlin

ENTRYPOINT ["/app/catlin"]

