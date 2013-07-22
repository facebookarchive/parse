package parse_test

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/daaku/go.flagconfig"
	"github.com/daaku/go.flagenv"
	"github.com/daaku/go.httpcontrol"
	"github.com/daaku/go.parse"
	"github.com/daaku/go.urlbuild"
)

var (
	defaultHttpTransport = &httpcontrol.Transport{
		MaxIdleConnsPerHost:   50,
		DialTimeout:           time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		RequestTimeout:        time.Minute,
		Stats:                 logRequestHandler,
	}
	defaultHttpClient  = &http.Client{Transport: defaultHttpTransport}
	defaultCredentials = parse.CredentialsFlag("parsetest")
	defaultParseClient = &parse.Client{
		Credentials: defaultCredentials,
		HttpClient:  defaultHttpClient,
	}

	logRequest = flag.Bool(
		"log-requests",
		false,
		"will trigger verbose logging of requests",
	)
)

func init() {
	defaultCredentials.ApplicationID = "spAVcBmdREXEk9IiDwXzlwe0p4pO7t18KFsHyk7j"
	defaultCredentials.JavaScriptKey = "7TzsJ3Dgmb1WYPALhYX6BgDhNo99f5QCfxLZOPmO"
	defaultCredentials.MasterKey = "3gPo5M3TGlFPvAaod7N4iSEtCmgupKZMIC2DoYJ3"
	defaultCredentials.RestApiKey = "t6ON64DfTrTL4QJC322HpWbhN6fzGYo8cnjVttap"

	flag.Usage = flagconfig.Usage
	flagconfig.Parse()
	flagenv.Parse()
	if err := defaultHttpTransport.Start(); err != nil {
		panic(err)
	}
}

func logRequestHandler(stats *httpcontrol.Stats) {
	if *logRequest {
		fmt.Println(stats.String())
		fmt.Println("Header", stats.Request.Header)
	}
}

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
	userID1 := parse.ID("user1")
	userID1Permission := &parse.Permissions{}
	userID2 := parse.ID("user2")
	userID2Permission := &parse.Permissions{Read: true}
	userID3 := parse.ID("user3")
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
		Request *parse.Request
		Error   string
	}{
		{
			Request: &parse.Request{Method: "GET"},
			Error:   `no URL provided`,
		},
		{
			Request: &parse.Request{Method: "GET", URL: &url.URL{}},
			Error:   `Get : unsupported protocol scheme ""`,
		},
		{
			Request: &parse.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "www.eadf5cfd365145e99d2a3ddeec5d5f00.com",
					Path:   "/",
				},
			},
			Error: `Get https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com/: ` +
				`lookup www.eadf5cfd365145e99d2a3ddeec5d5f00.com: no such host`,
		},
		{
			Request: &parse.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "api.parse.com",
					Path:   "/1/classes/Foo/Bar",
				},
			},
			Error: `GET request for URL https://api.parse.com/1/classes/Foo/Bar ` +
				`failed with code 101 and message object not found for get`,
		},
		{
			Request: &parse.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme:   "https",
					Host:     "api.parse.com",
					Path:     "/1/classes/Foo/Bar",
					RawQuery: "a=1",
				},
			},
			Error: `request for URL "https://api.parse.com/1/classes/Foo/Bar?a=1" ` +
				`failed with error URL cannot include query, use Params instead`,
		},
		{
			Request: &parse.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "api.parse.com",
					Path:   "/1/classes/Foo/Bar",
				},
				Params: []urlbuild.Param{parse.ParamLimit(0)},
			},
			Error: `GET request for URL ` +
				`https://api.parse.com/1/classes/Foo/Bar?limit=0 ` +
				`failed with code 101 and message object not found for get`,
		},
		{
			Request: &parse.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "api.parse.com",
					Path:   "/1/classes/Foo/Bar",
				},
				Params: []urlbuild.Param{parse.ParamWhere(map[int]int{})},
			},
			Error: `request for URL "https://api.parse.com/1/classes/Foo/Bar" ` +
				`failed with error for "where" json: unsupported type: map[int]int`,
		},
		{
			Request: &parse.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "api.parse.com",
					Path:   "/1/classes/Foo/Bar",
				},
				Body: map[int]int{},
			},
			Error: `GET request for URL "https://api.parse.com/1/classes/Foo/Bar" ` +
				`failed with error json: unsupported type: map[int]int`,
		},
	}

	t.Parallel()
	for _, ec := range cases {
		err := defaultParseClient.Do(ec.Request, nil)
		if err == nil {
			t.Fatal("was expecting error")
		}
		if actual := err.Error(); actual != ec.Error {
			t.Fatalf(`expected "%s" got "%s"`, ec.Error, actual)
		}
	}
}

func TestInvalidUnauthorizedRequest(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		Credentials: &parse.Credentials{},
		HttpClient:  defaultHttpClient,
	}
	u, err := parse.DefaultBaseURL.Parse("classes/Foo/Bar")
	if err != nil {
		t.Fatal(err)
	}
	req := parse.Request{Method: "GET", URL: u}
	err = c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const msg = `GET request for URL https://api.parse.com/1/classes/Foo/Bar ` +
		`failed with http status 401 Unauthorized and message unauthorized`
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}
}

func TestRedact(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		Credentials: &parse.Credentials{
			JavaScriptKey: "js-key",
			MasterKey:     "ms-key",
		},
		HttpClient: defaultHttpClient,
	}
	p := "/_JavaScriptKey=js-key&_MasterKey=ms-key"
	u := &url.URL{
		Scheme: "https",
		Host:   "www.eadf5cfd365145e99d2a3ddeec5d5f00.com",
		Path:   p,
	}

	req := parse.Request{Method: "GET", URL: u}
	err := c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	msg := fmt.Sprintf(
		`Get https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com%s: `+
			`lookup www.eadf5cfd365145e99d2a3ddeec5d5f00.com: no such host`,
		p,
	)
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}

	c.Redact = true
	err = c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const redacted = `Get ` +
		`https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com/_JavaScriptKey=-- ` +
		`REDACTED JAVASCRIPT KEY --&_MasterKey=-- REDACTED MASTER KEY --: ` +
		`lookup www.eadf5cfd365145e99d2a3ddeec5d5f00.com: no such host`
	if actual := err.Error(); actual != redacted {
		t.Fatalf(`expected "%s" got "%s"`, redacted, actual)
	}
}

func TestPostDeleteObject(t *testing.T) {
	t.Parallel()
	type obj struct {
		Answer int `json:"answer"`
	}

	oPostURL, err := parse.DefaultBaseURL.Parse("classes/Foo")
	if err != nil {
		t.Fatal(err)
	}

	oPost := &obj{Answer: 42}
	oPostResponse := &parse.Object{}
	oPostReq := parse.Request{Method: "POST", URL: oPostURL, Body: oPost}
	err = defaultParseClient.Do(&oPostReq, oPostResponse)
	if err != nil {
		t.Fatal(err)
	}
	if oPostResponse.ID == "" {
		t.Fatal("did not get an ID in the response")
	}

	p := fmt.Sprintf("classes/Foo/%s", oPostResponse.ID)
	oGetURL, err := parse.DefaultBaseURL.Parse(p)
	if err != nil {
		t.Fatal(err)
	}

	oGet := &obj{}
	oGetReq := parse.Request{Method: "GET", URL: oGetURL}
	err = defaultParseClient.Do(&oGetReq, oGet)
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

	oDelReq := parse.Request{Method: "DELETE", URL: oGetURL}
	err = defaultParseClient.Do(&oDelReq, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostDeleteObjectUsingObjectClass(t *testing.T) {
	t.Parallel()
	type obj struct {
		parse.Object
		Answer int `json:"answer"`
	}

	u, err := parse.DefaultBaseURL.Parse(
		"classes/TestPostDeleteObjectUsingObjectClass/",
	)
	if err != nil {
		t.Fatal(err)
	}

	foo := &parse.ObjectClient{
		Client:  defaultParseClient,
		BaseURL: u,
	}

	oPost := &obj{Answer: 42}
	oPostResponse, err := foo.Post(oPost)
	if err != nil {
		t.Fatal(err)
	}
	if oPostResponse.ID == "" {
		t.Fatal("did not get an ID in the response")
	}

	oGet := &obj{}
	err = foo.Get(oPostResponse.ID, oGet)
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

	if err := foo.Delete(oGet.ID); err != nil {
		t.Fatal(err)
	}
}
