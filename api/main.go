package main

import (
	"flag"
	"log"
	"net/http"

	internalClient "github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/web/linktoken"
	"github.com/VividCortex/siesta"
)

var (
	InternalAPIEndpoint = "http://localhost:4001/v1"
	TokenKey            = "test key 1234567"

	internalAPIClient *internalClient.Client
	tokenCodec        *linktoken.TokenCodec
)

func main() {
	addr := flag.String("addr", "localhost:4002", "Listen address")
	flag.StringVar(&InternalAPIEndpoint, "internal-api", InternalAPIEndpoint,
		"Base URI to the internal API")
	flag.StringVar(&TokenKey, "token-key", TokenKey, "Token key")
	flag.Parse()

	internalAPIClient = internalClient.New(InternalAPIEndpoint)
	tokenCodec = linktoken.NewTokenCodec(1, TokenKey)

	service := siesta.NewService("/")
	service.AddPre(middleware.RequestIdentifier)
	service.AddPost(middleware.ResponseGenerator)
	service.AddPost(middleware.ResponseWriter)

	service.AddPre(func(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
		// decode token data
		cookie, err := r.Cookie("ocl")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			q()
			return
		}
		token, err := tokenCodec.DecodeToken(cookie.Value, new(linktoken.UserTokenData))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
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
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			user, err := internalAPIClient.GetUser(userID)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			requestData.ResponseData = user
		})

	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, service))
}
