// The iam package provides types and functions for interaction with the AWS
// Identity and Access Management (IAM) service.
package iam

import (
	"encoding/xml"
	"launchpad.net/goamz/aws"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// The IAM type encapsulates operations operations with the IAM endpoint.
type IAM struct {
	aws.Auth
	aws.Region
}

// New creates a new IAM instance.
func New(auth aws.Auth, region aws.Region) *IAM {
	return &IAM{auth, region}
}

func (iam *IAM) query(params map[string]string, resp interface{}) error {
	params["Version"] = "2010-05-08"
	params["Timestamp"] = time.Now().In(time.UTC).Format(time.RFC3339)
	endpoint, err := url.Parse(iam.IAMEndpoint)
	if err != nil {
		return err
	}
	sign(iam.Auth, "GET", "/", params, endpoint.Host)
	endpoint.RawQuery = multimap(params).Encode()
	r, err := http.Get(endpoint.String())
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode > 200 {
		return buildError(r)
	}
	return xml.NewDecoder(r.Body).Decode(resp)
}

func buildError(r *http.Response) error {
	var (
		err    Error
		errors xmlErrors
	)
	xml.NewDecoder(r.Body).Decode(&errors)
	if len(errors.Errors) > 0 {
		err = errors.Errors[0]
	}
	err.StatusCode = r.StatusCode
	if err.Message == "" {
		err.Message = r.Status
	}
	return &err
}

func multimap(p map[string]string) url.Values {
	q := make(url.Values, len(p))
	for k, v := range p {
		q[k] = []string{v}
	}
	return q
}

// Response to a CreateUser request.
//
// See http://goo.gl/JS9Gz for more details.
type CreateUserResp struct {
	User      User   `xml:"CreateUserResult>User"`
	RequestId string `xml:"ResponseMetadata>RequestId"`
}

// User encapsulates a user managed by IAM.
//
// See http://goo.gl/BwIQ3 for more details.
type User struct {
	Arn  string
	Path string
	Id   string `xml:"UserId"`
	Name string `xml:"UserName"`
}

// CreateUser creates a new user in IAM.
//
// See http://goo.gl/JS9Gz for more details.
func (iam *IAM) CreateUser(name, path string) (*CreateUserResp, error) {
	params := map[string]string{
		"Action":   "CreateUser",
		"Path":     path,
		"UserName": name,
	}
	resp := new(CreateUserResp)
	if err := iam.query(params, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Response to a CreateAccessKey request.
//
// See http://goo.gl/L46Py for more details.
type CreateAccessKeyResp struct {
	AccessKey AccessKey `xml:"CreateAccessKeyResult>AccessKey"`
	RequestId string    `xml:"ResponseMetadata>RequestId"`
}

// AccessKey encapsulates an access key generated for a user.
//
// See http://goo.gl/LHgZR for more details.
type AccessKey struct {
	User   string `xml:"UserName"`
	Id     string `xml:"AccessKeyId"`
	Secret string `xml:"SecretAccessKey"`
	Status string
}

// CreateAccessKey creates a new access key in IAM.
//
// See http://goo.gl/L46Py for more details.
func (iam *IAM) CreateAccessKey(userName string) (*CreateAccessKeyResp, error) {
	params := map[string]string{
		"Action":   "CreateAccessKey",
		"UserName": userName,
	}
	resp := new(CreateAccessKeyResp)
	if err := iam.query(params, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type xmlErrors struct {
	Errors []Error `xml:"Errors>Error"`
}

// Error encapsulates an IAM error.
type Error struct {
	// HTTP status code of the error.
	StatusCode int

	// AWS code of the error.
	Code string

	// Message explaining the error.
	Message string
}

func (e *Error) Error() string {
	var prefix string
	if e.Code != "" {
		prefix = e.Code + ": "
	}
	if prefix == "" && e.StatusCode > 0 {
		prefix = strconv.Itoa(e.StatusCode) + ": "
	}
	return prefix + e.Message
}
