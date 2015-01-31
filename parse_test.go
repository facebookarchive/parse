package parse_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

var (
	defaultApplicationID = "spAVcBmdREXEk9IiDwXzlwe0p4pO7t18KFsHyk7j"
	defaultRestAPIKey    = parse.RestAPIKey("t6ON64DfTrTL4QJC322HpWbhN6fzGYo8cnjVttap")
	defaultParseClient   = &parse.Client{
		ApplicationID: defaultApplicationID,
		Credentials:   defaultRestAPIKey,
	}
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
			Error: `GET https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com/ failed with`,
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
		ensure.StringContains(t, err.Error(), ec.Error)
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

func TestRedact(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		ApplicationID: defaultApplicationID,
		Credentials:   parse.MasterKey("ms-key"),
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
		`GET https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com%s failed with`,
		p,
	)
	ensure.StringContains(t, err.Error(), msg)

	c.Redact = true
	_, err = c.Do(&req, nil, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const redacted = `GET ` +
		`https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com/_JavaScriptKey=js-key` +
		`&_MasterKey=-- REDACTED MASTER KEY -- failed with`
	ensure.StringContains(t, err.Error(), redacted)
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
		ApplicationID: defaultApplicationID,
		Credentials:   defaultRestAPIKey,
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
		ApplicationID: defaultApplicationID,
		Credentials:   defaultRestAPIKey,
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
		ApplicationID: defaultApplicationID,
		Credentials:   defaultRestAPIKey,
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
			ApplicationID: defaultApplicationID,
			Credentials:   defaultRestAPIKey,
			BaseURL:       u,
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

type tansportFunc func(*http.Request) (*http.Response, error)

func (t tansportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return t(r)
}

func TestCustomHTTPTransport(t *testing.T) {
	t.Parallel()
	const message = "hello world"
	c := &parse.Client{
		ApplicationID: defaultApplicationID,
		Transport: tansportFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New(message)
		}),
	}
	_, err := c.Do(&http.Request{}, nil, nil)
	ensure.Err(t, err, regexp.MustCompile(message))
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

func TestEmptyMasterKey(t *testing.T) {
	t.Parallel()
	var mk parse.MasterKey
	ensure.Err(t, mk.Modify(nil), regexp.MustCompile("empty MasterKey"))
}

func TestEmptyRestAPIKey(t *testing.T) {
	t.Parallel()
	var mk parse.RestAPIKey
	ensure.Err(t, mk.Modify(nil), regexp.MustCompile("empty RestAPIKey"))
}

func TestEmptySessionToken(t *testing.T) {
	t.Parallel()
	var st parse.SessionToken
	ensure.Err(t, st.Modify(nil), regexp.MustCompile("empty RestAPIKey"))
}

func TestEmptySessionTokenInSessionToken(t *testing.T) {
	t.Parallel()
	st := parse.SessionToken{RestAPIKey: "rk"}
	ensure.Err(t, st.Modify(nil), regexp.MustCompile("empty SessionToken"))
}

func TestEmptyApplicationID(t *testing.T) {
	t.Parallel()
	var c parse.Client
	_, err := c.Do(&http.Request{}, nil, nil)
	ensure.Err(t, err, regexp.MustCompile("empty ApplicationID"))
}

func TestUserAgent(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	c := &parse.Client{
		ApplicationID: defaultApplicationID,
		Transport: tansportFunc(func(r *http.Request) (*http.Response, error) {
			defer close(done)
			ensure.NotDeepEqual(t, r.Header.Get("User-Agent"), "")
			return nil, errors.New("")
		}),
	}
	c.Do(&http.Request{}, nil, nil)
	<-done
}

func TestCredentiasModifyError(t *testing.T) {
	t.Parallel()
	c := parse.Client{
		ApplicationID: defaultApplicationID,
		Credentials:   parse.RestAPIKey(""),
	}
	_, err := c.Do(&http.Request{}, nil, nil)
	ensure.Err(t, err, regexp.MustCompile("empty RestAPIKey"))
}

func TestAddCredentials(t *testing.T) {
	t.Parallel()
	const rk = "rk"
	const st = "st"
	done := make(chan struct{})
	c := &parse.Client{
		ApplicationID: defaultApplicationID,
		Transport: tansportFunc(func(r *http.Request) (*http.Response, error) {
			defer close(done)
			ensure.DeepEqual(t, r.Header.Get("X-Parse-Session-Token"), st)
			ensure.DeepEqual(t, r.Header.Get("X-Parse-REST-API-Key"), rk)
			return nil, errors.New("")
		}),
	}
	c = c.WithCredentials(parse.SessionToken{
		RestAPIKey:   rk,
		SessionToken: st,
	})
	c.Do(&http.Request{}, nil, nil)
	<-done
}
