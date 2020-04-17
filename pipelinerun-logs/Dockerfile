FROM golang:1.14.2-alpine3.11
WORKDIR /go/src/pipelinerun-logs
COPY . .
RUN go build -o ./pipelinerun-logs ./cmd/http
CMD ["./pipelinerun-logs"]
