package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Preetam/onecontactlink/internal-api/client"
	"github.com/Preetam/onecontactlink/middleware"
	"github.com/Preetam/onecontactlink/schema"
	"github.com/Preetam/onecontactlink/web/linktoken"
	log "github.com/Sirupsen/logrus"
	"github.com/VividCortex/mysqlerr"
	"github.com/VividCortex/siesta"
	"github.com/go-sql-driver/mysql"
	"github.com/mailgun/mailgun-go"
)

const (
	requestKey = "request"
)

func readRequest(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var request schema.Request
	err := json.NewDecoder(r.Body).Decode(&request)
	if err == nil {
		c.Set(requestKey, request)
	} else {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		q()
	}
}

func getRequest(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	var params siesta.Params
	requestID := params.Int("id", 0, "Request ID")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	request := schema.Request{
		ID: *requestID,
	}
	err = requestData.DB.QueryRow("SELECT from_user, to_user, status, created, updated"+
		" FROM requests WHERE id = ?", request.ID).
		Scan(&request.FromUser, &request.ToUser,
			&request.Status, &request.Created, &request.Updated)
	if err != nil {
		if err == sql.ErrNoRows {
			requestData.StatusCode = http.StatusNotFound
			log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
			return
		}
		requestData.StatusCode = http.StatusInternalServerError
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	requestData.ResponseData = request
}

func createRequest(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)
	request := c.Get(requestKey).(schema.Request)
	if request.FromUser == 0 || request.ToUser == 0 {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = `missing "from" or "to" user ID`
		return
	}

	now := time.Now().Unix()
	tx, err := requestData.DB.Begin()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	defer tx.Rollback()

	// Make sure both users exist

	userCount := 0
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE id = ? OR id = ?",
		request.FromUser, request.ToUser).Scan(&userCount)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	if userCount == 0 {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = "both user IDs must exist"
		return
	}

	request.Created = int(now)
	request.Updated = int(now)

	result, err := tx.Exec("INSERT INTO requests (from_user, to_user, created, updated) VALUES"+
		" (?, ?, ?, ?)", request.FromUser, request.ToUser, request.Created, request.Updated)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == mysqlerr.ER_DUP_ENTRY {
				// already exists
				requestData.StatusCode = http.StatusConflict
				log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)

				// send the existing request
				err := tx.QueryRow("SELECT id, status, created, updated FROM"+
					" requests WHERE from_user = ? AND to_user = ?",
					request.FromUser, request.ToUser).
					Scan(&request.ID, &request.Status,
						&request.Created, &request.Updated)
				if err != nil {
					// Some other error
					requestData.StatusCode = http.StatusInternalServerError
					log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
					return
				}
				requestData.ResponseData = request
				return
			}
		}
		// Some other error
		requestData.StatusCode = http.StatusInternalServerError
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	// We created a new request.
	lastID, err := result.LastInsertId()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	request.ID = int(lastID)
	requestData.ResponseData = request
}

func sendRequestEmail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	requestID := params.Int("id", 0, "Request ID")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	status := 0
	// Check if an email has already been sent
	err = requestData.DB.QueryRow("SELECT status FROM requests WHERE id = ?", *requestID).Scan(&status)
	if err != nil {
		requestData.StatusCode = http.StatusNotModified
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	if status >= schema.RequestStatusSent {
		// Email has already been sent. Nothing else to do.
		requestData.StatusCode = http.StatusNotModified
		return
	}

	receiverName := ""
	receiverEmail := ""
	receiverCode := ""
	senderName := ""
	senderEmail := ""

	err = requestData.DB.QueryRow("SELECT"+
		" u1.name AS from_name,"+
		" e1.address as from_address,"+
		" u2.name as to_name,"+
		" e2.address as to_address,"+
		" u2.code as to_code"+
		" FROM requests"+
		" JOIN users u1 ON u1.id = from_user"+
		" JOIN users u2 ON u2.id = to_user"+
		" JOIN emails e1 ON u1.main_email = e1.id"+
		" JOIN emails e2 ON u2.main_email = e2.id"+
		" WHERE requests.id = ?", *requestID).
		Scan(&senderName, &senderEmail, &receiverName, &receiverEmail, &receiverCode)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	linkToken := linktoken.NewLinkToken(&linktoken.RequestTokenData{
		Request: *requestID,
	}, int(time.Now().Unix()+86400))
	tokenStr, err := tokenCodec.EncodeToken(linkToken)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	manageLink := fmt.Sprintf("https://www.onecontact.link/m/%s", tokenStr)
	approveLink := manageLink + "?action=approve"
	rejectLink := manageLink + "?action=reject"
	messageContent := fmt.Sprintf(`Hi %s,

%s (%s) has requested your contact information using OneContact.Link.

Click on one of the following links to approve or reject this request.
We'll send them this email address if you approve.

Approve:
%s

Reject:
%s

These links expire in 1 day.

Cheers!
https://www.onecontact.link/
`, receiverName, senderName, senderEmail, approveLink, rejectLink)
	htmlMessageContent := fmt.Sprintf(`<p>Hi %s,</p>

<p>%s (%s) has requested your contact information using OneContact.Link.</p>

<p>Click on one of the following links to approve or reject this request.
We'll send them this email address if you approve.</p>

<p><a href='%s'>Approve</a></p>

<p><a href='%'>Reject</a></p>

<p>These links expire in 1 day.</p>
`, receiverName, senderName, senderEmail, approveLink, rejectLink)
	msg := client.EmailMessage{
		From:        `"OneContactLink Notifications" <notify@onecontact.link>`,
		To:          receiverEmail,
		Subject:     "OneContactLink request",
		Content:     messageContent,
		HTMLContent: htmlMessageContent,
	}
	err = sendMail(mg, msg)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = "couldn't send email"
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	_, err = requestData.DB.Exec("UPDATE requests SET status = ? WHERE id = ?",
		schema.RequestStatusSent, *requestID)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = "couldn't update request status"
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
}

func manageRequest(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	requestID := params.Int("id", 0, "Request ID")
	action := params.String("action", "", "Manage action")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	switch *action {
	case "approve", "reject":
		// OK
	default:
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = "action must be 'approve' or 'reject'"
		return
	}

	tx, err := requestData.DB.Begin()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
	defer tx.Rollback()

	status := 0
	// Check if the status has already been set
	err = tx.QueryRow("SELECT status FROM requests WHERE id = ?", *requestID).Scan(&status)
	if err != nil {
		requestData.StatusCode = http.StatusNotModified
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	if status > schema.RequestStatusSent {
		// Status has already been set
		requestData.StatusCode = http.StatusConflict
		return
	}

	switch *action {
	case "approve":
		status = schema.RequestStatusApproved
	case "reject":
		status = schema.RequestStatusRejected
	}

	_, err = tx.Exec("UPDATE requests SET status = ? WHERE id = ?", status, *requestID)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
}

func sendContactInfoMail(c siesta.Context, w http.ResponseWriter, r *http.Request) {
	mg := c.Get(MailgunContextKey).(mailgun.Mailgun)
	requestData := c.Get(middleware.RequestDataKey).(*middleware.RequestData)

	var params siesta.Params
	requestID := params.Int("id", 0, "Request ID")
	err := params.Parse(r.Form)
	if err != nil {
		requestData.StatusCode = http.StatusBadRequest
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	status := 0
	emailSentTs := 0
	err = requestData.DB.QueryRow("SELECT status, email_sent FROM requests WHERE id = ?", *requestID).
		Scan(&status, &emailSentTs)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	if status != schema.RequestStatusApproved {
		// Request has not been approved
		requestData.StatusCode = http.StatusForbidden
		return
	}

	if time.Now().Unix()-int64(emailSentTs) < 86400 {
		// Less than a day since the last contact info email was sent
		requestData.StatusCode = http.StatusBadRequest
		return
	}

	receiverName := ""
	receiverEmail := ""
	requestedName := ""
	requestedEmail := ""
	err = requestData.DB.QueryRow("SELECT u1.name AS from_name,"+
		" e1.address as from_address,"+
		" u2.name as to_name,"+
		" e2.address as to_address"+
		" FROM requests"+
		" JOIN users u1 ON u1.id = from_user"+
		" JOIN users u2 ON u2.id = to_user"+
		" JOIN emails e1 ON u1.main_email = e1.id"+
		" JOIN emails e2 ON u2.main_email = e2.id"+
		" WHERE requests.id = ?", *requestID).
		Scan(&receiverName, &receiverEmail, &requestedName, &requestedEmail)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	messageContent := fmt.Sprintf(`Hi %s,

%s has approved your contact request! You can reach them at %s.

Cheers!
https://www.onecontact.link/
`, receiverName, requestedName, requestedEmail)
	htmlMessageContent := fmt.Sprintf(`<p>Hi %s,</p>

<p>%s has approved your contact request! You can reach them at <a href='mailto:%s'>%s</a>.</p>
`, receiverName, requestedName, requestedEmail, requestedEmail)
	err = sendMail(mg, client.EmailMessage{
		From:        `"OneContactLink Notifications" <notify@onecontact.link>`,
		To:          receiverEmail,
		Subject:     requestedName + "'s contact information via OneContactLink",
		Content:     messageContent,
		HTMLContent: htmlMessageContent,
	})
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}

	_, err = requestData.DB.Exec("UPDATE requests SET email_sent = UNIX_TIMESTAMP() WHERE id = ?",
		*requestID)
	if err != nil {
		requestData.StatusCode = http.StatusInternalServerError
		requestData.ResponseError = err.Error()
		log.WithFields(log.Fields{"request_id": requestData.RequestID,"method": r.Method,"url": r.URL, "error": err.Error()}).Warnf("[Req %s] %v", requestData.RequestID, err)
		return
	}
}
