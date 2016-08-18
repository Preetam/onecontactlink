package middleware

import (
	"database/sql"
	"time"
)

type RequestData struct {
	RequestID     string
	StatusCode    int
	DB            *sql.DB
	ResponseData  interface{}
	ResponseError string
	Response      interface{}
	Start         time.Time
}
