package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"github.com/segmentio/ksuid"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// MainHandler displays the landing page
func MainHandler(w http.ResponseWriter, r *http.Request) {
	// Render the page
	t := tmpl.Lookup("mainPage")
	err := t.Execute(w, nil)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SubscribeHandler displays the page when an email address has been provided
func SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate the email address provided by the user
	emailAddr := r.PostFormValue("email")
	errs := validate.Field(emailAddr, "required,email")
	if errs != nil {
		log.Println(errs)
		http.Error(w, "Bad email address provided", http.StatusBadRequest)
		return
	}

	// Check if the email address is already in the database and verified
	var foundEmail int
	dbQuery := `
		SELECT count(email)
		FROM potential_customers
		WHERE email = $1
			AND token_verified = true`
	err := pg.QueryRow(dbQuery, emailAddr).Scan(&foundEmail)
	if err != nil {
		log.Printf("Looking for existing verified email failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if foundEmail != 0 {
		log.Printf("Potential customer '%v' just submitted their email, but it's already verified\n", emailAddr)
		http.Error(w, "That email address has already been submitted and verified", http.StatusBadRequest)
		return
	}

	// Generate new random token
	keyRaw, err := ksuid.NewRandom()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	verifyToken := keyRaw.String()
	encodedToken := base64.URLEncoding.EncodeToString([]byte(verifyToken))

	// Store the provided email address and token in the database
	dbQuery = `
		INSERT INTO potential_customers (email, token)
		VALUES ($1, $2)
		ON CONFLICT (email)
			DO UPDATE
				SET token = $2`
	commandTag, err := pg.Exec(dbQuery, emailAddr, verifyToken)
	if err != nil {
		log.Printf("Storing potential customer email failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if numRows := commandTag.RowsAffected(); numRows != 1 {
		log.Printf("Wrong number of rows (%v) affected while storing potential customer email '%v'\n", numRows, emailAddr)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the URL the user needs to click on
	var portString, verifyURL string
	if port != 443 {
		portString = fmt.Sprintf(":%v", port)
	}
	protocol := "https"
	if !httpsEnabled {
		protocol = "http"
	}
	verifyURL = fmt.Sprintf("%v://%v%v/verify?token=%v", protocol, hostName, portString, encodedToken)

	// Debugging output
	if debug {
		log.Printf(verifyURL)
	}

	// Send the verification email
	from := mail.NewEmail("Newdash.io", "interest@newdash.io")
	subject := "Please verify your email address"
	to := mail.NewEmail("", emailAddr)
	plainTextContent := fmt.Sprintf("Please visit this url to confirm your email address: %v", verifyURL)
	htmlContent := fmt.Sprintf(`Please visit this url to verify your email address: <a href="%v">%v</a>`, verifyURL, verifyURL)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(sendGridKey)
	_, err = client.Send(message)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Verification email sent to '%v'\n", emailAddr)

	// Render the page
	t := tmpl.Lookup("subscribePage")
	err = t.Execute(w, nil)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// VerifyHandler renders the main page
func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the token from the link
	v := r.FormValue("token")
	if v == "" {
		http.Error(w, "No verification token provided", http.StatusBadRequest)
		return
	}
	verifyToken, err := base64.URLEncoding.DecodeString(v)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate the token
	_, err = ksuid.Parse(string(verifyToken))
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Make sure the token is found in the database
	var foundToken int
	dbQuery := `
		SELECT count(token)
		FROM potential_customers
		WHERE token = $1`
	err = pg.QueryRow(dbQuery, verifyToken).Scan(&foundToken)
	if err != nil {
		log.Printf("Looking for existing token '%v' failed: %v\n", verifyToken, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if foundToken == 0 {
		log.Printf("A token '%v' has been submitted, but it's not present in the database\n", foundToken)
		http.Error(w, "That token value isn't known to us.  Broken email link?", http.StatusBadRequest)
		return
	}

	// Update the token status in the database
	dbQuery = `
		UPDATE potential_customers
		SET token_verified = true
		WHERE token = $1`
	commandTag, err := pg.Exec(dbQuery, verifyToken)
	if err != nil {
		log.Printf("Updating token status failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if numRows := commandTag.RowsAffected(); numRows != 1 {
		log.Printf("Wrong number of rows (%v) affected updating token '%v' status\n", numRows, verifyToken)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the page
	t := tmpl.Lookup("verifyPage")
	err = t.Execute(w, nil)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retrieve the email address corresponding to the token
	var email string
	dbQuery = `
		SELECT email
		FROM potential_customers
		WHERE token = $1`
	err = pg.QueryRow(dbQuery, verifyToken).Scan(&email)
	if err != nil {
		msg := fmt.Sprintf("Retrieving email address for token '%v' failed: %v\n", verifyToken, err)
		log.Printf(msg)
		emailAlert("Error when verifying token for Newdash interest", msg)
		return
	}
	if email == "" {
		msg := fmt.Sprintf("A token '%v' was verified, but its email address wasn't able to be retrieved\n", foundToken)
		emailAlert("Error when verifying token for Newdash interest", msg)
		return
	}

	// Send an email alerting me to the newly registered interest
	content := fmt.Sprintf("Someone has registered their interest in Newdash hosting: %v", email)
	emailAlert("New verified interest in Newdash hosting", content)
}

func emailAlert(subject, content string) {
	from := mail.NewEmail("Newdash.io", "interest@newdash.io")
	to := mail.NewEmail("", alertEmail)
	message := mail.NewSingleEmail(from, subject, to, content, content)
	client := sendgrid.NewSendClient(sendGridKey)
	_, err := client.Send(message)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(fmt.Sprintf("Alert email sent to '%v'", alertEmail))
}
