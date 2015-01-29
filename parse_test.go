package parse_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/facebookgo/parse"
)

var (
	defaultCredentials = &parse.Credentials{
		ApplicationID: "spAVcBmdREXEk9IiDwXzlwe0p4pO7t18KFsHyk7j",
		MasterKey:     "3gPo5M3TGlFPvAaod7N4iSEtCmgupKZMIC2DoYJ3",
		RestAPIKey:    "t6ON64DfTrTL4QJC322HpWbhN6fzGYo8cnjVttap",
	}
	defaultParseClient = &parse.Client{Credentials: defaultCredentials}
)

func TestPermissionEqual(t *testing.T) {
	t.Parallel()
	cases := []struct {
		p1, p2   *parse.Permissions
		expected bool
	}{
		{&parse.Permissions{Read: true}, &parse.Permissions{Read: true}, true},
		{&parse.Permissions{Read: true}, &parse.Permissions{}, false},
	}
	for n, c := range cases {
		if c.p1.Equal(c.p2) != c.expected {
			t.Fatalf("case %d was not as expected", n)
		}
	}
}

func TestACL(t *testing.T) {
	t.Parallel()
	publicPermission := &parse.Permissions{Read: true, Write: true}
	const (
		userID1 = "user1"
		userID2 = "user2"
		userID3 = "user3"
	)
	userID1Permission := &parse.Permissions{}
	userID2Permission := &parse.Permissions{Read: true}
	roleName1 := parse.RoleName("role1")
	roleName1Permission := &parse.Permissions{}
	roleName2 := parse.RoleName("role2")
	roleName2Permission := &parse.Permissions{}
	roleName3 := parse.RoleName("role3")
	acl := parse.ACL{
		parse.PublicPermissionKey:   publicPermission,
		string(userID1):             userID1Permission,
		string(userID2):             userID2Permission,
		"role:" + string(roleName1): roleName1Permission,
		"role:" + string(roleName2): roleName2Permission,
	}

	if !acl.Public().Equal(publicPermission) {
		t.Fatal("did not find expected public permission")
	}
	if !acl.ForUserID(userID1).Equal(userID1Permission) {
		t.Fatal("did not find expected userID1 permission")
	}
	if !acl.ForUserID(userID2).Equal(userID2Permission) {
		t.Fatal("did not find expected userID2 permission")
	}
	if acl.ForUserID(userID3) != nil {
		t.Fatal("did not find expected userID3 permission")
	}
	if !acl.ForRoleName(roleName1).Equal(roleName1Permission) {
		t.Fatal("did not find expected roleName1 permission")
	}
	if !acl.ForRoleName(roleName1).Equal(roleName2Permission) {
		t.Fatal("did not find expected roleName2 permission")
	}
	if acl.ForRoleName(roleName3) != nil {
		t.Fatal("did not find expected roleName3 permission")
	}
}

func TestErrorCases(t *testing.T) {
	cases := []struct {
		Request    *http.Request
		Body       interface{}
		Error      string
		StatusCode int
	}{
		{
			Request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "www.eadf5cfd365145e99d2a3ddeec5d5f00.com",
					Path:   "/",
				},
			},
			Error: `GET https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com/ ` +
				`failed with dial tcp: lookup ` +
				`www.eadf5cfd365145e99d2a3ddeec5d5f00.com: no such host`,
		},
		{
			Request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "api.parse.com",
					Path:   "/1/classes/Foo/Bar",
				},
			},
			Error: `GET https://api.parse.com/1/classes/Foo/Bar got 404 Not Found` +
				` failed with code 101 and message object not found for get`,
			StatusCode: 404,
		},
		{
			Request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "api.parse.com",
					Path:   "/1/classes/Foo/Bar",
				},
			},
			Body: map[int]int{},
			Error: `GET https://api.parse.com/1/classes/Foo/Bar failed with json:` +
				` unsupported type: map[int]int`,
		},
		{
			Request: &http.Request{
				Method: "GET",
				URL:    &url.URL{Path: "/"},
			},
			Error: `GET https://api.parse.com/ got 404 Not Found failed with` +
				` invalid character '<' looking for beginning of value`,
			StatusCode: 404,
		},
	}

	t.Parallel()
	for _, ec := range cases {
		res, err := defaultParseClient.Do(ec.Request, ec.Body, nil)
		if err == nil {
			t.Error("was expecting error")
		}
		if actual := err.Error(); actual != ec.Error {
			t.Errorf("expected\n%s\ngot\n%s", ec.Error, actual)
		}
		if ec.StatusCode != 0 {
			if res == nil {
				t.Error("did not get expected http.Response")
			}
			if res.StatusCode != ec.StatusCode {
				t.Errorf("expected %d got %d", ec.StatusCode, res.StatusCode)
			}
		}
	}
}

func TestInvalidUnauthorizedRequest(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		Credentials: &parse.Credentials{},
	}
	u, err := url.Parse("classes/Foo/Bar")
	if err != nil {
		t.Fatal(err)
	}
	req := http.Request{Method: "GET", URL: u}
	_, err = c.Do(&req, nil, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const msg = `GET https://api.parse.com/1/classes/Foo/Bar got 401 ` +
		`Unauthorized failed with message unauthorized`
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}
}

func TestRedact(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		Credentials: &parse.Credentials{
			MasterKey: "ms-key",
		},
	}
	p := "/_JavaScriptKey=js-key&_MasterKey=ms-key"
	u := &url.URL{
		Scheme: "https",
		Host:   "www.eadf5cfd365145e99d2a3ddeec5d5f00.com",
		Path:   p,
	}

	req := http.Request{Method: "GET", URL: u}
	_, err := c.Do(&req, nil, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	msg := fmt.Sprintf(
		`GET https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com%s failed `+
			`with dial tcp: lookup www.eadf5cfd365145e99d2a3ddeec5d5f00.com: `+
			`no such host`,
		p,
	)
	if actual := err.Error(); actual != msg {
		t.Fatalf("expected\n%s got\n%s", msg, actual)
	}

	c.Redact = true
	_, err = c.Do(&req, nil, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const redacted = `GET ` +
		`https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com/_JavaScriptKey=js-key` +
		`&_MasterKey=-- REDACTED MASTER KEY -- failed with dial tcp: ` +
		`lookup www.eadf5cfd365145e99d2a3ddeec5d5f00.com: no such host`
	if actual := err.Error(); actual != redacted {
		t.Fatalf("expected\n%s\ngot\n%s", redacted, actual)
	}
}

func TestPostDeleteObject(t *testing.T) {
	t.Parallel()
	type obj struct {
		Answer int `json:"answer"`
	}

	oPostURL, err := url.Parse("classes/Foo")
	if err != nil {
		t.Fatal(err)
	}

	oPost := &obj{Answer: 42}
	oPostResponse := &parse.Object{}
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

func TestMethodHelpers(t *testing.T) {
	t.Parallel()

	c := &parse.Client{
		Credentials: defaultCredentials,
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   "api.parse.com",
			Path:   "/1/classes/Foo/",
		},
	}

	type obj struct {
		Answer int `json:"answer"`
	}

	oPost := &obj{Answer: 42}
	oPostResponse := &parse.Object{}
	_, err := c.Post(nil, oPost, oPostResponse)
	if err != nil {
		t.Fatal(err)
	}
	if oPostResponse.ID == "" {
		t.Fatal("did not get an ID in the response")
	}

	oURL := &url.URL{Path: string(oPostResponse.ID)}

	oGet := &obj{}
	_, err = c.Get(oURL, oGet)
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

	oPut := &obj{Answer: 43}
	_, err = c.Put(oURL, oPut, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Get(oURL, oGet)
	if err != nil {
		t.Fatal(err)
	}
	if oGet.Answer != oPut.Answer {
		t.Fatalf(
			"did not get expected answer %d instead got %d",
			oPut.Answer,
			oGet.Answer,
		)
	}

	_, err = c.Delete(oURL, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNilGetWithDefaultBaseURL(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		Credentials: defaultCredentials,
	}
	_, err := c.Get(nil, nil)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	const expected = `GET https://api.parse.com/1/ got 404 Not Found failed ` +
		`with invalid character '<' looking for beginning of value`
	if err.Error() != expected {
		t.Fatalf(
			"did not get expected error\n%s instead got\n%s",
			expected,
			err,
		)
	}
}

func TestRelativeGetWithDefaultBaseURL(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		Credentials: defaultCredentials,
	}
	_, err := c.Get(&url.URL{Path: "Foo"}, nil)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	const expected = `GET https://api.parse.com/1/Foo got 404 Not Found failed` +
		` with invalid character '<' looking for beginning of value`
	if err.Error() != expected {
		t.Fatalf(
			"did not get expected error\n%s instead got\n%s",
			expected,
			err,
		)
	}
}

func TestServerAbort(t *testing.T) {
	t.Parallel()
	for _, code := range []int{200, 500} {
		server := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Add("Content-Length", "4000")
					w.WriteHeader(code)
					w.Write(bytes.Repeat([]byte("a"), 3000))
				},
			),
		)

		u, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		c := &parse.Client{
			Credentials: defaultCredentials,
			BaseURL:     u,
		}
		res := make(map[string]interface{})
		_, err = c.Get(nil, res)
		if err == nil {
			t.Fatalf("was expecting an error instead got %v", res)
		}
		expected := fmt.Sprintf(`GET %s`, server.URL)
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf(
				`did not contain expected error "%s" instead got "%s"`,
				expected,
				err,
			)
		}
		server.CloseClientConnections()
		server.Close()
	}
}

func TestEmptyClient(t *testing.T) {
	t.Parallel()
	c := &parse.Client{}
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
