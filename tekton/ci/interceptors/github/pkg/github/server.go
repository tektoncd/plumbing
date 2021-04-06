package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v34/github"
	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
)

type Server struct {
	client *http.Client

	// Maps event types -> Interceptor handlers.
	router map[string]Interceptor

	// Webhook Secret
	webhookSecret []byte
}

func New(c *http.Client, webhookSecret []byte) *Server {
	return &Server{
		client:        c,
		webhookSecret: webhookSecret,
		router: map[string]Interceptor{
			"issue_comment": &IssueComment{},
			"push":          &Push{},
			"pull_request":  &PullRequest{},
		},
	}
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	resp, err := s.handle(r)
	if err != nil {
		// non-OK http should generally be reserved for network or other
		// system issues. For any handler errors, wrap in an
		// InterceptorResponse.
		if resp == nil {
			resp = &v1alpha1.InterceptorResponse{
				Continue: false,
			}
		}
		if serr := new(StatusError); errors.As(err, serr) {
			resp.Status = serr.Status
		} else {
			resp.Status = v1alpha1.Status{
				Code:    codes.Internal,
				Message: err.Error(),
			}
		}
	}

	if err := json.NewEncoder(rw).Encode(resp); err != nil {
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rw, "error writing response:", err)
		return
	}
}

func (s *Server) handle(r *http.Request) (*v1alpha1.InterceptorResponse, error) {
	in := new(v1alpha1.InterceptorRequest)

	if err := json.NewDecoder(r.Body).Decode(in); err != nil {
		return nil, Errorf(codes.InvalidArgument, "error parsing request: %v", err)
	}

	// Validate webhook signature.
	headers := http.Header(in.Header)
	if sig := headers.Get("X-Hub-Signature-256"); sig != "" {
		fmt.Println(sig)
		if err := github.ValidateSignature(sig, []byte(in.Body), s.webhookSecret); err != nil {
			return nil, Error(codes.InvalidArgument, "unknown signature")
		}
	}

	// Route request.
	eventType := in.Header["X-Github-Event"]
	var i Interceptor
	for _, e := range eventType {
		var ok bool
		i, ok = s.router[e]
		if ok {
			break
		}
	}
	if i == nil {
		return nil, Errorf(codes.Unimplemented, "unsupported event type: %s", eventType)
	}

	cfg, err := Unmarshal(in.InterceptorParams)
	if err != nil {
		return nil, Errorf(codes.InvalidArgument, "error reading config: %v", err)
	}

	// TODO: Add authenticated App/PAT client support.
	return i.Execute(r.Context(), github.NewClient(s.client), cfg, in)
}

type Interceptor interface {
	Execute(ctx context.Context, client *github.Client, cfg *pb.Config, req *v1alpha1.InterceptorRequest) (*v1alpha1.InterceptorResponse, error)
}
