package linktoken

import (
	"encoding/json"
	"testing"
)

type testTokenData struct {
	Request int `json:"request"`
}

func (d *testTokenData) MarshalJSON() ([]byte, error) {
	type X testTokenData
	x := X(*d)
	return json.Marshal(x)
}

func (d *testTokenData) UnmarshalJSON(data []byte) error {
	type X *testTokenData
	x := X(d)
	return json.Unmarshal(data, &x)
}

func TestLinkToken(t *testing.T) {
	c := NewTokenCodec(1, "example key 1234")
	data := testTokenData{
		Request: 1,
	}
	expire := 100
	token := NewLinkToken(&data, expire)
	res, err := c.EncodeToken(token)
	if err != nil {
		t.Fatal(err)
	}

	decodedData := testTokenData{}
	token, err = c.DecodeToken(res, &data)
	if err != nil {
		t.Fatal(err)
	}

	if token.Expires != expire {
		t.Error("expected token.Expires to be %d, got %d", expire, token.Expires)
	}

	if data.Request != token.Data.(*testTokenData).Request {
		t.Errorf("expected token.Request to be %d, got %d", data.Request, decodedData.Request)
	}
}
