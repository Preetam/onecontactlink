package main

import (
	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"

	"github.com/VividCortex/siesta"

	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":4002", "Listen address")
	flag.Parse()

	internalClient := client.New("http://localhost:4001/v1")

	service := siesta.NewService("/v1")
	service.AddPre(middleware.RequestIdentifier)
	service.AddPre(func(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
		tokenValue, _, ok := r.BasicAuth()
		if !ok {
			c.Set(middleware.StatusCodeKey, http.StatusUnauthorized)
			q()
			return
		}
		token, err := internalClient.Authenticate(tokenValue)
		if err != nil {
			log.Println(err)
			c.Set(middleware.StatusCodeKey, http.StatusUnauthorized)
			q()
			return
		}
		c.Set("token", token)
	})
	service.AddPost(middleware.ResponseGenerator)
	service.AddPost(middleware.ResponseWriter)

	service.Route("GET", "/ping", "ping",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			c.Set(middleware.StatusCodeKey, http.StatusNoContent)
		})

	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
