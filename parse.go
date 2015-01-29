// Package parse provides a client for the Parse API.
package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/facebookgo/httperr"
)

// An Object Identifier.
type ID string

// Credentials to access an application.
type Credentials struct {
	ApplicationID ID
	RestApiKey    string
	MasterKey     string
}

// Describes Permissions for Read & Write.
type Permissions struct {
	Read  bool `json:"read,omitempty"`
	Write bool `json:"write,omitempty"`
}

// Check if other Permissions is equal.
func (p *Permissions) Equal(o *Permissions) bool {
	return p.Read == o.Read && p.Write == o.Write
}

// The required "name" field for Roles.
type RoleName string

// An ACL defines a set of permissions based on various facets.
type ACL map[string]*Permissions

// The key used by the API to represent public ACL permissions.
const PublicPermissionKey = "*"

// Permissions for the Public.
func (a ACL) Public() *Permissions {
	return a[PublicPermissionKey]
}

// Permissions for a specific user, if explicitly set.
func (a ACL) ForUserID(userID ID) *Permissions {
	return a[string(userID)]
}

// Permissions for a specific role name, if explicitly set.
func (a ACL) ForRoleName(roleName RoleName) *Permissions {
	return a["role:"+string(roleName)]
}

// Base Object.
type Object struct {
	ID        ID         `json:"objectId,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// User object.
type User struct {
	Object
	Email         string `json:"email,omitempty"`
	Username      string `json:"username,omitempty"`
	Phone         string `json:"phone,omitempty"`
	EmailVerified bool   `json:"emailVerified,omitempty"`
	SessionToken  string `json:"sessionToken,omitempty"`

	AuthData *struct {
		Facebook *struct {
			ID          string    `json:"id,omitempty"`
			AccessToken string    `json:"access_token,omitempty"`
			Expiration  time.Time `json:"expiration_date,omitempty"`
		} `json:"facebook,omitempty"`

		Twitter *struct {
			ID              string `json:"id,omitempty"`
			ScreenName      string `json:"screen_name,omitempty"`
			ConsumerKey     string `json:"consumer_key,omitempty"`
			ConsumerSecret  string `json:"consumer_secret,omitempty"`
			AuthToken       string `json:"auth_token,omitempty"`
			AuthTokenSecret string `json:"auth_token_secret,omitempty"`
		} `json:"twitter,omitempty"`

		Anonymous *struct {
			ID string `json:"id,omitempty"`
		} `json:"anonymous,omitempty"`
	} `json:"authData,omitempty"`
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
	client   *Client
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
		e.client.redactor(),
		e.request,
		e.response,
	).Error()
}

// The default base URL for the API.
var DefaultBaseURL = &url.URL{
	Scheme: "https",
	Host:   "api.parse.com",
	Path:   "/1/",
}

// Parse API Client.
type Client struct {
	// The underlying http.RoundTripper to perform the individual requests. When
	// nil http.DefaultTransport will be used.
	Transport http.RoundTripper

	// The base URL to parse relative URLs off. If you pass absolute URLs to Client
	// functions they are used as-is. When nil DefaultBaseURL will be used.
	BaseURL *url.URL

	// Application Credentials to be included in the API calls. When nil no
	// Credentials will be included.
	Credentials *Credentials

	// Redact sensitive information from errors when true.
	Redact bool
}

func (c *Client) transport() http.RoundTripper {
	if c.Transport == nil {
		return http.DefaultTransport
	}
	return c.Transport
}

// Perform a GET method call on the given url and unmarshal response into
// result.
func (c *Client) Get(u *url.URL, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "GET", URL: u}, nil, result)
}

// Perform a POST method call on the given url with the given body and
// unmarshal response into result.
func (c *Client) Post(u *url.URL, body, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "POST", URL: u}, body, result)
}

// Perform a PUT method call on the given url with the given body and
// unmarshal response into result.
func (c *Client) Put(u *url.URL, body, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "PUT", URL: u}, body, result)
}

// Perform a DELETE method call on the given url and unmarshal response into
// result.
func (c *Client) Delete(u *url.URL, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "DELETE", URL: u}, nil, result)
}

// Perform a Parse API call. This method modifies the request and adds the
// Authentication headers. The body is JSON encoded and for responses in the
// 2xx or 3xx range the response will be JSON decoded into result, for others
// an error of type Error will be returned.
func (c *Client) Do(req *http.Request, body, result interface{}) (*http.Response, error) {
	req.Proto = "HTTP/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1

	if req.URL == nil {
		if c.BaseURL == nil {
			req.URL = DefaultBaseURL
		} else {
			req.URL = c.BaseURL
		}
	} else {
		if !req.URL.IsAbs() {
			if c.BaseURL == nil {
				req.URL = DefaultBaseURL.ResolveReference(req.URL)
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

	if c.Credentials != nil {
		req.Header.Add(
			"X-Parse-Application-Id",
			string(c.Credentials.ApplicationID),
		)
		req.Header.Add("X-Parse-REST-API-Key", c.Credentials.RestApiKey)
	}

	// we need to buffer as Parse requires a Content-Length
	if body != nil {
		bd, err := json.Marshal(body)
		if err != nil {
			return nil, httperr.NewError(err, c.redactor(), req, nil)
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(bd))
		req.ContentLength = int64(len(bd))
	}

	res, err := c.transport().RoundTrip(req)
	if err != nil {
		return res, httperr.NewError(err, c.redactor(), req, res)
	}
	defer res.Body.Close()

	if res.StatusCode > 399 || res.StatusCode < 200 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, httperr.NewError(err, c.redactor(), req, res)
		}

		apiErr := &Error{
			StatusCode: res.StatusCode,
			request:    req,
			response:   res,
			client:     c,
		}
		if len(body) > 0 {
			err = json.Unmarshal(body, apiErr)
			if err != nil {
				return res, httperr.NewError(err, c.redactor(), req, res)
			}
		}
		return res, apiErr
	}

	if result == nil {
		_, err = io.Copy(ioutil.Discard, res.Body)
	} else {
		err = json.NewDecoder(res.Body).Decode(result)
	}
	if err != nil {
		return res, httperr.NewError(err, c.redactor(), req, res)
	}
	return res, nil
}

// Redact sensitive information from given string.
func (c *Client) redactor() httperr.Redactor {
	if !c.Redact || c.Credentials == nil || c.Credentials.MasterKey == "" {
		return httperr.RedactNoOp()
	}
	return strings.NewReplacer(
		c.Credentials.MasterKey,
		"-- REDACTED MASTER KEY --",
	)
}
