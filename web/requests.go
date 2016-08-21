package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"
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
