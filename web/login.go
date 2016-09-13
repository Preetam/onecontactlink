package main

import (
	"net/http"

	"github.com/Preetam/onecontactlink/middleware"
	"github.com/VividCortex/siesta"
)

func servePostLogin(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	captchaResult := extractCaptchaResult(c)
	switch captchaResult {
	case captchaResultOK:
		// nothing to do
	case captchaResultFail:
		requestData.StatusCode = http.StatusBadRequest
		templ.ExecuteTemplate(w, "invalid", map[string]string{
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
