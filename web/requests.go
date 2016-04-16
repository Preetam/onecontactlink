package main

import (
	"encoding/json"

	"github.com/VividCortex/siesta"

	"net/http"
	"net/url"
	"strings"
)

func serveGetRequest(w http.ResponseWriter, r *http.Request) {
	params := &siesta.Params{}
	linkStr := params.String("link", "", "link code")
	err := params.Parse(r.Form)

	invalidLink := func() {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Not a valid OneContactLink",
		})
		return
	}

	if err != nil || !strings.Contains(*linkStr, "-") {
		invalidLink()
	}
	parts := strings.Split(*linkStr, "-")
	if len(parts) != 2 {
		invalidLink()
	}

	templ.ExecuteTemplate(w, "request", map[string]string{
		"Name": "John Doe",
	})
}

func servePostRequest(w http.ResponseWriter, r *http.Request) {
	params := &siesta.Params{}
	recaptchaResponse := params.String("g-recaptcha-response", "", "reCAPTCHA response")
	nameStr := params.String("name", "", "name")
	emailStr := params.String("email", "", "email")
	linkStr := params.String("link", "", "link code")
	err := params.Parse(r.Form)

	invalidLink := func() {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Not a valid OneContactLink",
		})
		return
	}

	if err != nil || !strings.Contains(*linkStr, "-") {
		invalidLink()
	}
	parts := strings.Split(*linkStr, "-")
	if len(parts) != 2 {
		invalidLink()
	}

	if *recaptchaResponse == "" {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Bad CAPTCHA",
		})
		return
	}

	if *nameStr == "" || *emailStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Error": "You must provide a name and email address",
		})
		return
	}

	// verify CAPTCHA
	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   []string{RecaptchaSecret},
		"response": []string{*recaptchaResponse},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Name":    "John Doe",
			"Warning": "Something went wrong. Please try again.",
		})
		return
	}
	recaptchaAPIResponse := struct {
		Success bool `json:"success"`
	}{}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&recaptchaAPIResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Name":    "John Doe",
			"Warning": "Something went wrong. Please try again.",
		})
		return
	}
	if recaptchaAPIResponse.Success {
		templ.ExecuteTemplate(w, "success", map[string]string{
			"Success": "Request sent!",
		})
	} else {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Bad CAPTCHA",
		})
		return
	}
}
