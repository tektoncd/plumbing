FROM golang:1.26.1-alpine
WORKDIR /go/src/pipelinerun-logs
COPY . .
RUN go build -o ./pipelinerun-logs ./cmd/http
CMD ["./pipelinerun-logs"]
