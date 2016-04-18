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
	ErrNotFound = errors.New("client: not found")
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
	resp := middleware.APIResponse{
		Data: &request,
	}
	err := c.doRequest("POST", "/requests", request, &resp)
	if err != nil {
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusConflict {
				return request.ID, ErrConflict
			}
		}
		return 0, err
	}
	return request.ID, nil
}

func (c *Client) GetRequestLinkByCode(code string) (*schema.RequestLink, error) {
	requestLink := schema.RequestLink{}
	resp := middleware.APIResponse{
		Data: &requestLink,
	}
	err := c.doRequest("GET", fmt.Sprintf("/links/requestLinks/%s", code), nil, &resp)
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
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusNotFound {
				return nil, ErrNotFound
			}
		}
		return nil, err
	}
	return &user, nil
}

func (c *Client) CreateUser(user *schema.User) (*schema.User, error) {
	resp := middleware.APIResponse{
		Data: user,
	}
	err := c.doRequest("POST", "/users", user, &resp)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (c *Client) GetEmail(address string) (*schema.Email, error) {
	email := schema.Email{}
	resp := middleware.APIResponse{
		Data: &email,
	}
	err := c.doRequest("GET", fmt.Sprintf("/emails/%s", address), nil, &resp)
	if err != nil {
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusNotFound {
				return nil, ErrNotFound
			}
		}
		return nil, err
	}
	return &email, nil
}

func (c *Client) SendRequestEmail(id int) error {
	return c.doRequest("POST", fmt.Sprintf("/requests/%d/sendEmail", id), nil, nil)
}

func (c *Client) GetRequestByCode(code string) (*schema.Request, error) {
	request := schema.Request{}
	resp := middleware.APIResponse{
		Data: &request,
	}
	err := c.doRequest("GET", fmt.Sprintf("/links/requests/%s", code), nil, &resp)
	if err != nil {
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusNotFound {
				return nil, ErrNotFound
			}
		}
		return nil, err
	}
	return &request, nil
}

func (c *Client) ManageRequest(id int, action string) error {
	if action != "approve" && action != "reject" {
		return fmt.Errorf("client: invalid action")
	}
	return c.doRequest("POST", fmt.Sprintf("/requests/%d/manage?action=%s", id, action), nil, nil)
}
