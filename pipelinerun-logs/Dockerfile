FROM golang:1.13.1-alpine3.10
WORKDIR /go/src/pipelinerun-logs
COPY . .
RUN go build -o ./pipelinerun-logs ./cmd/http
CMD ["./pipelinerun-logs"]
