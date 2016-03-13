package middleware

import (
	"github.com/VividCortex/siesta"

	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
)

type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func RequestIdentifier(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestID := fmt.Sprintf("%08x", rand.Intn(0xffffffff))
	c.Set(RequestIDKey, requestID)
	log.Printf("[Req %s] %s %s", requestID, r.Method, r.URL)
}

func ResponseGenerator(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	response := APIResponse{}

	if data := c.Get(ResponseDataKey); data != nil {
		response.Data = data
	}

	if err := c.Get(ResponseErrorKey); err != nil {
		response.Error = err.(string)
	}

	if response.Data != nil && response.Error != "" {
		c.Set(ResponseKey, response)
	}
}

func ResponseWriter(c siesta.Context, w http.ResponseWriter, r *http.Request,
	quit func()) {
	// Set the request ID header.
	if requestID := c.Get(RequestIDKey); requestID != nil {
		w.Header().Set("X-Request-Id", requestID.(string))
	}

	// Set the content type.
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)

	// If we have a status code set in the context,
	// send that in the header.
	//
	// Go defaults to 200 OK.
	statusCode := c.Get(StatusCodeKey)
	if statusCode != nil {
		statusCodeInt := statusCode.(int)
		w.WriteHeader(statusCodeInt)
	}

	// Check to see if we have some sort of response.
	response := c.Get(ResponseKey)
	if response != nil {
		// We'll encode it as JSON without knowing
		// what it exactly is.
		enc.Encode(response)
	}

	// We're at the end of the middleware chain, so quit.
	quit()
}
