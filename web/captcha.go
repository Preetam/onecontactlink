package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/Preetam/onecontactlink/middleware"
	log "github.com/Sirupsen/logrus"
	"github.com/VividCortex/siesta"
)

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
