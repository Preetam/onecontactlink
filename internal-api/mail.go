package main

import (
	// std
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	// base
	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"
	"github.com/mailgun/mailgun-go"
	// vendor
	"github.com/VividCortex/siesta"
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

func sendAuthEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	emailAddress := params.String("email", "", "email address of user")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	userID := 0
	userName := ""
	mainEmail := ""
	log.Printf("[Req %s] using auth email %v", requestData.RequestID, *emailAddress)
	// Get the main email address of user with the given email address
	log.Println("SELECT users.id, users.name, e2.address FROM emails e1"+
		" JOIN users on users.id = e1.user"+
		" JOIN emails e2 ON users.main_email = e2.id"+
		" WHERE e1.address = ? AND e1.deleted = 0 AND users.deleted = 0 AND users.status = ?",
		*emailAddress, schema.UserStatusActive)
	err = requestData.DB.QueryRow("SELECT users.id, users.name, e2.address FROM emails e1"+
		" JOIN users on users.id = e1.user"+
		" JOIN emails e2 ON users.main_email = e2.id"+
		" WHERE e1.address = ? AND e1.deleted = 0 AND users.deleted = 0 AND users.status = ?",
		*emailAddress, schema.UserStatusActive).
		Scan(&userID, &userName, &mainEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			requestData.StatusCode = http.StatusNotFound
			requestData.ResponseError = "not found"
			log.Printf("[Req %s] %v", requestData.RequestID, err)
			return
		}
	}

	// Send an auth email
	linkToken := linktoken.NewLinkToken(&linktoken.UserTokenData{
		User: userID,
	}, int(time.Now().Unix()+900))
	tokenStr, err := tokenCodec.EncodeToken(linkToken)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	err = sendMail(mg, client.EmailMessage{
		To:      mainEmail,
		From:    `"OneContactLink Notifications" <notify@out.onecontact.link>`,
		Subject: "OneContactLink Login Link",
		Content: fmt.Sprintf(`Hi %s,

Here's your login link. It'll be active for 15 minutes.

%s

Cheers!
https://www.onecontact.link/
`, userName, "https://www.onecontact.link/auth/"+tokenStr),
	})
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
}

func sendMail(mg mailgun.Mailgun, msg client.EmailMessage) error {
	var err error

	if !DevMode {
		_, _, err = mg.Send(mailgun.NewMessage(msg.From, msg.Subject, msg.Content, msg.To))
	} else {
		log.Printf(`Sending mail:
From: %s
To: %s
Subject: %s
Content:
%s
`, msg.From, msg.To, msg.Subject, msg.Content)
	}

	return err
}
