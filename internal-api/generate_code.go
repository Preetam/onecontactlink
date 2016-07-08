package main

import (
	"crypto/rand"
)

func generateCode(length int) string {
	const valid = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	_, err := rand.Read(result)
	if err != nil {
		panic(err)
	}
	for i := 0; i < length; i++ {
		result[i] = valid[int(result[i])%len(valid)]
	}
	return string(result)
}
