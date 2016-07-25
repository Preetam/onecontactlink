package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/web/linktoken"
	"github.com/VividCortex/siesta"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mailgun/mailgun-go"
)

var (
	DSN           = "testuser@tcp(127.0.0.1:3306)/onecontactlink_test?charset=utf8"
	MailgunDomain = "samples.mailgun.org"
	MailgunKey    = "key-CHANGETHIS"
	MailgunPubKey = ""
	TokenKey      = "test key 1234567"
	DevMode       = false

	tokenCodec *linktoken.TokenCodec
)

const (
	MailgunContextKey = "mailgun"
)

func main() {
	addr := flag.String("addr", "localhost:4001", "Listen address")
	flag.StringVar(&DSN, "db-dsn", DSN, "MySQL dsn")
	flag.StringVar(&MailgunDomain, "mailgun-domain", MailgunDomain, "Mailgun domain")
	flag.StringVar(&MailgunKey, "mailgun-key", MailgunKey, "Mailgun private key")
	flag.StringVar(&MailgunPubKey, "mailgun-pubkey", MailgunPubKey, "Mailgun public key")
	flag.StringVar(&TokenKey, "token-key", TokenKey, "Token key")
	flag.BoolVar(&DevMode, "dev-mode", DevMode, "developer mode")
	flag.Parse()

	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatal(err)
	}
	mg := mailgun.NewMailgun(MailgunDomain, MailgunKey, MailgunPubKey)
	tokenCodec = linktoken.NewTokenCodec(1, TokenKey)

	service := siesta.NewService("/v1")
	service.AddPre(middleware.RequestIdentifier)
	service.AddPre(func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
		requestData.DB = db
		c.Set(MailgunContextKey, mg)
	})
	// Response generation
	service.AddPost(middleware.ResponseGenerator)
	service.AddPost(middleware.ResponseWriter)

	// Custom 404 handler
	service.SetNotFound(func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
		requestData.StatusCode = http.StatusNotFound
		requestData.ResponseError = "not found"
	})

	service.Route("GET", "/ping", "ping",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
			requestData.StatusCode = http.StatusNoContent
		})

	// Users
	service.Route("GET", "/users/:id", "gets a user", getUserByID)
	service.Route("POST", "/users/:id/activate", "activates a user", activateUser)
	service.Route("POST", "/users/:id/sendActivationEmail", "sends an activation email", sendActivationEmail)
	service.Route("GET", "/users/:id/requestLink", "gets request link", getRequestLinkByUser)
	service.Route("GET", "/users/:id/emails", "gets emails for a user", getEmailsForUser)
	service.Route("POST", "/users", "creates a user", siesta.Compose(readUser, createUser))

	// Emails
	service.Route("POST", "/emails", "creates an email", siesta.Compose(readEmail, createEmail))
	service.Route("GET", "/emails/:address", "gets an email", getEmailByAddress)
	service.Route("POST", "/emails/:address/activate", "activates an email", activateEmail)
	service.Route("POST", "/emails/:address/validate", "validates an email addres", postValidateEmailAddress)

	// Requests
	service.Route("POST", "/requests", "creates a request", siesta.Compose(readRequest, createRequest))
	service.Route("GET", "/requests/:id", "gets a request", getRequest)
	service.Route("POST", "/requests/:id/sendRequestEmail", "sends a request email to the user", sendRequestEmail)
	service.Route("POST", "/requests/:id/sendContactInfoEmail", "sends a contact info email to the requester", sendContactInfoMail)
	service.Route("POST", "/requests/:id/manage", "approves or rejects a request", manageRequest)

	// Links
	service.Route("GET", "/links/requestLinks/:requestLinkCode", "gets a request link", getRequestLinkByCode)

	// Authentication
	service.Route("POST", "/auth/send", "sends an email with a login link", sendAuthEmail)

	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
