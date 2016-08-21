package main

import (
	"net/http"
	"time"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/web/linktoken"
	"github.com/VividCortex/siesta"
)

func serveAuth(c siesta.Context, w http.ResponseWriter, r *http.Request) {
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

	linkToken, err := tokenCodec.DecodeToken(*linkStr, new(linktoken.UserTokenData))
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
	userID := linkToken.Data.(*linktoken.UserTokenData).User

	// get user information
	_, err = internalAPIClient.GetUser(userID)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
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
		"Success": "Logged in! Redirecting you to the app...",
	})
}

func serveDevModeAuth(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	params := &siesta.Params{}
	userID := params.Int("user", 0, "user ID")
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

	tokenExpires := time.Now().Add(48 * time.Hour)
	linkToken := linktoken.NewLinkToken(&linktoken.UserTokenData{
		User: *userID,
	}, int(tokenExpires.Unix()))

	// get user information
	_, err = internalAPIClient.GetUser(*userID)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	token, err := tokenCodec.EncodeToken(linkToken)
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
		Expires:  tokenExpires,
	})

	templ.ExecuteTemplate(w, "success", map[string]string{
		"Success": "Logged in!",
	})
}
