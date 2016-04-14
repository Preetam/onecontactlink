package main

import (
	"github.com/VividCortex/siesta"

	"log"
	"net/http"
	"strings"
)

func serveGetRequest(w http.ResponseWriter, r *http.Request) {
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
}
