package client

import (
	"github.com/Preetam/onecontactlink/schema"

	"fmt"
)

func (c Client) Authenticate(value string) (*schema.Token, error) {
	var token schema.Token
	err := c.doRequest("GET", fmt.Sprintf("/tokens/%s", value), nil, &token)
	return &token, err
}
