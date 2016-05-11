package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"

	"github.com/VividCortex/siesta"
)

func serveGetRequest(w http.ResponseWriter, r *http.Request) {
	params := &siesta.Params{}
	linkStr := params.String("link", "", "link code")
	err := params.Parse(r.Form)

	invalidLink := func() {
		w.WriteHeader(http.StatusNotFound)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Not a valid OneContactLink",
		})
		return
	}

	if err != nil || !strings.Contains(*linkStr, "-") {
		invalidLink()
		return
	}
	parts := strings.Split(*linkStr, "-")
	if len(parts) != 2 {
		invalidLink()
		return
	}

	// get request link
	requestLink, err := internalAPIClient.GetRequestLinkByCode(parts[1])
	if err != nil {
		invalidLink()
		return
	}

	// get user
	user, err := internalAPIClient.GetUser(requestLink.User)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong",
		})
		return
	}

	if user.Code != parts[0] {
		invalidLink()
		return
	}

	templ.ExecuteTemplate(w, "request", map[string]string{
		"Name": user.Name,
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
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Error": "Bad CAPTCHA",
		})
		return
	}

	// get request link
	requestLink, err := internalAPIClient.GetRequestLinkByCode(parts[1])
	if err != nil {
		invalidLink()
		return
	}

	// get user
	user, err := internalAPIClient.GetUser(requestLink.User)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Error": "Something went wrong",
		})
		return
	}

	if user.Code != parts[0] {
		invalidLink()
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
			"Name":    user.Name,
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
			"Name":    user.Name,
			"Warning": "Something went wrong. Please try again.",
		})
		return
	}
	if !recaptchaAPIResponse.Success {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Error": "Couldn't verify CAPTCHA. Please try again.",
		})
		return
	}

	toUser := user.ID
	fromUser := 0

	email, err := internalAPIClient.GetEmail(*emailStr)
	if err != nil {
		if err == client.ErrNotFound {
			// Email doesn't exist. Create a new user with the email.
			user, err := internalAPIClient.CreateUser(schema.NewUser(*nameStr, *emailStr))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				templ.ExecuteTemplate(w, "request", map[string]string{
					"Name":    user.Name,
					"Warning": "Something went wrong. Please try again.",
				})
				return
			}
			fromUser = user.ID
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			templ.ExecuteTemplate(w, "request", map[string]string{
				"Name":    user.Name,
				"Warning": "Something went wrong. Please try again.",
			})
			return
		}
	} else {
		fromUser = email.User
	}

	requestID, err := internalAPIClient.CreateRequest(fromUser, toUser)
	if err != nil {
		if err == client.ErrConflict {
			templ.ExecuteTemplate(w, "request", map[string]string{
				"Name": user.Name,
				"Info": "Looks like you already made this request. If " + user.Name +
					" has already approved your request, we'll send you their latest contact info.",
			})
			// Try to send another request email. This is idempotent.
			internalAPIClient.SendRequestEmail(requestID)
			internalAPIClient.SendContactInfoEmail(requestID)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Name":    user.Name,
			"Warning": "Something went wrong. Please try again.",
		})
		return
	}
	err = internalAPIClient.SendRequestEmail(requestID)
	if err != nil {
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Name":    user.Name,
			"Warning": "Something went wrong. Please try again.",
		})
		return
	}

	templ.ExecuteTemplate(w, "success", map[string]string{
		"Success": "Request sent!",
	})
}

func serveManageRequest(w http.ResponseWriter, r *http.Request) {
	params := &siesta.Params{}
	linkStr := params.String("link", "", "link token")
	actionStr := params.String("action", "", "action")
	err := params.Parse(r.Form)

	invalidLink := func() {
		w.WriteHeader(http.StatusNotFound)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Not a valid link",
		})
		return
	}

	if err != nil || !strings.Contains(*linkStr, "-") {
		invalidLink()
		return
	}
	parts := strings.Split(*linkStr, "-")
	if len(parts) != 2 {
		invalidLink()
		return
	}

	if *actionStr != "approve" && *actionStr != "reject" {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Invalid action.",
		})
		return
	}

	// extract request ID
	linktoken.NewLinkToken(data map[string]interface{}, expires int)

	err = internalAPIClient.ManageRequest(request.ID, *actionStr)
	if err != nil {
		if serverErr, ok := err.(client.ServerError); ok {
			if int(serverErr) == http.StatusConflict {
				templ.ExecuteTemplate(w, "invalid", map[string]string{
					"Warning": "Oops, you already responded to this request. We can't change" +
						" anything right now.",
				})
				return
			}
		}

		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		log.Println(err)
		return

	}

	err = internalAPIClient.SendContactInfoEmail(request.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		log.Println(err)
		return
	}

	templ.ExecuteTemplate(w, "success", map[string]string{
		"Success": "Approved! We'll send them an email with your contact information.",
	})
}
