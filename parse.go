// Package parse provides a client for the Parse API.
package parse

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// An Object Identifier.
type ID string

// Credentials to access an application.
type Credentials struct {
	ApplicationID ID
	JavaScriptKey string
	MasterKey     string
	RestApiKey    string
}

// Credentials configured via flags. For example, if name is "parse", it will
// provide:
//
//     -parse.application-id=abc123
//     -parse.javascript-key=def456
//     -parse.master-key=ghi789
func CredentialsFlag(name string) *Credentials {
	credentials := &Credentials{}
	flag.StringVar(
		(*string)(&credentials.ApplicationID),
		name+".application-id",
		"",
		name+" Application ID",
	)
	flag.StringVar(
		&credentials.JavaScriptKey,
		name+".javascript-key",
		"",
		name+" JavaScript Key",
	)
	flag.StringVar(
		&credentials.MasterKey,
		name+".master-key",
		"",
		name+" Master Key",
	)
	flag.StringVar(
		&credentials.RestApiKey,
		name+".rest-api-key",
		"",
		name+" REST API Key",
	)
	return credentials
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
	AuthData      *struct {
		Twitter *struct {
			ID              string `json:"id,omitempty"`
			ScreenName      string `json:"screen_name,omitempty"`
			ConsumerKey     string `json:"consumer_key,omitempty"`
			ConsumerSecret  string `json:"consumer_secret,omitempty"`
			AuthToken       string `json:"auth_token,omitempty"`
			AuthTokenSecret string `json:"auth_token_secret,omitempty"`
		} `json:"twitter,omitempty"`
		Facebook *struct {
			ID          string    `json:"id,omitempty"`
			AccessToken string    `json:"access_token,omitempty"`
			Expiration  time.Time `json:"expiration_date,omitempty"`
		} `json:"facebook,omitempty"`
		Anonymous *struct {
			ID string `json:"id,omitempty"`
		} `json:"anonymous,omitempty"`
	} `json:"authData,omitempty"`
}

// Redact known sensitive information.
func redactIf(c *Client, s string) string {
	if c.Redact {
		var args []string
		if c.Credentials.JavaScriptKey != "" {
			args = append(
				args,
				c.Credentials.JavaScriptKey,
				"-- REDACTED JAVASCRIPT KEY --",
			)
		}
		if c.Credentials.MasterKey != "" {
			args = append(args, c.Credentials.MasterKey, "-- REDACTED MASTER KEY --")
		}
		return strings.NewReplacer(args...).Replace(s)
	}
	return s
}

// An Error from the Parse API.
type Error struct {
	// These are provided by the Parse API and may not always be available.
	Message string `json:"error"`
	Code    int    `json:"code"`

	// Always contains the *http.Request.
	request *http.Request `json:"-"`

	// May contain the *http.Response including a readable Body.
	response *http.Response `json:"-"`

	client *Client `json:"-"`
}

func (e *Error) Error() string {
	var buf bytes.Buffer
	fmt.Fprintf(
		&buf,
		"%s request for URL %s failed with",
		e.request.Method,
		redactIf(e.client, e.request.URL.String()),
	)

	if e.Code != 0 {
		fmt.Fprintf(&buf, " code %d", e.Code)
	} else if e.response != nil {
		fmt.Fprintf(&buf, " http status %s", e.response.Status)
	}

	if e.Message != "" {
		fmt.Fprintf(&buf, " and message %s", redactIf(e.client, e.Message))
	}

	return buf.String()
}

// Redacts sensitive information from an existing error.
type redactError struct {
	actual error
	client *Client
}

func (e *redactError) Error() string {
	return redactIf(e.client, e.actual.Error())
}

// An internal error during request processing.
type internalError struct {
	// May contain the *http.Request.
	request *http.Request

	// May contain the *http.Response including a readable Body.
	response *http.Response

	// The actual error.
	actual error

	client *Client
}

func (e *internalError) Error() string {
	var buf bytes.Buffer
	fmt.Fprintf(
		&buf,
		`%s request for URL "%s"`,
		e.request.Method,
		redactIf(e.client, e.request.URL.String()),
	)

	fmt.Fprintf(
		&buf,
		" failed with error %s",
		redactIf(e.client, e.actual.Error()),
	)

	if e.response != nil {
		fmt.Fprintf(
			&buf,
			" http status %s (%d)",
			e.response.Status,
			e.response.StatusCode,
		)
	}

	return buf.String()
}

// The underlying Http Client.
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// The default base URL for the API.
var DefaultBaseURL = &url.URL{
	Scheme: "https",
	Host:   "api.parse.com",
	Path:   "/1/",
}

// Parse API Client.
type Client struct {
	Credentials *Credentials
	BaseURL     *url.URL
	HttpClient  HttpClient
	Redact      bool // Redact sensitive information from errors when true
}

// Perform a HEAD method call on the given url.
func (c *Client) Head(u *url.URL) (*http.Response, error) {
	return c.method("HEAD", u, nil, nil)
}

// Perform a GET method call on the given url and unmarshal response into
// result.
func (c *Client) Get(u *url.URL, result interface{}) (*http.Response, error) {
	return c.method("GET", u, nil, result)
}

// Perform a POST method call on the given url with the given body and
// unmarshal response into result.
func (c *Client) Post(u *url.URL, body, result interface{}) (*http.Response, error) {
	return c.method("POST", u, body, result)
}

// Perform a PUT method call on the given url with the given body and
// unmarshal response into result.
func (c *Client) Put(u *url.URL, body, result interface{}) (*http.Response, error) {
	return c.method("PUT", u, body, result)
}

// Perform a DELETE method call on the given url and unmarshal response into
// result.
func (c *Client) Delete(u *url.URL, result interface{}) (*http.Response, error) {
	return c.method("DELETE", u, nil, result)
}

// Method helper.
func (c *Client) method(method string, u *url.URL, body, result interface{}) (*http.Response, error) {
	if u == nil {
		if c.BaseURL == nil {
			u = DefaultBaseURL
		} else {
			u = c.BaseURL
		}
	} else {
		if !u.IsAbs() {
			if c.BaseURL == nil {
				u = DefaultBaseURL.ResolveReference(u)
			} else {
				u = c.BaseURL.ResolveReference(u)
			}
		}
	}

	req := &http.Request{
		Method:     method,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       u.Host,
		Header:     make(http.Header),
	}

	return c.Transport(req, body, result)
}

// Perform a Parse API call. This method modifies the request and adds the
// Authentication headers. The body is JSON encoded and for responses in the
// 2xx or 3xx range the response will be unmarshalled into result, for others
// an error of type Error will be returned.
func (c *Client) Transport(req *http.Request, body, result interface{}) (*http.Response, error) {
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Add("X-Parse-Application-Id", string(c.Credentials.ApplicationID))
	req.Header.Add("X-Parse-REST-API-Key", c.Credentials.RestApiKey)

	// we need to buffer as Parse requires a Content-Length
	if body != nil {
		bd, err := json.Marshal(body)
		if err != nil {
			return nil, &internalError{
				request: req,
				actual:  err,
				client:  c,
			}
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(bd))
		req.ContentLength = int64(len(bd))
	}

	res, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, &redactError{
			actual: err,
			client: c,
		}
	}
	defer res.Body.Close()

	if res.StatusCode > 399 || res.StatusCode < 200 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, &internalError{
				request:  req,
				response: res,
				actual:   err,
				client:   c,
			}
		}

		apiErr := &Error{
			request:  req,
			response: res,
			client:   c,
		}
		err = json.Unmarshal(body, apiErr)
		if err != nil {
			return res, &internalError{
				request:  req,
				response: res,
				actual:   err,
				client:   c,
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
		return res, &internalError{
			request:  req,
			response: res,
			actual:   err,
			client:   c,
		}
	}
	return res, nil
}
