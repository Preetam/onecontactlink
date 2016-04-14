package main

import (
	"database/sql"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"

	"github.com/VividCortex/siesta"

	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const (
	userKey = "user"
)

func readUser(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var user schema.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err == nil {
		c.Set(userKey, user)
	} else {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		q()
	}
}

func createUser(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	user := c.Get(userKey).(schema.User)

	if user.Name == "" || user.MainEmail == "" {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = "missing name or email"
		return
	}

	tx, err := requestData.DB.Begin()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	defer tx.Rollback()

	// Check if email address exists
	var email schema.Email
	err = tx.QueryRow("SELECT id, address, user FROM emails WHERE address = ?", user.MainEmail).
		Scan(&email.ID, &email.Address, &email.User)
	if err != nil && err != sql.ErrNoRows {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	now := time.Now().Unix()
	if email.ID != 0 {
		// Email already exists
		if email.User != 0 {
			// Email is already associated with another user
			requestData.StatusCode = http.StatusConflict
			requestData.ResponseError = "email already in use"
			return
		}
	} else {
		execResult, err := tx.Exec("INSERT INTO emails (address, created, updated)"+
			" VALUES (?, ?, ?)", user.MainEmail, now, now)
		emailID, err := execResult.LastInsertId()
		if err != nil {
			requestData.StatusCode = http.StatusInternalServerError
			log.Printf("[Req %s] %v", requestData.RequestID, err)
			return
		}
		email.ID = int(emailID)
	}

	user.Code = generateCode(4)
	user.Created = int(now)
	user.Updated = int(now)
	execResult, err := tx.Exec("INSERT INTO users (name, code, main_email, created, updated)"+
		" VALUES (?, ?, ?, ?, ?)", user.Name, user.Code, email.ID, now, now)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	userID, err := execResult.LastInsertId()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	user.ID = int(userID)

	_, err = tx.Exec("UPDATE emails SET user = ? WHERE id = ?", user.ID, email.ID)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	err = tx.Commit()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	requestData.ResponseData = user
}

func generateCode(length int) string {
	const valid = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = valid[rand.Intn(len(valid))]
	}
	return string(result)
}
