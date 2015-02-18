// +build integration

package parse_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/facebookgo/parse"
)

var realTransport = true

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
	c := &parse.Client{
		ApplicationID: defaultApplicationID,
	}
	req := http.Request{Method: "GET", URL: &url.URL{Path: "classes/Foo/Bar"}}
	_, err := c.Do(&req, nil, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const msg = `GET https://api.parse.com/1/classes/Foo/Bar got 401 ` +
		`Unauthorized failed with message unauthorized`
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}
}
