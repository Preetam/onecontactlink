package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"
	log "github.com/Sirupsen/logrus"
	"github.com/VividCortex/siesta"
	"github.com/mailgun/mailgun-go"
)

func sendAuthEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	emailAddress := params.String("email", "", "email address of user")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
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
			log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
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
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	err = sendMail(mg, client.EmailMessage{
		To:      mainEmail,
		From:    `"OneContactLink Notifications" <notify@out.onecontact.link>`,
		Subject: "OneContactLink Login Link",
		Content: fmt.Sprintf("Hi %s,\n\nHere's your login link. It'll be active for 15 minutes.\n\n"+
			"%s", userName, "https://www.onecontact.link/auth/"+tokenStr),
		HTMLContent: fmt.Sprintf("<p>Hi %s,</p><p>Click <a href='%s'>here</a> to log in.</p>"+
			"<p>This link will only be active for 15 minutes.</p>",
			userName, "https://www.onecontact.link/auth/"+tokenStr),
	})
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
}

func sendMail(mg mailgun.Mailgun, msg client.EmailMessage) error {
	var err error

	if !DevMode {
		textContent := fmt.Sprintf(textEmailTemplate, msg.Content)
		mailgunMessage := mailgun.NewMessage(msg.From, msg.Subject, textContent, msg.To)
		if msg.HTMLContent != "" {
			htmlContent := fmt.Sprintf(htmlEmailTemplate, msg.HTMLContent)
			mailgunMessage.SetHtml(htmlContent)
		}
		_, _, err = mg.Send(mailgunMessage)
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

func sendUserActivationEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	userID := params.Int("id", 0, "user ID")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	name := ""
	status := 0
	emailAddress := ""

	err = requestData.DB.QueryRow("SELECT users.name, users.status, emails.address"+
		" FROM users JOIN emails ON users.main_email = emails.id"+
		" WHERE users.id = ?", *userID).Scan(&name, &status, &emailAddress)
	if err != nil {
		if err == sql.ErrNoRows {
			requestData.StatusCode = http.StatusNotFound
			return
		}
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	if status == schema.UserStatusActive {
		// Request has not been approved
		requestData.StatusCode = http.StatusNotModified
		return
	}

	token, err := tokenCodec.EncodeToken(linktoken.NewLinkToken(&linktoken.ActivationTokenData{
		ActivateUser: *userID,
	}, int(time.Now().Unix()+86400)))
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	err = sendMail(mg, client.EmailMessage{
		From:    `"OneContactLink" <noreply@out.onecontact.link>`,
		To:      emailAddress,
		Subject: "Activate OneContactLink Account",
		Content: fmt.Sprintf("Hi %s,\n\n"+
			"Thanks for signing up. Click the following link to activate your account: https://www.onecontact.link/activate/%s\n\n"+
			"That link will only be valid for 1 day.", name, token),
		HTMLContent: fmt.Sprintf("<p>Hi %s,</p>"+
			"<p>Thanks for signing up. Click "+
			"<a href='https://www.onecontact.link/activate/%s'>here</a>"+
			" to activate your account.</p>"+
			"<p>That link will only be valid for 1 day.</p>", name, token),
	})
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
}
