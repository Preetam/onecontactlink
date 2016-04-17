package client

import (
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"

	"fmt"
)

type EmailMessage struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

func (c *Client) Authenticate(value string) (*schema.Token, error) {
	var token schema.Token
	resp := middleware.APIResponse{
		Data: &token,
	}
	err := c.doRequest("GET", fmt.Sprintf("/tokens/%s", value), nil, &resp)
	return &token, err
}

// CreateRequest creates a request. The return values are the ID for the created
// request and an error.
func (c *Client) CreateRequest(fromUser, toUser int) (int, error) {
	// TODO
	return 0, nil
}
