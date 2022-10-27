package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/token"
)

type Object struct {
	ID          string            `json:"id"`
	Name        string            `json:"name,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Finilizer   map[string]string `json:"finilizer,omitempty"`
	Version     int64             `json:"version"`
	CreatedAt   time.Time         `json:"created_at"`
	DeletedAt   *time.Time        `json:"deleted_at,omitempty"`

	Value any `json:"value"`
}

func (o Object) V(keys ...string) (any, bool) {
	var v func(any, []string) (any, bool)
	v = func(obj any, keys []string) (any, bool) {
		if len(keys) == 0 {
			return obj, true
		}
		m, ok := obj.(map[string]any)
		if !ok {
			return nil, false
		}
		child, ok := m[keys[0]]
		if !ok {
			return nil, false
		}
		return v(child, keys[1:])
	}
	return v(o.Value, keys)
}

type Client struct {
	ResourceName string
	Token        token.BaseToken
	APIServer    string
	HTTP         *http.Client
}

var (
	ErrEmptyID   = errors.New("empty id not acceptable")
	ErrNotFound  = errors.New("error not found")
	ErrForbidden = errors.New("forbidden")
	ErrConflict  = errors.New("conflict")
	ErrUnknown   = errors.New("unknown error")
)

func (c Client) For(resourceName string) Client {
	c.ResourceName = resourceName
	return c
}

func (c *Client) ResourceURL(parts ...string) string {
	j := strings.Join(parts, "/")
	if len(j) > 0 {
		j += "/"
	}
	return fmt.Sprintf("%s/%s/%s", c.APIServer, c.ResourceName, j)
}

func (c *Client) Get(id string) (*Object, error) {
	if id == "" {
		return nil, ErrEmptyID
	}

	resourceUrl := c.ResourceURL(id)
	req, err := http.NewRequest("GET", resourceUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("token", c.Token.Token())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusForbidden:
		return nil, ErrForbidden
	case http.StatusBadRequest:
		fallthrough
	default:
		return nil, ErrUnknown
	}

	var result Object
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Create(obj Object) error {
	if obj.ID == "" {
		return ErrEmptyID
	}
	resourceURL := c.ResourceURL()

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(obj); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", resourceURL, buf)
	if err != nil {
		return err
	}
	req.Header.Add("token", c.Token.Token())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusBadRequest:
		fallthrough
	default:
		return ErrUnknown
	}
}

func (c *Client) List(
	annotations map[string]string) ([]Object, error) {

	q := make(url.Values)
	for k, v := range annotations {
		q.Set(k, v)
	}

	resourceURL := c.ResourceURL() + "?" + q.Encode()
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("token", c.Token.Token())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden:
		return nil, ErrForbidden
	default:
		return nil, ErrUnknown
	}

	var result []Object
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) Update(obj *Object) error {
	if obj == nil || obj.ID == "" {
		return errors.New("empty id not acceptable")
	}
	resourceURL := c.ResourceURL(obj.ID)

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(obj); err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", resourceURL, buf)
	if err != nil {
		return err
	}
	req.Header.Add("token", c.Token.Token())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusConflict:
		return ErrConflict
	case http.StatusBadRequest:
		fallthrough
	default:
		return ErrUnknown
	}

	return json.NewDecoder(resp.Body).Decode(obj)
}

func (c *Client) Delete(obj Object) error {
	if obj.ID == "" {
		return errors.New("empty id not acceptable")
	}
	resourceURL := c.ResourceURL(obj.ID)

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(obj); err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", resourceURL, buf)
	if err != nil {
		return err
	}
	req.Header.Add("token", c.Token.Token())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusBadRequest:
		fallthrough
	default:
		return ErrUnknown
	}
}
