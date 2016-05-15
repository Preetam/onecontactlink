package linktoken

import "testing"

func TestLinkToken(t *testing.T) {
	c := NewTokenCodec(1, "example key 1234")

	data := map[string]interface{}{"request": float64(1)}
	expire := 100
	token := NewLinkToken(data, expire)
	res, err := c.EncodeToken(token)
	if err != nil {
		t.Fatal(err)
	}

	token, err = c.DecodeToken(res)
	if err != nil {
		t.Fatal(err)
	}

	if token.Expires != expire {
		t.Error("expected token.Expires to be %d, got %d", expire, token.Expires)
	}

	if _, ok := token.Data["request"].(float64); !ok {
		t.Errorf("expected token.Data[\"request\"] to be %f, got %v", 1, token.Data["request"])
	}
}
