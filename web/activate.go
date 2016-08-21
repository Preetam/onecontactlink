package main

import (
	"net/http"
	"time"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/web/linktoken"
	"github.com/VividCortex/siesta"
)

func serveActivate(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	params := &siesta.Params{}
	linkStr := params.String("link", "", "link token")
	err := params.Parse(r.Form)
	invalidLink := func() {
		requestData.StatusCode = http.StatusNotFound
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Not a valid link",
		})
		return
	}
	if err != nil {
		invalidLink()
		return
	}

	linkToken, err := tokenCodec.DecodeToken(*linkStr, new(linktoken.ActivationTokenData))
	if err != nil {
		invalidLink()
		return
	}

	// Check if token expired
	if linkToken.Expires <= int(time.Now().Unix()) {
		// Token expired.
		requestData.StatusCode = http.StatusBadRequest
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "This link has expired.",
		})
		return
	}

	// extract user ID
	userID := linkToken.Data.(*linktoken.ActivationTokenData).ActivateUser

	// activate user
	err = internalAPIClient.ActivateUser(userID)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	// get user information
	_, err = internalAPIClient.GetUser(userID)
	if err != nil {
		// Token expired.
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	userToken := linktoken.NewLinkToken(&linktoken.UserTokenData{
		User: userID,
	}, int(time.Now().Unix()+86400))

	// Set a cookie
	token, err := tokenCodec.EncodeToken(userToken)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
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
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Add("Refresh", "2; /app")

	templ.ExecuteTemplate(w, "success", map[string]string{
		"Success": "Activated! Redirecting you to the app...",
	})
}

func serveActivateEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	params := &siesta.Params{}
	linkStr := params.String("link", "", "link token")
	err := params.Parse(r.Form)
	invalidLink := func() {
		requestData.StatusCode = http.StatusNotFound
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Not a valid link",
		})
		return
	}
	if err != nil {
		invalidLink()
		return
	}

	linkToken, err := tokenCodec.DecodeToken(*linkStr, new(linktoken.EmailActivationTokenData))
	if err != nil {
		invalidLink()
		return
	}

	// Check if token expired
	if linkToken.Expires <= int(time.Now().Unix()) {
		// Token expired.
		requestData.StatusCode = http.StatusBadRequest
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "This link has expired.",
		})
		return
	}

	// extract email address
	email := linkToken.Data.(*linktoken.EmailActivationTokenData).ActivateEmail

	err = internalAPIClient.ActivateEmail(email)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	w.Header().Add("Refresh", "2; /app")

	templ.ExecuteTemplate(w, "success", map[string]string{
		"Success": "Activated! Redirecting you to the app...",
	})
}
