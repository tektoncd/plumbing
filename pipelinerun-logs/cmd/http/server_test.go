package main

import (
	"net/url"
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
		conf := &config.Config{}
		s := &Server{
			conf: conf,
		}
		err := s.validateBuildID(tc.buildID)
		if tc.shouldErr && err == nil {
			t.Errorf("expected error for build ID %q but received none", tc.buildID)
		} else if !tc.shouldErr && err != nil {
			t.Errorf("didnt expect error for build ID %q but received: %v", tc.buildID, err)
		}
	}
}

func TestGetParams(t *testing.T) {
	for _, tc := range []struct {
		description         string
		supportedNamespaces string
		url                 string
		expectedParams      *logRequestParams
		expectError         bool
	}{{
		description:         "supports namespace query when set of supported namespaces is 1",
		supportedNamespaces: "foo",
		url:                 "https://foo.com?namespace=foo&buildid=12",
		expectedParams:      &logRequestParams{namespace: "foo", buildID: "12"},
		expectError:         false,
	}, {
		description:         "supports namespace query when set of supported namespaces is more than 1",
		supportedNamespaces: "bar , baz,  foo,quux",
		url:                 "https://foo.com?namespace=foo&buildid=12",
		expectedParams:      &logRequestParams{namespace: "foo", buildID: "12"},
		expectError:         false,
	}, {
		description:         "errors if requested namespace is not in supported set",
		supportedNamespaces: "foo,baz, bazinga, etc ",
		url:                 "https://foo.com?namespace=bar&buildid=12",
		expectedParams:      nil,
		expectError:         true,
	}} {
		t.Run(tc.description, func(t *testing.T) {
			conf := &config.Config{Namespace: tc.supportedNamespaces}
			s := &Server{conf: conf}
			s.buildNamespaceSet()
			url, err := url.Parse(tc.url)
			if err != nil {
				t.Errorf("invalid url: %v", err)
			}
			p, err := s.getParams(url)
			if err != nil && !tc.expectError {
				t.Errorf("did not expect error but received %v", err)
			} else if p == nil && tc.expectedParams != nil {
				t.Errorf("expected non-nil params but received nil for url %v", tc.url)
			} else if p != nil && p.namespace != tc.expectedParams.namespace {
				t.Errorf("expected namespace %q received %q", tc.expectedParams.namespace, p.namespace)
			} else if p != nil && p.buildID != tc.expectedParams.buildID {
				t.Errorf("expected build id %q received %q", tc.expectedParams.buildID, p.buildID)
			} else if err == nil && tc.expectError {
				t.Errorf("expected error but received nil for url %v", tc.url)
			}
		})
	}
}
