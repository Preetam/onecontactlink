package main

import (
	internalClient "github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/web/linktoken"

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
	TokenKey        = ""
	CookieDomain    = ".localhost"
	DevMode         = false

	templ             *template.Template
	internalAPIClient *internalClient.Client
	tokenCodec        *linktoken.TokenCodec
)

func main() {
	addr := flag.String("addr", "localhost:4003", "Listen address")
	staticDir := flag.String("static-dir", "./static", "Path to static content")
	templatesDir := flag.String("templates-dir", "./templates", "Path to templates")
	internalAPIURL := flag.String("internal-api", "http://localhost:4001/v1", "internal API URL")
	flag.StringVar(&RecaptchaSecret, "recaptcha-secret", "", "reCHAPTCHA API secret")
	flag.StringVar(&TokenKey, "token-key", TokenKey, "Token key")
	flag.StringVar(&CookieDomain, "cookie-domain", CookieDomain, "Cookie domain")
	flag.BoolVar(&DevMode, "dev-mode", DevMode, "Developer mode")

	flag.Parse()

	var err error
	templ, err = template.ParseGlob(filepath.Join(*templatesDir, "*"))
	if err != nil {
		log.Fatal(err)
	}

	internalAPIClient = internalClient.New(*internalAPIURL)
	tokenCodec = linktoken.NewTokenCodec(1, TokenKey)

	service := siesta.NewService("/")
	service.DisableTrimSlash() // required for static file handler
	service.AddPre(middleware.RequestIdentifier)

	service.Route("GET", "/", "serves index", func(w http.ResponseWriter, r *http.Request) {
		templ.ExecuteTemplate(w, "index", nil)
	})

	service.Route("GET", "/login", "serves login page", func(w http.ResponseWriter, r *http.Request) {
		templ.ExecuteTemplate(w, "login", nil)
	})
	service.Route("POST", "/login", "serves login form submit", servePostLogin)

	// contact link pages
	service.Route("GET", "/r/:link", "handles contact request page", serveGetRequest)
	service.Route("POST", "/r/:link", "handles contact request submission", servePostRequest)

	// manage link
	service.Route("GET", "/m/:link", "handles request management page", serveManageRequest)

	// auth link
	service.Route("GET", "/auth/:link", "handles authentication links", serveAuth)

	service.SetNotFound(http.FileServer(http.Dir(*staticDir)))
	log.Println("static directory set to", *staticDir)
	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
