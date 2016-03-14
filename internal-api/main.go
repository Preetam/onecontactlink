package main

import (
	"github.com/Preetam/onecontactlink/middleware"

	"github.com/VividCortex/siesta"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mailgun/mailgun-go"

	"database/sql"
	"flag"
	"log"
	"net/http"
)

var (
	DSN           = "testuser@tcp(127.0.0.1:3306)/mail_test?charset=utf8"
	MailgunDomain = "samples.mailgun.org"
	MailgunKey    = "key-CHANGETHIS"
	MailgunPubKey = ""
)

const (
	MailgunContextKey = "mailgun"
)

func main() {
	addr := flag.String("addr", ":4001", "Listen address")
	flag.StringVar(&DSN, "db-dsn", DSN, "MySQL dsn")
	flag.StringVar(&MailgunDomain, "mailgun-domain", MailgunDomain, "Mailgun domain")
	flag.StringVar(&MailgunKey, "mailgun-key", MailgunKey, "Mailgun private key")
	flag.StringVar(&MailgunPubKey, "mailgun-pubkey", MailgunPubKey, "Mailgun public key")
	flag.Parse()

	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatal(err)
	}
	mg := mailgun.NewMailgun(MailgunDomain, MailgunKey, MailgunPubKey)

	service := siesta.NewService("/v1")
	service.AddPre(middleware.RequestIdentifier)
	service.AddPre(func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		c.Set(middleware.DBKey, db)
		c.Set(MailgunContextKey, mg)
	})
	// Response generation
	service.AddPost(middleware.ResponseGenerator)
	service.AddPost(middleware.ResponseWriter)

	// Custom 404 handler
	service.SetNotFound(func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		c.Set(middleware.StatusCodeKey, http.StatusNotFound)
		c.Set(middleware.ResponseErrorKey, "not found")
	})

	service.Route("GET", "/ping", "ping",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			c.Set(middleware.StatusCodeKey, http.StatusNoContent)
		})

	service.Route("GET", "/tokens/:token", "usage", getToken)
	service.Route("POST", "/misc/sendMail", "usage", siesta.Compose(emailMessageReader, sendEmail))

	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
