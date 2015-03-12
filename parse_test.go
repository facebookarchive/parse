package parse_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
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
	defaultRestAPIKey    = parse.RestAPIKey{
		ApplicationID: defaultApplicationID,
		RestAPIKey:    "t6ON64DfTrTL4QJC322HpWbhN6fzGYo8cnjVttap",
	}
)

type transportFunc func(*http.Request) (*http.Response, error)

func (t transportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return t(r)
}

func jsonB(t testing.TB, v interface{}) []byte {
	b, err := json.Marshal(v)
	ensure.Nil(t, err)
	return b
}

func TestErrorCases(t *testing.T) {
	cases := []struct {
		Request    *http.Request
		Body       interface{}
		Error      string
		StatusCode int
		Transport  http.RoundTripper
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
			Error: "foo bar",
			Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("foo bar")
			}),
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
			Error:      `code 101 and message "object not found for get"`,
			StatusCode: http.StatusNotFound,
			Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				j := jsonB(t, parse.Error{
					Code:    101,
					Message: "object not found for get",
				})
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Status:     "404 Not Found",
					Body:       ioutil.NopCloser(bytes.NewReader(j)),
				}, nil
			}),
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
			Body:  map[int]int{},
			Error: "unsupported type: map[int]int",
			Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				panic("not reached")
			}),
		},
		{
			Request: &http.Request{
				Method: "GET",
				URL:    &url.URL{Path: "/"},
			},
			Error:      `invalid character '<' looking for beginning of value`,
			StatusCode: 404,
			Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Status:     "404 Not Found",
					Body:       ioutil.NopCloser(strings.NewReader("<html>")),
				}, nil
			}),
		},
	}

	t.Parallel()
	for _, ec := range cases {
		c := &parse.Client{
			Credentials: defaultRestAPIKey,
		}
		if !realTransport {
			c.Transport = ec.Transport
		}
		res, err := c.Do(ec.Request, ec.Body, nil)
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

func TestMethodHelpers(t *testing.T) {
	t.Parallel()
	expected := []string{"GET", "POST", "PUT", "DELETE"}
	count := 0
	c := &parse.Client{
		Credentials: defaultRestAPIKey,
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   "api.parse.com",
			Path:   "/1/classes/Foo/",
		},
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			ensure.DeepEqual(t, r.Method, expected[count])
			count++
			return nil, errors.New("")
		}),
	}
	c.Get(nil, nil)
	c.Post(nil, nil, nil)
	c.Put(nil, nil, nil)
	c.Delete(nil, nil)
	ensure.DeepEqual(t, count, len(expected))
}

func TestNilGetWithDefaultBaseURL(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	c := &parse.Client{
		Credentials: defaultRestAPIKey,
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			defer close(done)
			ensure.DeepEqual(t, r.URL.String(), "https://api.parse.com/1/")
			return nil, errors.New("")
		}),
	}
	c.Get(nil, nil)
	<-done
}

func TestRelativeGetWithDefaultBaseURL(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	c := &parse.Client{
		Credentials: defaultRestAPIKey,
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			defer close(done)
			ensure.DeepEqual(t, r.URL.String(), "https://api.parse.com/1/Foo")
			return nil, errors.New("")
		}),
	}
	c.Get(&url.URL{Path: "Foo"}, nil)
	<-done
}

func TestResolveReferenceWithBase(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	c := &parse.Client{
		Credentials: defaultRestAPIKey,
		BaseURL: &url.URL{
			Path: "/1/",
		},
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			defer close(done)
			ensure.DeepEqual(t, r.URL.String(), "/1/Foo")
			return nil, errors.New("")
		}),
	}
	c.Get(&url.URL{Path: "Foo"}, nil)
	<-done
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
			Credentials: defaultRestAPIKey,
			BaseURL:     u,
		}
		res := make(map[string]interface{})
		_, err = c.Get(nil, res)
		ensure.NotNil(t, err)
		server.CloseClientConnections()
		server.Close()
	}
}

func TestCustomHTTPTransport(t *testing.T) {
	t.Parallel()
	const message = "hello world"
	c := &parse.Client{
		Transport: transportFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New(message)
		}),
	}
	_, err := c.Do(&http.Request{}, nil, nil)
	ensure.Err(t, err, regexp.MustCompile(message))
}

func TestMasterKeyEmptyApplicationID(t *testing.T) {
	t.Parallel()
	var mk parse.MasterKey
	ensure.Err(t, mk.Modify(nil), regexp.MustCompile("empty ApplicationID"))
}

func TestEmptyMasterKey(t *testing.T) {
	t.Parallel()
	mk := parse.MasterKey{ApplicationID: defaultApplicationID}
	ensure.Err(t, mk.Modify(nil), regexp.MustCompile("empty MasterKey"))
}

func TestRestAPIKeyEmptyApplicationID(t *testing.T) {
	t.Parallel()
	var mk parse.RestAPIKey
	ensure.Err(t, mk.Modify(nil), regexp.MustCompile("empty ApplicationID"))
}

func TestEmptyRestAPIKey(t *testing.T) {
	t.Parallel()
	mk := parse.RestAPIKey{ApplicationID: defaultApplicationID}
	ensure.Err(t, mk.Modify(nil), regexp.MustCompile("empty RestAPIKey"))
}

func TestEmptySessionToken(t *testing.T) {
	t.Parallel()
	var st parse.SessionToken
	ensure.Err(t, st.Modify(nil), regexp.MustCompile("empty ApplicationID"))
}

func TestEmptySessionTokenMissingRestAPIKey(t *testing.T) {
	t.Parallel()
	st := parse.SessionToken{ApplicationID: defaultApplicationID}
	ensure.Err(t, st.Modify(nil), regexp.MustCompile("empty RestAPIKey"))
}

func TestEmptySessionTokenInSessionToken(t *testing.T) {
	t.Parallel()
	st := parse.SessionToken{
		ApplicationID: defaultApplicationID,
		RestAPIKey:    "rk",
	}
	ensure.Err(t, st.Modify(nil), regexp.MustCompile("empty SessionToken"))
}

func TestUserAgent(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	c := &parse.Client{
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
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
		Credentials: parse.RestAPIKey{},
	}
	_, err := c.Do(&http.Request{}, nil, nil)
	ensure.Err(t, err, regexp.MustCompile("empty ApplicationID"))
}

func TestAddCredentials(t *testing.T) {
	t.Parallel()
	const rk = "rk"
	const st = "st"
	done := make(chan struct{})
	c := &parse.Client{
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			defer close(done)
			ensure.DeepEqual(t, r.Header.Get("X-Parse-Application-ID"), defaultApplicationID)
			ensure.DeepEqual(t, r.Header.Get("X-Parse-Session-Token"), st)
			ensure.DeepEqual(t, r.Header.Get("X-Parse-REST-API-Key"), rk)
			return nil, errors.New("")
		}),
	}
	c = c.WithCredentials(parse.SessionToken{
		ApplicationID: defaultApplicationID,
		RestAPIKey:    rk,
		SessionToken:  st,
	})
	c.Do(&http.Request{}, nil, nil)
	<-done
}

func TestContentLengthHeader(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	c := &parse.Client{
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			defer close(done)
			ensure.DeepEqual(t, r.ContentLength, int64(4))
			return nil, errors.New("")
		}),
	}
	c.Post(nil, true, nil)
	<-done
}

func TestSuccessfulRequest(t *testing.T) {
	t.Parallel()
	expected := map[string]int{"answer": 42}
	c := &parse.Client{
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(jsonB(t, expected))),
			}, nil
		}),
	}
	var m map[string]int
	_, err := c.Post(nil, true, &m)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, m, expected)
}

func TestMasterKeyModify(t *testing.T) {
	t.Parallel()
	var req http.Request
	k := parse.MasterKey{
		ApplicationID: defaultApplicationID,
		MasterKey:     "42",
	}
	ensure.Nil(t, k.Modify(&req))
	ensure.DeepEqual(t, req.Header.Get("X-Parse-Application-ID"), k.ApplicationID)
	ensure.DeepEqual(t, req.Header.Get("X-Parse-Master-Key"), k.MasterKey)
}

func TestRestAPIKeyModify(t *testing.T) {
	t.Parallel()
	var req http.Request
	k := parse.RestAPIKey{
		ApplicationID: defaultApplicationID,
		RestAPIKey:    "42",
	}
	ensure.Nil(t, k.Modify(&req))
	ensure.DeepEqual(t, req.Header.Get("X-Parse-Application-ID"), k.ApplicationID)
	ensure.DeepEqual(t, req.Header.Get("X-Parse-REST-API-Key"), k.RestAPIKey)
}

func TestSessionTokenModify(t *testing.T) {
	t.Parallel()
	st := parse.SessionToken{
		ApplicationID: defaultApplicationID,
		RestAPIKey:    "42",
		SessionToken:  "43",
	}
	var req http.Request
	ensure.Nil(t, st.Modify(&req))
	ensure.DeepEqual(t, req.Header.Get("X-Parse-Application-ID"), st.ApplicationID)
	ensure.DeepEqual(t, req.Header.Get("X-Parse-REST-API-Key"), st.RestAPIKey)
	ensure.DeepEqual(t, req.Header.Get("X-Parse-Session-Token"), st.SessionToken)
}
