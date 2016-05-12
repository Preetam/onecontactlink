package linktoken

import "testing"

func TestLinkToken(t *testing.T) {
	c := NewTokenCodec(1, "example key 1234")

	data := map[string]interface{}{"request": 1}
	expire := 100
	token, err := NewLinkToken(data, expire)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.EncodeToken(token)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)

	token, err = c.DecodeToken(res)
	if err != nil {
		t.Fatal(err)
	}

	if token.Expires != expire {
		t.Error("expected token.Expires to be %d, got %d", expire, token.Expires)
	}

	if _, ok := token.Data["request"].(int); !ok {
		t.Errorf("expected token.Data[\"request\"] to be %d, got %v", 1, token.Data["request"])
	}
}