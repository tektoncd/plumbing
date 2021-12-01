FROM golang:1.17.6@sha256:0fa6504d3f1613f554c42131b8bf2dd1b2346fb69c2fc24a312e7cba6c87a71e
WORKDIR /go/src/pipelinerun-logs
COPY . .
RUN go build -o ./pipelinerun-logs ./cmd/http
CMD ["./pipelinerun-logs"]
