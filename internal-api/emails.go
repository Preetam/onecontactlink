package main

import (
	"database/sql"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"

	"github.com/VividCortex/siesta"

	"log"
	"net/http"
)

func getEmailByAddress(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	var params siesta.Params
	address := params.String("address", "", "Email address")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	email := schema.Email{
		Address: *address,
	}
	err = requestData.DB.QueryRow("SELECT id, user, created, updated FROM emails WHERE address = ?",
		email.Address).Scan(&email.ID, &email.User, &email.Created, &email.Updated)
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
	requestData.ResponseData = email
}
