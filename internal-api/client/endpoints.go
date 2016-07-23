package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
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
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusNotFound {
				return nil, ErrNotFound
			}
		}
		return nil, err
	}
	return &requestLink, nil
}

func (c *Client) GetRequestLinkByUser(user int) (*schema.RequestLink, error) {
	requestLink := schema.RequestLink{}
	resp := middleware.APIResponse{
		Data: &requestLink,
	}
	err := c.doRequest("GET", fmt.Sprintf("/users/%d/requestLink", user), nil, &resp)
	if err != nil {
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusNotFound {
				return nil, ErrNotFound
			}
		}
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

func (c *Client) ActivateUser(id int) error {
	err := c.doRequest("POST", fmt.Sprintf("/users/%d/activate", id), nil, nil)
	if err != nil {
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusNotFound {
				return ErrNotFound
			}
		}
		return err
	}
	return nil
}

func (c *Client) SendActivationEmail(id int) error {
	err := c.doRequest("POST", fmt.Sprintf("/users/%d/sendActivationEmail", id), nil, nil)
	if err != nil {
		if serverErr, ok := err.(ServerError); ok {
			if serverErr == http.StatusNotFound {
				return ErrNotFound
			}
		}
		return err
	}
	return nil
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

func (c *Client) GetUserEmails(user int) ([]schema.Email, error) {
	emails := []schema.Email{}
	resp := middleware.APIResponse{
		Data: &emails,
	}
	err := c.doRequest("GET", fmt.Sprintf("/users/%d/emails", user), nil, &resp)
	if err != nil {
		return nil, err
	}
	return emails, nil
}

func (c *Client) ValidateEmail(address string) (bool, error) {
	isValid := false
	resp := middleware.APIResponse{
		Data: &isValid,
	}
	err := c.doRequest("POST", fmt.Sprintf("/emails/%s/validate", address), nil, &resp)
	return isValid, err
}

func (c *Client) SendRequestEmail(id int) error {
	return c.doRequest("POST", fmt.Sprintf("/requests/%d/sendRequestEmail", id), nil, nil)
}

func (c *Client) SendContactInfoEmail(id int) error {
	return c.doRequest("POST", fmt.Sprintf("/requests/%d/sendContactInfoEmail", id), nil, nil)
}

func (c *Client) ManageRequest(id int, action string) error {
	if action != "approve" && action != "reject" {
		return fmt.Errorf("client: invalid action")
	}
	return c.doRequest("POST", fmt.Sprintf("/requests/%d/manage?action=%s", id, action), nil, nil)
}

func (c *Client) SendAuth(email string) error {
	return c.doRequest("POST", "/auth/send?email="+url.QueryEscape(email), nil, nil)
}
