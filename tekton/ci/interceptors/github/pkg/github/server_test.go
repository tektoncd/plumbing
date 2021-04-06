package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestServer(t *testing.T) {
	webhookSecret := []byte("hunter2")
	srv := httptest.NewServer(New(http.DefaultClient, webhookSecret))
	defer srv.Close()

	f, err := ioutil.ReadFile("testdata/push.json")
	if err != nil {
		log.Fatal(err)
	}
	req := v1alpha1.InterceptorRequest{
		Body: string(f),
		Header: map[string][]string{
			"X-Github-Event":      {"push"},
			"X-Hub-Signature-256": {signature(f, webhookSecret)},
		},
		InterceptorParams: map[string]interface{}{
			"config": &pb.Config{
				Push: &pb.PushConfig{},
			},
		},
	}
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(req); err != nil {
		log.Fatal(err)
	}

	resp, err := srv.Client().Post(srv.URL, "application-json", b)
	out, _ := httputil.DumpResponse(resp, true)
	t.Log(out)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Fatal(resp, err)
	}

}

func signature(b, webhookSecret []byte) string {
	mac := hmac.New(sha256.New, webhookSecret)
	mac.Write(b)
	return fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
}
