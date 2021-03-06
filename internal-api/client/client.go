package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	http    *http.Client
	base    string
	headers map[string]string
}

type ServerError int

func (e ServerError) Error() string {
	return fmt.Sprintf("client: server status code %d", e)
}

func New(baseURI string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: time.Second * 30,
		},
		headers: map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		},
		base: strings.TrimRight(baseURI, "/"),
	}
}

func (c *Client) doRequest(verb string, address string, body, response interface{}) error {
	payload := bytes.NewBuffer(nil)
	if body != nil {
		err := json.NewEncoder(payload).Encode(body)
		if err != nil {
			return err
		}
	}

	request, err := http.NewRequest(verb, c.base+address, payload)
	if err != nil {
		return err
	}

	for key, val := range c.headers {
		request.Header.Set(key, val)
	}

	res, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		// Not a 2xx status code
		// Try to send a response back if we have something.
		if response != nil {
			json.NewDecoder(res.Body).Decode(response)
			// Ignore errors
		}
		return ServerError(res.StatusCode)
	}

	if response != nil {
		err := json.NewDecoder(res.Body).Decode(response)
		if err != nil {
			return err
		}
	}

	return nil
}
