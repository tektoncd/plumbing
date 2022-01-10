package testutil

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"

	"github.com/google/go-github/v41/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
)

const (
	// DefaultTEPReadmeContent is used in tests that expect a consistent README.md
	DefaultTEPReadmeContent = `there are three teps in here
on later lines
|[TEP-1234](1234-something-or-other.md) | Some TEP Title | proposed | 2021-12-20 |
|[TEP-5678](5678-second-one.md) | Another TEP Title | proposed | 2021-12-20 |
|[TEP-4321](4321-third-one.md) | Yet Another TEP Title | implementing | 2021-12-20 |
tada, three valid TEPs
`
)

var (
	// ReadmeURL is used for configuring handlers expecting to get the contents of the TEP index README.
	ReadmeURL = fmt.Sprintf("/repos/%s/%s/contents/%s/%s", ghclient.TEPsOwner, ghclient.TEPsRepo,
		ghclient.TEPsDirectory, ghclient.TEPsReadmeFile)

	defaultRunParams = map[string]string{
		tep.ActionParamName:      "opened",
		tep.PRNumberParamName:    "1",
		tep.PRTitleParamName:     "Some PR",
		tep.PRBodyParamName:      "A PR body, without any TEPs in it",
		tep.PackageParamName:     "tektoncd/pipeline",
		tep.PRIsMergedParamName:  "false",
		tep.GitRevisionParamName: "someSha",
	}
)

// DefaultREADMEHandlerFunc returns a function that serves the default readme content
func DefaultREADMEHandlerFunc() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, GHContentJSON(DefaultTEPReadmeContent))
	}
}

// NoCommentsOnPRHandlerFunc returns a function that handles returning no comments for a PR, or posting a new comment
func NoCommentsOnPRHandlerFunc(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			_, _ = fmt.Fprint(w, `[]`)
			return
		case "POST":
			_, _ = fmt.Fprint(w, `{"id":1}`)
			return
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}
}

// ConstructRunParams generates a slice of Params from a map of overrides of default parameters and a second map of additional parameters
func ConstructRunParams(overrides map[string]string, additionalParams map[string]string) []v1beta1.Param {
	var params []v1beta1.Param

	for key, defaultValue := range defaultRunParams {
		if overrideValue, ok := overrides[key]; ok {
			if overrideValue != "" {
				params = append(params, v1beta1.Param{
					Name:  key,
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: overrideValue},
				})
			}
		} else {
			params = append(params, v1beta1.Param{
				Name:  key,
				Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: defaultValue},
			})
		}
	}

	for k, v := range additionalParams {
		params = append(params, v1beta1.Param{
			Name:  k,
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: v},
		})
	}

	return params
}

// GHContentJSON takes a string and returns the string of JSON we'd expect from GitHub for a file's contents, using a
// default filename and path
func GHContentJSON(content string) string {
	return GHContentJSONDetailed(content, "SOMEFILE", "SOMEPATH")
}

// GHContentJSONDetailed takes a string, a filename, and a path, and returns the string of JSON we'd expect from GitHub for a file's contents
func GHContentJSONDetailed(content, filename, path string) string {
	encContent := base64.StdEncoding.EncodeToString([]byte(content))

	return fmt.Sprintf(`{
		  "type": "file",
          "content": "%s",
		  "encoding": "base64",
		  "size": 1234,
		  "name": "%s",
		  "path": "%s"
		}`, encContent, filename, path)
}

// SetupFakeGitHub configures a fake GitHub server and returns it
func SetupFakeGitHub() (*github.Client, *http.ServeMux, func()) {
	apiPath := "/api-v3"

	mux := http.NewServeMux()

	handler := http.NewServeMux()
	handler.Handle(apiPath+"/", http.StripPrefix(apiPath, mux))

	server := httptest.NewServer(handler)

	client := github.NewClient(nil)
	ghURL, _ := url.Parse(server.URL + apiPath + "/")
	client.BaseURL = ghURL
	client.UploadURL = ghURL

	return client, mux, server.Close
}
