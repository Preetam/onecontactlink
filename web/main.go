package main

import (
	internalClient "github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"

	"github.com/VividCortex/siesta"

	"flag"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

var (
	// RecaptchaSecret is the API secret used to verify reCHAPTCHA responses
	RecaptchaSecret = ""

	templ             *template.Template
	internalAPIClient *internalClient.Client
)

func main() {
	addr := flag.String("addr", "localhost:4003", "Listen address")
	staticDir := flag.String("static-dir", "./static", "Path to static content")
	templatesDir := flag.String("templates-dir", "./templates", "Path to templates")
	internalAPIURL := flag.String("internal-api", "http://localhost:4001/v1", "internal API URL")
	flag.StringVar(&RecaptchaSecret, "recaptcha-secret", "", "reCHAPTCHA API secret")
	flag.Parse()

	var err error
	templ, err = template.ParseGlob(filepath.Join(*templatesDir, "*"))
	if err != nil {
		log.Fatal(err)
	}

	internalAPIClient = internalClient.New(*internalAPIURL)

	service := siesta.NewService("/")
	service.DisableTrimSlash() // required for static file handler
	service.AddPre(middleware.RequestIdentifier)

	service.Route("GET", "/", "serves index", func(w http.ResponseWriter, r *http.Request) {
		templ.ExecuteTemplate(w, "index", nil)
	})

	// contact link pages
	service.Route("GET", "/r/:link", "handles contact request page", serveGetRequest)
	service.Route("POST", "/r/:link", "handles contact request submission", servePostRequest)

	// manage link
	service.Route("GET", "/m/:link", "handles request management page", serveManageRequest)

	service.SetNotFound(http.FileServer(http.Dir(*staticDir)))
	log.Println("static directory set to", *staticDir)
	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
