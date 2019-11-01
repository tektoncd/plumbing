package main

import (
	"testing"

	"github.com/tektoncd/plumbing/pipelinerun-logs/pkg/config"
)

func TestValidateBuildID(t *testing.T) {
	for _, tc := range []struct {
		buildID   string
		shouldErr bool
	}{{
		buildID:   "",
		shouldErr: true,
	}, {
		buildID:   " ",
		shouldErr: true,
	}, {
		buildID:   "abcdef12-abc1-def2-abc3-def123456789",
		shouldErr: false,
	}, {
		buildID:   "123456",
		shouldErr: false,
	}} {
		conf := &config.Config{
			Hostname:  "",
			Port:      "",
			Project:   "",
			Cluster:   "",
			Namespace: "",
		}
		s := &Server{
			conf:        conf,
			client:      nil,
			adminClient: nil,
			entriesTmpl: nil,
		}
		err := s.validateBuildID(tc.buildID)
		if tc.shouldErr && err == nil {
			t.Errorf("expected error for build ID %q but received none", tc.buildID)
		} else if !tc.shouldErr && err != nil {
			t.Errorf("didnt expect error for build ID %q but received: %v", tc.buildID, err)
		}
	}
}
