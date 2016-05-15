package linktoken

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
)

var (
	ErrInvalidLinkToken = errors.New("link token invalid")
)

type TokenCodec struct {
	key        string
	keyVersion int
}

func NewTokenCodec(keyVersion int, key string) *TokenCodec {
	return &TokenCodec{
		key:        key,
		keyVersion: keyVersion,
	}
}

type LinkToken struct {
	Data    map[string]interface{} `json:"data"`
	Expires int                    `json:"expires"`
}

func NewLinkToken(data map[string]interface{}, expires int) *LinkToken {
	return &LinkToken{
		Data:    data,
		Expires: expires,
	}
}

func (c *TokenCodec) EncodeToken(token *LinkToken) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(token)

	block, err := aes.NewCipher([]byte(c.key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, buf.Bytes(), nil)

	nonceStr := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(nonce)
	dataStr := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(ciphertext)
	versionStr := strconv.FormatInt(int64(c.keyVersion), 10)
	return versionStr + "." + nonceStr + "." + dataStr, nil
}

func (c *TokenCodec) DecodeToken(tokenString string) (*LinkToken, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidLinkToken
	}

	versionStr := parts[0]
	nonce, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(parts[2])
	if err != nil {
		return nil, err
	}

	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		return nil, err
	}

	if c.keyVersion != int(version) {
		return nil, ErrInvalidLinkToken
	}

	block, err := aes.NewCipher([]byte(c.key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	data, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	linkToken := LinkToken{}
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&linkToken)
	if err != nil {
		return nil, err
	}
	return &linkToken, nil
}
