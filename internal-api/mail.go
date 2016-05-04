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
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var message client.EmailMessage
	err := json.NewDecoder(r.Body).Decode(&message)
	if err == nil {
		c.Set(messageKey, message)
	} else {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		q()
	}
}

func sendEmailHandler(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	msg := c.Get(messageKey).(client.EmailMessage)
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	err := sendMail(mg, msg)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	requestData.StatusCode = http.StatusNoContent
}

func sendMail(mg mailgun.Mailgun, msg client.EmailMessage) error {
	_, _, err := mg.Send(mailgun.NewMessage(msg.From, msg.Subject, msg.Content, msg.To))
	return err
}
