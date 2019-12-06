FROM golang:1.13.1-alpine3.10
WORKDIR /go/src/buildbot
COPY . .
RUN go build -o ./buildbot ./cmd/buildbot
CMD ["./buildbot"]
