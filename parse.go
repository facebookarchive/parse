// Package parse provides a client for the Parse API.
package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	userAgentHeader     = "User-Agent"
	userAgent           = "go-parse-1"
	masterKeyHeader     = "X-Parse-Master-Key"
	restAPIKeyHeader    = "X-Parse-REST-API-Key"
	sessionTokenHeader  = "X-Parse-Session-Token"
	applicationIDHeader = "X-Parse-Application-ID"
)

var (
	errEmptyApplicationID = errors.New("parse: cannot use empty ApplicationID")
	errEmptyMasterKey     = errors.New("parse: cannot use empty MasterKey")
	errEmptyRestAPIKey    = errors.New("parse: cannot use empty RestAPIKey")
	errEmptySessionToken  = errors.New("parse: cannot use empty SessionToken")

	// The default base URL for the API.
	defaultBaseURL = url.URL{
		Scheme: "https",
		Host:   "api.parse.com",
		Path:   "/1/",
	}
)

// Credentials allows for adding authentication information to a request.
type Credentials interface {
	Modify(r *http.Request) error
}

// MasterKey adds the Master Key to the request.
type MasterKey struct {
	ApplicationID string
	MasterKey     string
}

// Modify adds the Master Key header.
func (m MasterKey) Modify(r *http.Request) error {
	if m.ApplicationID == "" {
		return errEmptyApplicationID
	}
	if m.MasterKey == "" {
		return errEmptyMasterKey
	}
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set(applicationIDHeader, string(m.ApplicationID))
	r.Header.Set(masterKeyHeader, string(m.MasterKey))
	return nil
}

// RestAPIKey adds the Rest API Key to the request.
type RestAPIKey struct {
	ApplicationID string
	RestAPIKey    string
}

// Modify adds the Rest API Key header.
func (k RestAPIKey) Modify(r *http.Request) error {
	if k.ApplicationID == "" {
		return errEmptyApplicationID
	}
	if k.RestAPIKey == "" {
		return errEmptyRestAPIKey
	}
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set(applicationIDHeader, string(k.ApplicationID))
	r.Header.Set(restAPIKeyHeader, string(k.RestAPIKey))
	return nil
}

// SessionToken adds the Rest API Key and the Session Token to the request.
type SessionToken struct {
	ApplicationID string
	RestAPIKey    string
	SessionToken  string
}

// Modify adds the Session Token header.
func (t SessionToken) Modify(r *http.Request) error {
	if t.ApplicationID == "" {
		return errEmptyApplicationID
	}
	if t.RestAPIKey == "" {
		return errEmptyRestAPIKey
	}
	if t.SessionToken == "" {
		return errEmptySessionToken
	}
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set(applicationIDHeader, string(t.ApplicationID))
	r.Header.Set(restAPIKeyHeader, string(t.RestAPIKey))
	r.Header.Set(sessionTokenHeader, string(t.SessionToken))
	return nil
}

// An Error from the Parse API. When a valid Parse JSON error is found, the
// returned error will be of this type.
type Error struct {
	Message string `json:"error"`
	Code    int    `json:"code"`
}

func (e *Error) Error() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "parse: api error with ")
	if e.Code != 0 {
		fmt.Fprintf(&buf, "code=%d", e.Code)
	}
	if e.Code != 0 && e.Message != "" {
		fmt.Fprint(&buf, " and ")
	}
	if e.Message != "" {
		fmt.Fprintf(&buf, "message=%q", e.Message)
	}
	return buf.String()
}

// A RawError with the HTTP StatusCode and Body. When a valid Parse JSON error
// is not found, the returned error will be of this type.
type RawError struct {
	StatusCode int
	Body       []byte
}

func (e *RawError) Error() string {
	return fmt.Sprintf("parse: error with status=%d and body=%q", e.StatusCode, e.Body)
}

// Client provides access to the Parse API.
type Client struct {
	// The underlying http.RoundTripper to perform the individual requests. When
	// nil http.DefaultTransport will be used.
	Transport http.RoundTripper

	// The base URL to parse relative URLs off. If you pass absolute URLs to
	// Client functions they are used as-is. When nil, the production Parse URL
	// will be used.
	BaseURL *url.URL

	// Credentials if set, will be included on every request.
	Credentials Credentials
}

func (c *Client) transport() http.RoundTripper {
	if c.Transport == nil {
		return http.DefaultTransport
	}
	return c.Transport
}

// Get performs a GET method call on the given url and unmarshal response into
// result.
func (c *Client) Get(u *url.URL, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "GET", URL: u}, nil, result)
}

// Post performs a POST method call on the given url with the given body and
// unmarshal response into result.
func (c *Client) Post(u *url.URL, body, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "POST", URL: u}, body, result)
}

// Put performs a PUT method call on the given url with the given body and
// unmarshal response into result.
func (c *Client) Put(u *url.URL, body, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "PUT", URL: u}, body, result)
}

// Delete performs a DELETE method call on the given url and unmarshal response
// into result.
func (c *Client) Delete(u *url.URL, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "DELETE", URL: u}, nil, result)
}

// RoundTrip performs a RoundTrip ignoring the request and response bodies. It
// is up to the caller to close them. This method modifies the request.
func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Proto = "HTTP/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1

	if req.URL == nil {
		if c.BaseURL == nil {
			req.URL = &defaultBaseURL
		} else {
			req.URL = c.BaseURL
		}
	} else {
		if !req.URL.IsAbs() {
			if c.BaseURL == nil {
				req.URL = defaultBaseURL.ResolveReference(req.URL)
			} else {
				req.URL = c.BaseURL.ResolveReference(req.URL)
			}
		}
	}

	if req.Host == "" {
		req.Host = req.URL.Host
	}

	if req.Header == nil {
		req.Header = make(http.Header)
	}

	req.Header.Add(userAgentHeader, userAgent)
	if c.Credentials != nil {
		if err := c.Credentials.Modify(req); err != nil {
			return nil, err
		}
	}

	res, err := c.transport().RoundTrip(req)
	if err != nil {
		return res, err
	}

	if res.StatusCode > 399 || res.StatusCode < 200 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, err
		}

		if len(body) > 0 {
			var apiErr Error
			if json.Unmarshal(body, &apiErr) == nil {
				return res, &apiErr
			}
		}
		return res, &RawError{
			StatusCode: res.StatusCode,
			Body:       body,
		}
	}

	return res, nil
}

// Do performs a Parse API call. This method modifies the request and adds the
// Authentication headers. The body is JSON encoded and for responses in the
// 2xx or 3xx range the response will be JSON decoded into result, for others
// an error of type Error will be returned.
func (c *Client) Do(req *http.Request, body, result interface{}) (*http.Response, error) {
	// we need to buffer as Parse requires a Content-Length
	if body != nil {
		bd, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		if req.Header == nil {
			req.Header = make(http.Header)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Body = ioutil.NopCloser(bytes.NewReader(bd))
		req.ContentLength = int64(len(bd))
	}

	res, err := c.RoundTrip(req)
	if err != nil {
		return res, err
	}
	defer res.Body.Close()

	if result != nil {
		if err := json.NewDecoder(res.Body).Decode(result); err != nil {
			return res, err
		}
	}
	return res, nil
}

// WithCredentials returns a new instance of the Client using the given
// Credentials. It discards the previous Credentials.
func (c *Client) WithCredentials(cr Credentials) *Client {
	var c2 Client
	c2 = *c
	c2.Credentials = cr
	return &c2
}
