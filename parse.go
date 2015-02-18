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

	"github.com/facebookgo/httperr"
)

const (
	userAgentHeader    = "User-Agent"
	userAgent          = "go-parse-2015-01-31"
	masterKeyHeader    = "X-Parse-Master-Key"
	restAPIKeyHeader   = "X-Parse-REST-API-Key"
	sessionTokenHeader = "X-Parse-Session-Token"
)

var (
	errEmptyMasterKey     = errors.New("parse: cannot use empty MasterKey Credentials")
	errEmptyRestAPIKey    = errors.New("parse: cannot use empty RestAPIKey Credentials")
	errEmptySessionToken  = errors.New("parse: cannot use empty SessionToken Credentials")
	errEmptyApplicationID = errors.New("parse: cannot use empty ApplicationID")
)

// Credentials allows for adding authentication information to a request.
type Credentials interface {
	Modify(r *http.Request) error
}

// MasterKey adds the Master Key to the request.
type MasterKey string

// Modify adds the Master Key header.
func (m MasterKey) Modify(r *http.Request) error {
	if m == "" {
		return errEmptyMasterKey
	}
	r.Header.Set(masterKeyHeader, string(m))
	return nil
}

// RestAPIKey adds the Rest API Key to the request.
type RestAPIKey string

// Modify adds the Rest API Key header.
func (k RestAPIKey) Modify(r *http.Request) error {
	if k == "" {
		return errEmptyRestAPIKey
	}
	r.Header.Set(restAPIKeyHeader, string(k))
	return nil
}

// SessionToken adds the Rest API Key and the Session Token to the request.
type SessionToken struct {
	RestAPIKey   string
	SessionToken string
}

// Modify adds the Session Token header.
func (t SessionToken) Modify(r *http.Request) error {
	if t.RestAPIKey == "" {
		return errEmptyRestAPIKey
	}
	if t.SessionToken == "" {
		return errEmptySessionToken
	}
	r.Header.Set(restAPIKeyHeader, string(t.RestAPIKey))
	r.Header.Set(sessionTokenHeader, string(t.SessionToken))
	return nil
}

// An Error from the Parse API.
type Error struct {
	// These are provided by the Parse API and may not always be available.
	Message string `json:"error"`
	Code    int    `json:"code"`

	// This is the HTTP StatusCode.
	StatusCode int `json:"-"`

	request  *http.Request
	response *http.Response
}

func (e *Error) Error() string {
	var buf bytes.Buffer
	if e.Code != 0 {
		fmt.Fprintf(&buf, "code %d", e.Code)
	}
	if e.Code != 0 && e.Message != "" {
		fmt.Fprint(&buf, " and ")
	}
	if e.Message != "" {
		fmt.Fprintf(&buf, "message %s", e.Message)
	}
	return httperr.NewError(
		errors.New(buf.String()),
		httperr.RedactNoOp(),
		e.request,
		e.response,
	).Error()
}

// The default base URL for the API.
var defaultBaseURL = &url.URL{
	Scheme: "https",
	Host:   "api.parse.com",
	Path:   "/1/",
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

	// Application ID must always be specified.
	ApplicationID string

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

// Do performs a Parse API call. This method modifies the request and adds the
// Authentication headers. The body is JSON encoded and for responses in the
// 2xx or 3xx range the response will be JSON decoded into result, for others
// an error of type Error will be returned.
func (c *Client) Do(req *http.Request, body, result interface{}) (*http.Response, error) {
	req.Proto = "HTTP/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1

	if req.URL == nil {
		if c.BaseURL == nil {
			req.URL = defaultBaseURL
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
	if c.ApplicationID == "" {
		return nil, errEmptyApplicationID
	}
	req.Header.Add("X-Parse-Application-Id", c.ApplicationID)

	if c.Credentials != nil {
		if err := c.Credentials.Modify(req); err != nil {
			return nil, err
		}
	}

	// we need to buffer as Parse requires a Content-Length
	if body != nil {
		bd, err := json.Marshal(body)
		if err != nil {
			return nil, httperr.NewError(err, httperr.RedactNoOp(), req, nil)
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(bd))
		req.ContentLength = int64(len(bd))
	}

	res, err := c.transport().RoundTrip(req)
	if err != nil {
		return res, httperr.NewError(err, httperr.RedactNoOp(), req, res)
	}
	defer res.Body.Close()

	if res.StatusCode > 399 || res.StatusCode < 200 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, httperr.NewError(err, httperr.RedactNoOp(), req, res)
		}

		apiErr := &Error{
			StatusCode: res.StatusCode,
			request:    req,
			response:   res,
		}
		if len(body) > 0 {
			err = json.Unmarshal(body, apiErr)
			if err != nil {
				return res, httperr.NewError(err, httperr.RedactNoOp(), req, res)
			}
		}
		return res, apiErr
	}

	if result != nil {
		if err := json.NewDecoder(res.Body).Decode(result); err != nil {
			return res, httperr.NewError(err, httperr.RedactNoOp(), req, res)
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
