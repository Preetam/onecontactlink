package middleware

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/VividCortex/siesta"
)

type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func RequestIdentifier(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := &RequestData{
		RequestID: fmt.Sprintf("%08x", rand.Intn(0xffffffff)),
		Start:     time.Now(),
	}
	c.Set(RequestDataKey, requestData)
	log.WithFields(log.Fields{
		"request_id": requestData.RequestID,
		"method":     r.Method,
		"url":        r.URL,
	}).Infof("[Req %s] %s %s", requestData.RequestID, r.Method, r.URL)
}

func ResponseGenerator(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(RequestDataKey).(*RequestData)
	response := APIResponse{}

	if data := requestData.ResponseData; data != nil {
		response.Data = data
	}

	response.Error = requestData.ResponseError

	if response.Data != nil || response.Error != "" {
		c.Set(ResponseKey, response)
	}
}

func ResponseWriter(c siesta.Context, w http.ResponseWriter, r *http.Request,
	quit func()) {
	requestData := c.Get(RequestDataKey).(*RequestData)
	// Set the request ID header.
	if requestData.RequestID != "" {
		w.Header().Set("X-Request-Id", requestData.RequestID)
	}

	// Set the content type.
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)

	// If we have a status code set in the context,
	// send that in the header.
	//
	// Go defaults to 200 OK.
	if requestData.StatusCode != 0 {
		w.WriteHeader(requestData.StatusCode)
	}

	// Check to see if we have some sort of response.
	response := c.Get(ResponseKey)
	if response != nil {
		// We'll encode it as JSON without knowing
		// what it exactly is.
		enc.Encode(response)
	}

	now := time.Now()
	latencyDur := now.Sub(requestData.Start)
	log.WithFields(log.Fields{
		"request_id":  requestData.RequestID,
		"method":      r.Method,
		"url":         r.URL,
		"status_code": requestData.StatusCode,
		"latency":     latencyDur.Seconds(),
	}).Infof("[Req %s] %s %s status code %d latency %v", requestData.RequestID, r.Method, r.URL,
		requestData.StatusCode, latencyDur)

	// We're at the end of the middleware chain, so quit.
	quit()
}
