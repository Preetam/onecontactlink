package main

import (
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"

	"github.com/VividCortex/siesta"

	"database/sql"
	"log"
	"net/http"
)

func getToken(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	db := c.Get(middleware.DBKey).(*sql.DB)
	requestID := c.Get(middleware.RequestIDKey).(string)

	var params siesta.Params
	tokenValue := params.String("token", "", "API token")
	err := params.Parse(r.Form)
	if err != nil {
		c.Set(middleware.StatusCodeKey, http.StatusBadRequest)
		c.Set(middleware.ResponseErrorKey, err)
		log.Printf("[Req %s] %v", requestID, err)
		return
	}

	token := schema.Token{}
	err = db.QueryRow("SELECT id, user, value, created, updated, deleted FROM tokens WHERE"+
		" value = ?", *tokenValue).
		Scan(&token.ID, &token.User, &token.Value, &token.Created, &token.Updated, &token.Deleted)
	if err != nil {
		c.Set(middleware.StatusCodeKey, http.StatusNotFound)
		c.Set(middleware.ResponseErrorKey, err.Error())
		log.Printf("[Req %s] %v", requestID, err)
		return
	}
	c.Set(middleware.ResponseDataKey, token)
}
