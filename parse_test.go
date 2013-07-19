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
)

var (
	httpTransport = &httpcontrol.Transport{
		MaxIdleConnsPerHost:   50,
		DialTimeout:           time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		RequestTimeout:        time.Minute,
		Stats:                 logRequestHandler,
	}
	httpClient         = &http.Client{Transport: httpTransport}
	defaultCredentials = parse.CredentialsFlag("parsetest")
	defaultTestClient  = &parse.Client{
		Credentials: defaultCredentials,
		HttpClient:  httpClient,
	}
	logRequest = flag.Bool(
		"log-requests",
		false,
		"will trigger verbose logging of requests",
	)
)

func init() {
	flag.Usage = flagconfig.Usage
	flagconfig.Parse()
	flagenv.Parse()
	if defaultCredentials.ApplicationID == "" {
		defaultCredentials.ApplicationID = "spAVcBmdREXEk9IiDwXzlwe0p4pO7t18KFsHyk7j"
		defaultCredentials.JavaScriptKey = "7TzsJ3Dgmb1WYPALhYX6BgDhNo99f5QCfxLZOPmO"
		defaultCredentials.MasterKey = "3gPo5M3TGlFPvAaod7N4iSEtCmgupKZMIC2DoYJ3"
		defaultCredentials.RestApiKey = "t6ON64DfTrTL4QJC322HpWbhN6fzGYo8cnjVttap"
	}
	if err := httpTransport.Start(); err != nil {
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

func TestInvalidPath(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		BaseURL:     &url.URL{},
		HttpClient:  httpClient,
		Credentials: &parse.Credentials{},
	}
	req := parse.Request{Method: "GET", Path: ":"}
	err := c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const msg = `request for path ":" failed with error parse :: missing protocol scheme`
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}
}

func TestInvalidBaseURLWith(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		BaseURL:     &url.URL{},
		HttpClient:  httpClient,
		Credentials: &parse.Credentials{},
	}
	req := parse.Request{Method: "GET"}
	err := c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const msg = `Get : unsupported protocol scheme ""`
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}
}

func TestUnreachableURL(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   "www.eadf5cfd365145e99d2a3ddeec5d5f00.com",
			Path:   "/",
		},
		HttpClient:  httpClient,
		Credentials: &parse.Credentials{},
	}
	req := parse.Request{Method: "GET", Path: "/"}
	err := c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const msg = `Get https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com/: lookup www.eadf5cfd365145e99d2a3ddeec5d5f00.com: no such host`
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}
}

func TestInvalidUnauthorizedRequest(t *testing.T) {
	t.Parallel()
	c := &parse.Client{
		HttpClient:  httpClient,
		Credentials: &parse.Credentials{},
	}
	req := parse.Request{Method: "GET", Path: "classes/Foo/Bar"}
	err := c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const msg = `GET request for URL https://api.parse.com/1/classes/Foo/Bar failed with http status 401 Unauthorized and message unauthorized`
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
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   "www.eadf5cfd365145e99d2a3ddeec5d5f00.com",
			Path:   "/",
		},
		HttpClient: httpClient,
	}
	p := "/_JavaScriptKey=js-key&_MasterKey=ms-key"

	req := parse.Request{Method: "GET", Path: p}
	err := c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	msg := fmt.Sprintf(`Get https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com%s: lookup www.eadf5cfd365145e99d2a3ddeec5d5f00.com: no such host`, p)
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}

	c.Redact = true
	err = c.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const redacted = `Get https://www.eadf5cfd365145e99d2a3ddeec5d5f00.com/_JavaScriptKey=-- REDACTED JAVASCRIPT KEY --&_MasterKey=-- REDACTED MASTER KEY --: lookup www.eadf5cfd365145e99d2a3ddeec5d5f00.com: no such host`
	if actual := err.Error(); actual != redacted {
		t.Fatalf(`expected "%s" got "%s"`, redacted, actual)
	}
}

func TestInvalidGetRequest(t *testing.T) {
	t.Parallel()
	req := parse.Request{Method: "GET", Path: "classes/Foo/Bar"}
	err := defaultTestClient.Do(&req, nil)
	if err == nil {
		t.Fatal("was expecting error")
	}
	const msg = `GET request for URL https://api.parse.com/1/classes/Foo/Bar failed with code 101 and message object not found for get`
	if actual := err.Error(); actual != msg {
		t.Fatalf(`expected "%s" got "%s"`, msg, actual)
	}
}

func TestPostDeleteObject(t *testing.T) {
	t.Parallel()
	type obj struct {
		Answer int `json:"answer"`
	}

	oPost := &obj{Answer: 42}
	oPostResponse := &parse.Object{}
	oPostReq := parse.Request{Method: "POST", Path: "classes/Foo", Body: oPost}
	err := defaultTestClient.Do(&oPostReq, oPostResponse)
	if err != nil {
		t.Fatal(err)
	}
	if oPostResponse.ID == "" {
		t.Fatal("did not get an ID in the response")
	}

	p := fmt.Sprintf("classes/Foo/%s", oPostResponse.ID)
	oGet := &obj{}
	oGetReq := parse.Request{Method: "GET", Path: p}
	err = defaultTestClient.Do(&oGetReq, oGet)
	if err != nil {
		t.Fatal(err)
	}
	if oGet.Answer != oPost.Answer {
		t.Fatalf("did not get expected answer %d instead got %d", oPost.Answer, oGet.Answer)
	}

	oDelReq := parse.Request{Method: "DELETE", Path: p}
	err = defaultTestClient.Do(&oDelReq, nil)
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
	foo := &parse.ObjectDB{
		Client:    defaultTestClient,
		ClassName: "TestPostDeleteObjectUsingObjectClass",
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
		t.Fatalf("did not get expected answer %d instead got %d", oPost.Answer, oGet.Answer)
	}

	if err := foo.Delete(oGet.ID); err != nil {
		t.Fatal(err)
	}
}
