package main

import (
	"github.com/Preetam/onecontactlink/middleware"

	"github.com/VividCortex/siesta"

	"flag"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

var (
	templ *template.Template
)

func main() {
	addr := flag.String("addr", ":4003", "Listen address")
	staticDir := flag.String("static-dir", "./static", "Path to static content")
	templatesDir := flag.String("templates-dir", "./templates", "Path to templates")
	flag.Parse()

	var err error
	templ, err = template.ParseGlob(filepath.Join(*templatesDir, "*"))
	if err != nil {
		log.Fatal(err)
	}

	service := siesta.NewService("/")
	service.AddPre(middleware.RequestIdentifier)

	service.Route("GET", "/", "serves index", func(w http.ResponseWriter, r *http.Request) {
		templ.ExecuteTemplate(w, "index", nil)
	})

	service.SetNotFound(http.FileServer(http.Dir(*staticDir)))
	log.Println("static directory set to", *staticDir)
	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
