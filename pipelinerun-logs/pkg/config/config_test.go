package config

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		c             *Config
		expectedError string
	}{{
		c: &Config{
			Hostname:  "localhost",
			Port:      "9999",
			Project:   "FooProject",
			Cluster:   "FooCluster",
			Namespace: "",
		},
		expectedError: "namespace",
	}, {
		c: &Config{
			Hostname:  "localhost",
			Port:      "9999",
			Project:   "FooProject",
			Cluster:   "",
			Namespace: "FooNamespace",
		},
		expectedError: "cluster",
	}} {
		err := tc.c.Validate()
		if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
			t.Errorf("expected error container %v received %v", tc.expectedError, err)
		}
	}
}
