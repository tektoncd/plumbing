package github

import (
	"encoding/json"

	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"google.golang.org/protobuf/encoding/protojson"
)

func Unmarshal(in map[string]interface{}) (*pb.Config, error) {
	raw, ok := in["config"]
	if !ok {
		return &pb.Config{}, nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	cfg := new(pb.Config)
	if err := protojson.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
