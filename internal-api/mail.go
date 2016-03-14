package main

import (
	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"

	"github.com/VividCortex/siesta"
	"github.com/mailgun/mailgun-go"

	"encoding/json"
	"log"
	"net/http"
)

const (
	messageKey = "mailgun-message"
)

func emailMessageReader(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
	requestID := c.Get(middleware.RequestIDKey).(string)
	var message client.EmailMessage
	err := json.NewDecoder(r.Body).Decode(&message)
	if err == nil {
		c.Set(messageKey, message)
	} else {
		c.Set(middleware.StatusCodeKey, http.StatusBadRequest)
		c.Set(middleware.ResponseErrorKey, err.Error())
		log.Printf("[Req %s] %v", requestID, err)
		q()
	}
}

func sendEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	msg := c.Get(messageKey).(client.EmailMessage)
	requestID := c.Get(middleware.RequestIDKey).(string)

	_, _, err := mg.Send(mailgun.NewMessage(msg.From, msg.Subject, msg.Content, msg.To))
	if err != nil {
		c.Set(middleware.StatusCodeKey, http.StatusInternalServerError)
		c.Set(middleware.ResponseErrorKey, err.Error())
		log.Printf("[Req %s] %v", requestID, err)
		return
	}

	c.Set(middleware.StatusCodeKey, http.StatusNoContent)
}
