// +build integration

package parse_test

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

var (
	realTransport = true

	defaultParseClient = &parse.Client{
		Credentials: defaultRestAPIKey,
	}
)

func TestPostDeleteObject(t *testing.T) {
	t.Parallel()
	type obj struct {
		Answer int `json:"answer"`
	}

	oPostURL, err := url.Parse("classes/Foo")
	if err != nil {
		t.Fatal(err)
	}

	type Object struct {
		ID string `json:"objectId,omitempty"`
	}

	oPost := &obj{Answer: 42}
	oPostResponse := &Object{}
	oPostReq := http.Request{Method: "POST", URL: oPostURL}
	_, err = defaultParseClient.Do(&oPostReq, oPost, oPostResponse)
	if err != nil {
		t.Fatal(err)
	}
	if oPostResponse.ID == "" {
		t.Fatal("did not get an ID in the response")
	}

	p := fmt.Sprintf("classes/Foo/%s", oPostResponse.ID)
	oGetURL, err := url.Parse(p)
	if err != nil {
		t.Fatal(err)
	}

	oGet := &obj{}
	oGetReq := http.Request{Method: "GET", URL: oGetURL}
	_, err = defaultParseClient.Do(&oGetReq, nil, oGet)
	if err != nil {
		t.Fatal(err)
	}
	if oGet.Answer != oPost.Answer {
		t.Fatalf(
			"did not get expected answer %d instead got %d",
			oPost.Answer,
			oGet.Answer,
		)
	}

	oDelReq := http.Request{Method: "DELETE", URL: oGetURL}
	_, err = defaultParseClient.Do(&oDelReq, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMissingCredentials(t *testing.T) {
	t.Parallel()
	var c parse.Client
	req := http.Request{Method: "GET", URL: &url.URL{Path: "classes/Foo/Bar"}}
	_, err := c.Do(&req, nil, nil)
	ensure.NotNil(t, err)
	ensure.Err(t, err, regexp.MustCompile(`parse: api error with message="unauthorized"`))
}
