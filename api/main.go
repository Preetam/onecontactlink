package main

import (
	"encoding/json"
	"flag"
	"net/http"

	internalClient "github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"
	log "github.com/Sirupsen/logrus"
	"github.com/VividCortex/siesta"
)

var (
	InternalAPIEndpoint = "http://localhost:4001/v1"
	TokenKey            = "test key 1234567"
	DevMode             = false

	internalAPIClient *internalClient.Client
	tokenCodec        *linktoken.TokenCodec
)

func main() {
	addr := flag.String("addr", "localhost:4002", "Listen address")
	flag.StringVar(&InternalAPIEndpoint, "internal-api", InternalAPIEndpoint,
		"Base URI to the internal API")
	flag.StringVar(&TokenKey, "token-key", TokenKey, "Token key")
	flag.BoolVar(&DevMode, "dev-mode", DevMode, "Developer mode")
	flag.Parse()

	if !DevMode {
		log.SetFormatter(&log.JSONFormatter{})
	}

	internalAPIClient = internalClient.New(InternalAPIEndpoint)
	tokenCodec = linktoken.NewTokenCodec(1, TokenKey)

	service := siesta.NewService("/")
	service.AddPre(middleware.RequestIdentifier)
	service.AddPost(middleware.ResponseGenerator)
	service.AddPost(middleware.ResponseWriter)

	service.AddPre(func(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
		requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

		// decode token data
		cookie, err := r.Cookie("ocl")
		if err != nil {
			requestData.StatusCode = http.StatusUnauthorized
			q()
			return
		}
		token, err := tokenCodec.DecodeToken(cookie.Value, new(linktoken.UserTokenData))
		if err != nil {
			requestData.StatusCode = http.StatusUnauthorized
			q()
			return
		}
		userID := token.Data.(*linktoken.UserTokenData).User
		c.Set("user", userID)
		log.Println("user:", c.Get("user"))
	})

	service.Route("GET", "/ping", "ping",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
			requestData.ResponseData = "pong"
		})

	service.Route("GET", "/user", "user",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
			userID, ok := c.Get("user").(int)
			if !ok {
				requestData.StatusCode = http.StatusBadRequest
				return
			}
			user, err := internalAPIClient.GetUser(userID)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusInternalServerError
				return
			}
			requestData.ResponseData = user
		})

	service.Route("GET", "/emails", "get emails",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
			userID, ok := c.Get("user").(int)
			if !ok {
				requestData.StatusCode = http.StatusBadRequest
				return
			}
			emails, err := internalAPIClient.GetUserEmails(userID)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusInternalServerError
				return
			}
			requestData.ResponseData = emails
		})

	service.Route("POST", "/emails", "create an email",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
			userID, ok := c.Get("user").(int)
			if !ok {
				requestData.StatusCode = http.StatusBadRequest
				return
			}

			var email schema.Email
			err := json.NewDecoder(r.Body).Decode(&email)
			if err != nil {
				requestData.StatusCode = http.StatusBadRequest
				requestData.ResponseError = err.Error()
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				return
			}

			createdEmail, err := internalAPIClient.CreateEmail(userID, email.Address)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusInternalServerError
				return
			}

			requestData.ResponseData = createdEmail
		})

	service.Route("POST", "/emails/:address/send_activation", "send activation email",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

			var params siesta.Params
			address := params.String("address", "", "Email address")
			err := params.Parse(r.Form)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusBadRequest
				return
			}

			err = internalAPIClient.SendEmailActivationEmail(*address)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusInternalServerError
				return
			}
		})

	service.Route("DELETE", "/emails/:address", "delete email address",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

			var params siesta.Params
			address := params.String("address", "", "Email address")
			err := params.Parse(r.Form)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusBadRequest
				return
			}

			err = internalAPIClient.DeleteEmail(*address)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusInternalServerError
				return
			}
		})

	service.Route("GET", "/contact_link", "user",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

			userID, ok := c.Get("user").(int)
			if !ok {
				requestData.StatusCode = http.StatusBadRequest
				return
			}
			user, err := internalAPIClient.GetUser(userID)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusInternalServerError
				return
			}
			requestLink, err := internalAPIClient.GetRequestLinkByUser(userID)
			if err != nil {
				log.WithFields(log.Fields{"request_id": requestData.RequestID,
					"method": r.Method,
					"url":    r.URL,
					"error":  err.Error()}).
					Warnf("[Req %s] %v", requestData.RequestID, err)
				requestData.StatusCode = http.StatusInternalServerError
				return
			}
			requestData.ResponseData = "http://www.onecontact.link/r/" +
				user.Code + "-" +
				requestLink.Code
		})

	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
