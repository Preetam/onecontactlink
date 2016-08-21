package main

import (
	"net/http"

	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/VividCortex/siesta"
)

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
