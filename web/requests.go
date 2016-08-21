package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"
	log "github.com/Sirupsen/logrus"
	"github.com/VividCortex/siesta"
)

var (
	captchaResultOK    = 0
	captchaResultFail  = 1
	captchaResultError = 2
)

func serveGetRequest(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	params := &siesta.Params{}
	linkStr := params.String("link", "", "link code")
	err := params.Parse(r.Form)

	invalidLink := func() {
		requestData.StatusCode = http.StatusNotFound
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
		requestData.StatusCode = http.StatusInternalServerError
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

func servePostRequest(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	captchaResult := extractCaptchaResult(c)
	switch captchaResult {
	case captchaResultOK:
		// nothing to do
	case captchaResultFail:
		requestData.StatusCode = http.StatusBadRequest
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Error": "Invalid CAPTCHA.",
		})
		return
	case captchaResultError:
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "request", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}
	params := &siesta.Params{}
	nameStr := params.String("name", "", "name")
	emailStr := params.String("email", "", "email")
	linkStr := params.String("link", "", "link code")
	err := params.Parse(r.Form)

	invalidLink := func() {
		requestData.StatusCode = http.StatusBadRequest
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
		requestData.StatusCode = http.StatusInternalServerError
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
		requestData.StatusCode = http.StatusBadRequest
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
				requestData.StatusCode = http.StatusInternalServerError
				templ.ExecuteTemplate(w, "request", map[string]string{
					"Name":    user.Name,
					"Warning": "Something went wrong. Please try again.",
				})
				return
			}
			fromUser = user.ID
		} else {
			requestData.StatusCode = http.StatusInternalServerError
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
		requestData.StatusCode = http.StatusInternalServerError
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

func serveManageRequest(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	params := &siesta.Params{}
	linkStr := params.String("link", "", "link token")
	actionStr := params.String("action", "", "action")
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

	linkToken, err := tokenCodec.DecodeToken(*linkStr, new(linktoken.RequestTokenData))
	if err != nil {
		invalidLink()
		return
	}

	if *actionStr != "approve" && *actionStr != "reject" {
		requestData.StatusCode = http.StatusBadRequest
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Invalid action.",
		})
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

		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return

	}

	if *actionStr == "approve" {
		err = internalAPIClient.SendContactInfoEmail(requestID)
		if err != nil {
			requestData.StatusCode = http.StatusInternalServerError
			templ.ExecuteTemplate(w, "invalid", map[string]string{
				"Error": "Something went wrong. Please try again.",
			})
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

func servePostLogin(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	captchaResult := extractCaptchaResult(c)
	switch captchaResult {
	case captchaResultOK:
		// nothing to do
	case captchaResultFail:
		requestData.StatusCode = http.StatusBadRequest
		templ.ExecuteTemplate(w, "in", map[string]string{
			"Error": "Invalid CAPTCHA.",
		})
		return
	case captchaResultError:
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "login", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}
	params := &siesta.Params{}
	emailStr := params.String("email", "", "email")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		templ.ExecuteTemplate(w, "login", map[string]string{
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

func servePostSignup(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	captchaResult := extractCaptchaResult(c)
	switch captchaResult {
	case captchaResultOK:
		// nothing to do
	case captchaResultFail:
		requestData.StatusCode = http.StatusBadRequest
		templ.ExecuteTemplate(w, "signup", map[string]string{
			"Error": "Invalid CAPTCHA.",
		})
		return
	case captchaResultError:
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "signup", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}
	params := &siesta.Params{}
	nameStr := params.String("name", "", "name")
	emailStr := params.String("email", "", "email")
	err := params.Parse(r.Form)
	if err != nil || *nameStr == "" || *emailStr == "" {
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "signup", map[string]string{
			"Error": "Invalid name or email.",
		})
		return
	}

	// Check if email exists.
	email, err := internalAPIClient.GetEmail(*emailStr)
	if err != nil && err != client.ErrNotFound {
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "signup", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	userToReceiveActivationEmail := 0
	if err == client.ErrNotFound {
		// Email not found. Create a user with that email address.
		user, err := internalAPIClient.CreateUser(schema.NewUser(*nameStr, *emailStr))
		if err != nil {
			requestData.StatusCode = http.StatusInternalServerError
			templ.ExecuteTemplate(w, "signup", map[string]string{
				"Error": "Something went wrong. Please try again.",
			})
			return
		}
		userToReceiveActivationEmail = user.ID
	} else {
		// Email already exists. Check if it's associated with an active user.
		user, err := internalAPIClient.GetUser(email.User)
		if err != nil {
			requestData.StatusCode = http.StatusInternalServerError
			templ.ExecuteTemplate(w, "signup", map[string]string{
				"Error": "Something went wrong. Please try again.",
			})
			return
		}

		switch user.Status {
		case schema.UserStatusDefault:
			// User hasn't been activated.
			userToReceiveActivationEmail = user.ID
		case schema.UserStatusActive:
			requestData.StatusCode = http.StatusConflict
			templ.ExecuteTemplate(w, "invalid", map[string]string{
				"Error": "This email address is already associated with an active user.",
			})
			return
		}
	}

	// Send activation email.
	err = internalAPIClient.SendActivationEmail(userToReceiveActivationEmail)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		templ.ExecuteTemplate(w, "invalid", map[string]string{
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	requestData.StatusCode = http.StatusInternalServerError
	templ.ExecuteTemplate(w, "success", map[string]string{
		"Success": "We've sent you an email to activate your account.",
	})
}

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

func verifyCaptcha(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	if DevMode {
		c.Set(middleware.CaptchaResult, captchaResultOK)
		return
	}

	params := &siesta.Params{}
	recaptchaResponse := params.String("g-recaptcha-response", "", "reCAPTCHA response")
	err := params.Parse(r.Form)
	if err != nil {
		log.WithFields(log.Fields{"request_id": requestData.RequestID,
			"method": r.Method,
			"url":    r.URL,
			"error":  err.Error()}).
			Warnf("[Req %s] %v", requestData.RequestID, err)
		c.Set(middleware.CaptchaResult, captchaResultFail)
		return
	}

	if *recaptchaResponse == "" {
		c.Set(middleware.CaptchaResult, captchaResultFail)
		return
	}

	// verify CAPTCHA
	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   []string{RecaptchaSecret},
		"response": []string{*recaptchaResponse},
	})
	if err != nil {
		log.WithFields(log.Fields{"request_id": requestData.RequestID,
			"method": r.Method,
			"url":    r.URL,
			"error":  err.Error()}).
			Warnf("[Req %s] %v", requestData.RequestID, err)
		c.Set(middleware.CaptchaResult, captchaResultError)
		return
	}
	recaptchaAPIResponse := struct {
		Success bool `json:"success"`
	}{}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&recaptchaAPIResponse)
	if err != nil {
		log.WithFields(log.Fields{"request_id": requestData.RequestID,
			"method": r.Method,
			"url":    r.URL,
			"error":  err.Error()}).
			Warnf("[Req %s] %v", requestData.RequestID, err)
		c.Set(middleware.CaptchaResult, captchaResultError)
		return
	}
	if !recaptchaAPIResponse.Success {
		c.Set(middleware.CaptchaResult, captchaResultFail)
		return
	}
	c.Set(middleware.CaptchaResult, captchaResultOK)
}

func extractCaptchaResult(c siesta.Context) int {
	result, ok := c.Get(middleware.CaptchaResult).(int)
	if !ok {
		return captchaResultError
	}
	return result
}
