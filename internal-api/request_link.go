package main

import (
	"net/http"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	log "github.com/Sirupsen/logrus"
	"github.com/VividCortex/siesta"
)

func getRequestLinkByCode(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	requestLinkCode := params.String("requestLinkCode", "", "Request link")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
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
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	requestData.ResponseData = requestLink
}

func getRequestLinkByUser(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	user := params.Int("id", 0, "user ID")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	requestLink := schema.RequestLink{
		User: *user,
	}
	err = requestData.DB.QueryRow("SELECT id, code, created, updated FROM request_links"+
		" WHERE user = ? AND deleted = 0", requestLink.User).
		Scan(&requestLink.ID, &requestLink.Code, &requestLink.Created,
			&requestLink.Updated)
	if err != nil {
		requestData.StatusCode = http.StatusNotFound
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	requestData.ResponseData = requestLink
}
