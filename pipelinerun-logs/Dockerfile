FROM golang:1.20.4-alpine3.16
WORKDIR /go/src/pipelinerun-logs
COPY . .
RUN go build -o ./pipelinerun-logs ./cmd/http
CMD ["./pipelinerun-logs"]
