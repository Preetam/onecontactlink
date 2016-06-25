package main

import (
	// std
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
	// base
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	// vendor
	"github.com/VividCortex/siesta"
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

func getUserByID(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	var params siesta.Params
	userID := params.Int("id", 0, "User ID")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	user := schema.User{
		ID: *userID,
	}
	err = requestData.DB.QueryRow("SELECT users.name, users.code, emails.address, users.status,"+
		" users.created, users.updated FROM users JOIN emails"+
		" ON users.main_email = emails.id"+
		" WHERE users.id = ? AND users.main_email != 0", user.ID).Scan(&user.Name, &user.Code,
		&user.MainEmail, &user.Status, &user.Created, &user.Updated)
	if err != nil {
		if err == sql.ErrNoRows {
			requestData.StatusCode = http.StatusNotFound
			log.Printf("[Req %s] %v", requestData.RequestID, err)
			return
		}
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	requestData.ResponseData = user
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
		// Create email address
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

	// Create user
	user.Code = generateCode(schema.UserCodeSize)
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

	// Link email to user
	_, err = tx.Exec("UPDATE emails SET user = ? WHERE id = ?", user.ID, email.ID)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	// Generate a request link
	_, err = tx.Exec("INSERT INTO request_links (user, code, created, updated)"+
		" VALUES (?,?,?,?)", user.ID, generateCode(schema.RequestLinkCodeSize), now, now)
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
