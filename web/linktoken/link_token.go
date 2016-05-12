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
	"math"
	"math/big"
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
	Nonce   int                    `json:"nonce"`
	Data    map[string]interface{} `json:"data"`
	Expires int                    `json:"expires"`
}

func NewLinkToken(data map[string]interface{}, expires int) (*LinkToken, error) {
	nonce, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if err != nil {
		return nil, err
	}
	return &LinkToken{
		Nonce:   int(nonce.Int64()),
		Data:    data,
		Expires: expires,
	}, nil
}

func (c *TokenCodec) EncodeToken(token *LinkToken) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(token)

	block, err := aes.NewCipher([]byte(c.key))
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(buf.String()))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], buf.Bytes())
	data := ciphertext[aes.BlockSize:]

	ivStr := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(iv)
	dataStr := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
	versionStr := strconv.FormatInt(int64(c.keyVersion), 10)
	return versionStr + "." + ivStr + "." + dataStr, nil
}

func (c *TokenCodec) DecodeToken(tokenString string) (*LinkToken, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidLinkToken
	}

	versionStr := parts[0]
	iv, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(parts[2])
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
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(data, data)

	linkToken := LinkToken{}
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&linkToken)
	if err != nil {
		return nil, err
	}
	return &linkToken, nil
}
