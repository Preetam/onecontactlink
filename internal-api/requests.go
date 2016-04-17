package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"

	"github.com/VividCortex/mysqlerr"
	"github.com/VividCortex/siesta"
	"github.com/go-sql-driver/mysql"

	"net/http"
)

const (
	requestKey = "request"
)

func readRequest(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var request schema.Request
	err := json.NewDecoder(r.Body).Decode(&request)
	if err == nil {
		c.Set(requestKey, request)
	} else {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		q()
	}
}

func createRequest(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	request := c.Get(requestKey).(schema.Request)
	if request.FromUser == 0 || request.ToUser == 0 {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = `missing "from" or "to" user ID`
		return
	}

	code := generateCode(8)
	now := time.Now().Unix()
	tx, err := requestData.DB.Begin()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	defer tx.Rollback()

	// Make sure both users exist

	userCount := 0
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE id = ? OR id = ?",
		request.FromUser, request.ToUser).Scan(&userCount)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	if userCount == 0 {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = "both user IDs must exist"
		return
	}

	request.Code = code
	request.Created = int(now)
	request.Updated = int(now)

	result, err := tx.Exec("INSERT INTO requests (code, from_user, to_user, created, updated) VALUES"+
		" (?, ?, ?, ?, ?)", request.Code, request.FromUser,
		request.ToUser, request.Created, request.Updated)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == mysqlerr.ER_DUP_ENTRY {
				// already exists
				requestData.StatusCode = http.StatusConflict
				log.Printf("[Req %s] %v", requestData.RequestID, err)
				return
			}
		}
		// Some other error
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	// We created a new request.
	lastID, err := result.LastInsertId()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	request.ID = int(lastID)
	requestData.ResponseData = request
}
