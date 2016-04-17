package main

import "math/rand"

func generateCode(length int) string {
	const valid = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = valid[rand.Intn(len(valid))]
	}
	return string(result)
}
