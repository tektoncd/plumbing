package main

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		q             Query
		expectedError string
	}{{
		q: Query{
			Project:   "",
			Cluster:   "FooCluster",
			Namespace: "FooNamespace",
			BuildID:   "123456",
		},
		expectedError: "project",
	}, {
		q: Query{
			Project:   "FooProject",
			Cluster:   "",
			Namespace: "FooNamespace",
			BuildID:   "123456",
		},
		expectedError: "cluster",
	}, {
		q: Query{
			Project:   "FooProject",
			Cluster:   "FooCluster",
			Namespace: "",
			BuildID:   "123456",
		},
		expectedError: "namespace",
	}, {
		q: Query{
			Project:   "FooProject",
			Cluster:   "FooCluster",
			Namespace: "FooNamespace",
			BuildID:   "",
		},
		expectedError: "build id",
	}} {
		err := tc.q.Validate()
		if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
			t.Errorf("expected error containing %v received %v", tc.expectedError, err)
		}
	}
}
