package client

import (
	"io"
	"net/http"
)

type bucketClient struct {
	Client
}

func BucketClient(baseClient Client) bucketClient {
	return bucketClient{
		baseClient.For("buckets"),
	}
}

func (c bucketClient) PushBucket(bucket Object, file io.Reader) error {
	url := c.Client.ResourceURL(bucket.ID, "file")

	req, err := http.NewRequest("POST", url, file)
	if err != nil {
		return err
	}
	req.Header.Add("token", c.Client.Token.Token())
	req.Header.Add("Content-Type", "application/octet-stream")

	resp, err := c.Client.HTTP.Do(req)
	if err != nil {
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return ErrUnknown
	}
}

func (c bucketClient) PopBucket(bucket Object, file io.Writer) error {
	url := c.Client.ResourceURL(bucket.ID, "file")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("token", c.Client.Token.Token())

	resp, err := c.Client.HTTP.Do(req)
	if err != nil {
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return ErrUnknown
	}
	_, err = io.Copy(file, resp.Body)
	return err
}
