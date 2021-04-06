//go:generate rm -rf config_go_proto
//go:generate mkdir config_go_proto
//go:generate protoc --go_out=config_go_proto --go-grpc_out=config_go_proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative -I$GOPATH/src/github.com/googleapis/googleapis -I. config.proto
package proto
