package main

import (
	"github.com/Preetam/onecontactlink/middleware"

	"github.com/VividCortex/siesta"
	_ "github.com/go-sql-driver/mysql"

	"database/sql"
	"flag"
	"log"
	"net/http"
)

var (
	DSN = "testuser@tcp(127.0.0.1:3306)/mail_test?charset=utf8"
)

func main() {
	addr := flag.String("addr", ":4001", "Listen address")
	dbDSN := flag.String("db-dsn", DSN, "MySQL dsn")
	flag.Parse()

	db, err := sql.Open("mysql", *dbDSN)
	if err != nil {
		log.Fatal(err)
	}

	service := siesta.NewService("/v1")

	service.AddPre(middleware.RequestIdentifier)

	// Add access to the state via the context in every handler.
	service.AddPre(func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		c.Set(middleware.DBKey, db)
	})

	// We'll add the authenticator middleware to the "pre" chain.
	// It will ensure that every request has a valid token.
	//service.AddPre(authenticator)

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

	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
