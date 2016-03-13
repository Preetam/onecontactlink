package main

import (
	"github.com/Preetam/onecontactlink/middleware"

	"github.com/VividCortex/siesta"

	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":4003", "Listen address")
	staticDir := flag.String("static-dir", "./static", "Path to static content")
	flag.Parse()

	service := siesta.NewService("/")
	service.AddPre(middleware.RequestIdentifier)
	service.SetNotFound(http.FileServer(http.Dir(*staticDir)))
	log.Println("static directory set to", *staticDir)
	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
