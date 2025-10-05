package pkg

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"google.golang.org/grpc/codes"
)

func TestInterceptor_Process(t *testing.T) {
	prBody := `{"pr": "body-content"}`
	wantResponse := triggersv1.InterceptorResponse{
		Extensions: map[string]interface{}{
			"add_pr_body": map[string]interface{}{
				"pull_request_body": map[string]interface{}{
					"pr": "body-content",
				},
			},
		},
		Continue: true,
	}

	t.Run("without auth token", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte(prBody))
		}))
		i := Interceptor{}
		req := triggersv1.InterceptorRequest{
			Extensions: map[string]interface{}{
				"add_pr_body": map[string]interface{}{
					"pull_request_url": ts.URL,
				},
			},
		}
		got := i.Process(context.Background(), &req)
		if diff := cmp.Diff(&wantResponse, got); diff != "" {
			t.Fatalf("-want/+got: %s", diff)
		}
	})

	t.Run("with auth token", func(t *testing.T) {
		var gotToken string
		wantToken := "token abcde"
		ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			gotToken = request.Header.Get("Authorization")
			writer.Write([]byte(prBody))
		}))
		i := Interceptor{
			AuthToken: "abcde",
		}
		req := triggersv1.InterceptorRequest{
			Extensions: map[string]interface{}{
				"add_pr_body": map[string]interface{}{
					"pull_request_url": ts.URL,
				},
			},
		}
		got := i.Process(context.Background(), &req)
		if diff := cmp.Diff(wantToken, gotToken); diff != "" {
			t.Fatalf("Authorization header mismatch -want/+got: %s", diff)
		}
		if diff := cmp.Diff(&wantResponse, got); diff != "" {
			t.Fatalf("Resonse mismatch -want/+got: %s", diff)
		}
	})

}

func TestInterceptor_Process_Error(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  triggersv1.InterceptorRequest
		want triggersv1.InterceptorResponse
	}{{
		name: "empty extensions",
		req: triggersv1.InterceptorRequest{
			Extensions: map[string]interface{}{},
		},
		want: triggersv1.InterceptorResponse{
			Extensions: nil,
			Continue:   false,
			Status: triggersv1.Status{
				Code:    codes.FailedPrecondition,
				Message: "no 'add_pr_body' found in the extensions",
			},
		},
	}, {
		name: "no add_pr_body in extensions",
		req: triggersv1.InterceptorRequest{
			Extensions: map[string]interface{}{
				"foo": "bar",
			},
		},
		want: triggersv1.InterceptorResponse{
			Extensions: nil,
			Continue:   false,
			Status: triggersv1.Status{
				Code:    codes.FailedPrecondition,
				Message: "no 'add_pr_body' found in the extensions",
			},
		},
	}, {
		name: "no pull_request_url found",
		req: triggersv1.InterceptorRequest{
			Extensions: map[string]interface{}{
				"add_pr_body": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
		want: triggersv1.InterceptorResponse{
			Extensions: nil,
			Continue:   false,
			Status: triggersv1.Status{
				Code:    codes.FailedPrecondition,
				Message: "no 'pull_request_url' found",
			},
		},
	}, {
		name: "pull_request_url not a string",
		req: triggersv1.InterceptorRequest{
			Extensions: map[string]interface{}{
				"add_pr_body": map[string]interface{}{
					"pull_request_url": 4000,
				},
			},
		},
		want: triggersv1.InterceptorResponse{
			Extensions: nil,
			Continue:   false,
			Status: triggersv1.Status{
				Code:    codes.FailedPrecondition,
				Message: "'pull_request_url' found, but not a string",
			},
		},
	}, {
		name: "bad url",
		req: triggersv1.InterceptorRequest{
			Extensions: map[string]interface{}{
				"add_pr_body": map[string]interface{}{
					"pull_request_url": "bad_url",
				},
			},
		},
		want: triggersv1.InterceptorResponse{
			Extensions: nil,
			Continue:   false,
			Status: triggersv1.Status{
				Code:    codes.Internal, // TODO(dibyom): This should be a different error code
				Message: `Get "bad_url": unsupported protocol scheme ""`,
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			i := Interceptor{}
			got := i.Process(context.Background(), &tc.req)
			if diff := cmp.Diff(&tc.want, got); diff != "" {
				t.Fatalf("-want/+got: %s", diff)
			}
		})
	}
}

type requestOption func(*http.Request)

// creates a GitHub hook type request - no secret is provided in testing.
func createRequest(method, url, event, token string, body []byte, opts ...requestOption) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Github-Event", event)
	req.Header.Set("X-Github-Delivery", "testing-123")
	if token != "" {
		req.Header.Add("Authorization", "token "+token)
	}
	for _, o := range opts {
		o(req)
	}
	return req
}
