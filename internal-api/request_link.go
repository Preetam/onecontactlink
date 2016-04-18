package main

import (
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"

	"github.com/VividCortex/siesta"

	"log"
	"net/http"
)

func getRequestLinkByCode(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	requestLinkCode := params.String("requestLinkCode", "", "Request link")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	requestLink := schema.RequestLink{
		Code: *requestLinkCode,
	}
	err = requestData.DB.QueryRow("SELECT id, user, created, updated, deleted FROM request_links"+
		" WHERE code = ?", requestLink.Code).
		Scan(&requestLink.ID, &requestLink.User, &requestLink.Created,
			&requestLink.Updated, &requestLink.Deleted)
	if err != nil {
		requestData.StatusCode = http.StatusNotFound
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	requestData.ResponseData = requestLink
}
