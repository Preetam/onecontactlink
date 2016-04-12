package main

import (
	"github.com/Preetam/onecontactlink/middleware"

	"github.com/VividCortex/siesta"

	"flag"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
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
	service.DisableTrimSlash()
	service.AddPre(middleware.RequestIdentifier)

	service.Route("GET", "/", "serves index", func(w http.ResponseWriter, r *http.Request) {
		templ.ExecuteTemplate(w, "index", nil)
	})

	service.Route("GET", "/l/:link", "serves link requests", func(w http.ResponseWriter, r *http.Request) {
		params := &siesta.Params{}
		linkStr := params.String("link", "", "link code")
		err := params.Parse(r.Form)
		if err != nil || !strings.Contains(*linkStr, "-") {
			w.WriteHeader(http.StatusBadRequest)
			templ.ExecuteTemplate(w, "invalid", map[string]string{
				"Error": "Not a valid OneContactLink",
			})
			return
		}
		log.Println("requested link:", *linkStr)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Name": "John Doe",
		})
	})

	service.SetNotFound(http.FileServer(http.Dir(*staticDir)))
	log.Println("static directory set to", *staticDir)
	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
