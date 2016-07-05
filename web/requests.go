package main

import (
	// std
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	// base
	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"
	// vendor
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

	if err != nil {
		invalidLink()
		return
	}

	linkToken, err := tokenCodec.DecodeToken(*linkStr, new(linktoken.RequestTokenData))
	if err != nil {
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

	// Check if token expired
	if linkToken.Expires <= int(time.Now().Unix()) {
		// Token expired.
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "This link has expired.",
		})
		return
	}

	// extract request ID
	requestID := linkToken.Data.(*linktoken.RequestTokenData).Request
	err = internalAPIClient.ManageRequest(requestID, *actionStr)
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

	if *actionStr == "approve" {
		err = internalAPIClient.SendContactInfoEmail(requestID)
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
	} else {
		templ.ExecuteTemplate(w, "success", map[string]string{
			"Success": "Rejected. That email won't be able to send you any more requests.",
		})
	}
}

func serveAuth(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	params := &siesta.Params{}
	linkStr := params.String("link", "", "link token")
	err := params.Parse(r.Form)
	invalidLink := func() {
		w.WriteHeader(http.StatusNotFound)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Not a valid link",
		})
		return
	}
	if err != nil {
		invalidLink()
		return
	}

	linkToken, err := tokenCodec.DecodeToken(*linkStr, new(linktoken.UserTokenData))
	if err != nil {
		invalidLink()
		return
	}

	// Check if token expired
	if linkToken.Expires <= int(time.Now().Unix()) {
		// Token expired.
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "This link has expired.",
		})
		return
	}

	// extract user ID
	userID := linkToken.Data.(*linktoken.UserTokenData).User

	// get user information
	_, err = internalAPIClient.GetUser(userID)
	if err != nil {
		// Token expired.
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	// Update the expiration and set a cookie
	linkToken.Expires = int(time.Now().Unix() + 86400)

	token, err := tokenCodec.EncodeToken(linkToken)
	if err != nil {
		// Token expired.
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "ocl",
		Value:    token,
		Domain:   CookieDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   !DevMode,
	})

	w.Header().Add("Refresh", "2; /app")

	templ.ExecuteTemplate(w, "success", map[string]string{
		"Success": "Logged in! Redirecting you to the app...",
	})
}

func servePostLogin(w http.ResponseWriter, r *http.Request) {
	params := &siesta.Params{}
	emailStr := params.String("email", "", "email")
	err := params.Parse(r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Error": "Invalid parameters.",
		})
		return
	}

	internalAPIClient.SendAuth(*emailStr)
	templ.ExecuteTemplate(w, "success", map[string]string{
		"Info": "We've sent a login link to '" + *emailStr +
			"' if it's associated with a valid account.",
	})
}

func serveDevModeAuth(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	params := &siesta.Params{}
	userID := params.Int("user", 0, "user ID")
	err := params.Parse(r.Form)
	invalidLink := func() {
		w.WriteHeader(http.StatusNotFound)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Not a valid link",
		})
		return
	}
	if err != nil {
		invalidLink()
		return
	}

	linkToken := linktoken.NewLinkToken(&linktoken.UserTokenData{
		User: *userID,
	}, int(time.Now().Unix()+86400))

	// get user information
	_, err = internalAPIClient.GetUser(*userID)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	token, err := tokenCodec.EncodeToken(linkToken)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "ocl",
		Value:    token,
		Domain:   CookieDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   !DevMode,
	})

	templ.ExecuteTemplate(w, "success", map[string]string{
		"Success": "Logged in!",
	})
}

func verifyCaptcha(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
	if DevMode {
		return
	}

	params := &siesta.Params{}
	recaptchaResponse := params.String("g-recaptcha-response", "", "reCAPTCHA response")
	err := params.Parse(r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Error": "Missing CAPTCHA parameter.",
		})
		q()
		return
	}

	if *recaptchaResponse == "" {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "login", map[string]string{
			"Error": "Bad CAPTCHA",
		})
		q()
		return
	}

	// verify CAPTCHA
	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   []string{RecaptchaSecret},
		"response": []string{*recaptchaResponse},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "login", map[string]string{
			"Warning": "Something went wrong. Please try again.",
		})
		q()
		return
	}
	recaptchaAPIResponse := struct {
		Success bool `json:"success"`
	}{}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&recaptchaAPIResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templ.ExecuteTemplate(w, "login", map[string]string{
			"Warning": "Something went wrong. Please try again.",
		})
		q()
		return
	}
	if !recaptchaAPIResponse.Success {
		w.WriteHeader(http.StatusBadRequest)
		templ.ExecuteTemplate(w, "login", map[string]string{
			"Error": "Couldn't verify CAPTCHA. Please try again.",
		})
		q()
		return
	}
}
