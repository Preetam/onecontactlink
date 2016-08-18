package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"
	log "github.com/Sirupsen/logrus"
	"github.com/VividCortex/mysqlerr"
	"github.com/VividCortex/siesta"
	"github.com/go-sql-driver/mysql"
	"github.com/mailgun/mailgun-go"
)

const (
	emailKey = "request"
)

func readEmail(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var email schema.Email
	err := json.NewDecoder(r.Body).Decode(&email)
	if err == nil {
		c.Set(emailKey, email)
	} else {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		q()
	}
}

func createEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	email := c.Get(emailKey).(schema.Email)
	if email.User == 0 || email.Address == "" {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = `missing user ID or email address`
		return
	}

	now := time.Now().Unix()

	result, err := requestData.DB.Exec("INSERT INTO emails (address, user, status, created, updated, deleted)"+
		" VALUES (?, ?, ?, ?, ?, 0)",
		email.Address, email.User, schema.EmailStatusDefault, now, now)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == mysqlerr.ER_DUP_ENTRY {
				// already exists
				requestData.StatusCode = http.StatusConflict
				log.Printf("[Req %s] %v", requestData.RequestID, err)
				return
			}
		}
		// Some other error
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	email.ID = int(lastID)
	requestData.ResponseData = email
}

func getEmailByAddress(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	var params siesta.Params
	address := params.String("address", "", "Email address")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	email := schema.Email{
		Address: *address,
	}
	err = requestData.DB.QueryRow("SELECT id, user, created, updated FROM emails"+
		" WHERE address = ? AND deleted = 0",
		email.Address).Scan(&email.ID, &email.User, &email.Created, &email.Updated)
	if err != nil {
		if err == sql.ErrNoRows {
			requestData.StatusCode = http.StatusNotFound
			log.Printf("[Req %s] %v", requestData.RequestID, err)
			return
		}
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	requestData.ResponseData = email
}

func activateEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	var params siesta.Params
	address := params.String("address", "", "Email address")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	_, err = requestData.DB.Exec("UPDATE emails SET status = ? WHERE address = ? AND deleted = 0",
		schema.EmailStatusActive, *address)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.Printf("[Req %s] %v", requestData.RequestID, err)
	}
}

func postValidateEmailAddress(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	var params siesta.Params
	address := params.String("address", "", "Email address")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	validation, err := mg.ValidateEmail(*address)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	requestData.ResponseData = validation.IsValid
}

func sendEmailActivationEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	address := params.String("address", "", "Email address")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	name := ""
	status := 0
	err = requestData.DB.QueryRow("SELECT emails.status, users.name FROM emails"+
		" JOIN users ON emails.user = users.id WHERE emails.address = ? AND emails.deleted = 0",
		*address).Scan(&status, &name)
	if err != nil {
		if err == sql.ErrNoRows {
			requestData.StatusCode = http.StatusNotFound
			return
		}
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	if status != schema.EmailStatusDefault {
		requestData.StatusCode = http.StatusNotModified
		return
	}

	token, err := tokenCodec.EncodeToken(linktoken.NewLinkToken(&linktoken.EmailActivationTokenData{
		ActivateEmail: *address,
	}, int(time.Now().Unix()+86400)))
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	err = sendMail(mg, client.EmailMessage{
		From:    `"OneContactLink" <noreply@out.onecontact.link>`,
		To:      *address,
		Subject: "Activate Email Address",
		Content: fmt.Sprintf("Hi %s,\n\n"+
			"Click the following link to activate your new email address: https://www.onecontact.link/activate-email/%s\n\n"+
			"That link will only be valid for 1 day.", name, token),
		HTMLContent: fmt.Sprintf("<p>Hi %s,</p>"+
			"<p>Click "+
			"<a href='https://www.onecontact.link/activate-email/%s'>here</a>"+
			" to activate your new email address.</p>"+
			"<p>That link will only be valid for 1 day.</p>", name, token),
	})
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.Printf("[Req %s] %v", requestData.RequestID, err)
		return
	}
}
