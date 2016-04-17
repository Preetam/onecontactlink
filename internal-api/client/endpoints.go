package client

import (
	"errors"
	"net/http"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"

	"fmt"
)

var (
	ErrConflict = errors.New("client: conflict")
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
	request := schema.Request{
		FromUser: fromUser,
		ToUser:   toUser,
	}
	err := c.doRequest("POST", "/requests", request, &request)
	if err != nil {
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusConflict {
				return 0, ErrConflict
			}
		}
		return 0, err
	}
	return request.ID, nil
}

func (c *Client) GetRequestLink(code string) (*schema.RequestLink, error) {
	requestLink := schema.RequestLink{}
	resp := middleware.APIResponse{
		Data: &requestLink,
	}
	err := c.doRequest("GET", fmt.Sprintf("/links/requests/%s", code), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &requestLink, nil
}

func (c *Client) GetUser(id int) (*schema.User, error) {
	user := schema.User{}
	resp := middleware.APIResponse{
		Data: &user,
	}
	err := c.doRequest("GET", fmt.Sprintf("/users/%d", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
