package main

import (
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"

	"github.com/VividCortex/siesta"

	"log"
	"net/http"
)

func getToken(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	tokenValue := params.String("token", "", "API token")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	token := schema.Token{}
	err = requestData.DB.QueryRow("SELECT id, user, value, created, updated, deleted FROM tokens WHERE"+
		" value = ?", *tokenValue).
		Scan(&token.ID, &token.User, &token.Value, &token.Created, &token.Updated, &token.Deleted)
	if err != nil {
		requestData.StatusCode = http.StatusNotFound
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	requestData.ResponseData = token
}
