package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/segmentio/ksuid"
	"github.com/smtp2go-oss/smtp2go-go"
)

// MainHandler displays the landing page
func MainHandler(w http.ResponseWriter, r *http.Request) {
	pageData := struct {
		TurnstileEnabled bool
		TurnstileSiteKey string
	}{
		TurnstileEnabled,
		TurnstileSiteKey,
	}

	// Render the page
	t := tmpl.Lookup("mainPage")
	err := t.Execute(w, pageData)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SubscribeHandler displays the page when an email address has been provided
func SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	// Was a valid Cloudflare Turnstile response provided
	turnstileTokenSuccess := false
	if TurnstileEnabled {
		turnstileToken := r.PostFormValue("cf-turnstile-response")
		if turnstileToken == "" {
			http.Error(w, "Missing Cloudflare Turnstile token", http.StatusForbidden)
			return
		}

		if debug {
			log.Printf("Cloudflare Token provided: %v", turnstileToken)
		}

		// JSON encode the Turnstile token, for checking by the backend
		x := struct {
			TurnstileToken     string `json:"response"`
			TurnstileSecretKey string `json:"secret"`
		}{
			turnstileToken,
			TurnstileSecretKey,
		}
		jsonBody, err := json.MarshalIndent(x, "", " ")
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Validate the provided Cloudflare Turnstile token using the siteverity end point
		resp, err := http.Post("https://challenges.cloudflare.com/turnstile/v0/siteverify", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Turnstile response structure as per: https://developers.cloudflare.com/turnstile/get-started/server-side-validation/
		type TurnstileResponse struct {
			Success            bool      `json:"success"`
			ChallengeTimestamp time.Time `json:"challenge_ts"`
			Hostname           string    `json:"hostname"`
			ErrorCodes         []string  `json:"error-codes"`
			Action             string    `json:"action"`
			CustomerData       string    `json:"cdata"`
		}

		// Convert the JSON into something usable
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var tsResp TurnstileResponse
		if err = json.Unmarshal(raw, &tsResp); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if debug {
			log.Println("**************************")
			log.Println("Turnstile response failure")
			log.Println("**************************")
			log.Printf("Turnstile response status code: %s", resp.Status)
			log.Printf("Turnstile response Success : '%v'", tsResp.Success)
			log.Printf("Turnstile response Challenge Timestamp : '%v'", tsResp.ChallengeTimestamp)
			log.Printf("Turnstile response Hostname : '%v'", tsResp.Hostname)
			log.Printf("Turnstile response Errorcode : '%v'", tsResp.ErrorCodes)
			log.Printf("Turnstile response Action : '%v'", tsResp.Action)
			log.Printf("Turnstile response CustomerData : '%v'", tsResp.CustomerData)
			log.Println("**************************")
			log.Println("")
		}

		// Mark the submission as valid if Turnstile succeeded
		if tsResp.Success {
			turnstileTokenSuccess = true
		} else {
			// Turnstile validation failed
			http.Error(w, "Cloudflare Turnstile doesn't think you're a real user.  Please go back and try again.", http.StatusForbidden)
			return
		}
	}

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
	err := pg.QueryRow(context.Background(), dbQuery, emailAddr).Scan(&foundEmail)
	if err != nil {
		log.Printf("Looking for existing verified email failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if foundEmail != 0 {
		log.Printf("Potential customer '%v' just submitted their email, but it's already verified", emailAddr)
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
		INSERT INTO potential_customers (email, token, passed_turnstile_check)
		VALUES ($1, $2, $3)
		ON CONFLICT (email)
			DO UPDATE
				SET token = $2, passed_turnstile_check = $3`
	commandTag, err := pg.Exec(context.Background(), dbQuery, emailAddr, verifyToken, turnstileTokenSuccess)
	if err != nil {
		log.Printf("Storing potential customer email failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if numRows := commandTag.RowsAffected(); numRows != 1 {
		log.Printf("Wrong number of rows (%v) affected while storing potential customer email '%v'", numRows, emailAddr)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the URL the user needs to click on
	var portString, verifyURL string
	if (port != 443) && (port != 8443) {
		portString = fmt.Sprintf(":%v", port)
	}
	protocol := "https"
	if !httpsEnabled {
		protocol = "http"
	}
	verifyURL = fmt.Sprintf("%v://%v%v/ver?token=%v", protocol, hostName, portString, encodedToken)

	// Debugging output
	if debug {
		log.Printf(verifyURL)
	}

	// Send the verification email
	email := smtp2go.Email{
		From:     "Newdash <reply@newdash.io>",
		To:       []string{fmt.Sprintf("<%s>", emailAddr)},
		Subject:  "Please verify your email address",
		TextBody: fmt.Sprintf("Please visit this url to confirm your email address: %v", verifyURL),
		HtmlBody: fmt.Sprintf(`Please visit this url to verify your email address: <a href="%v">%v</a>`, verifyURL, verifyURL),
	}
	res, err := smtp2go.Send(&email)
	if err != nil {
		log.Printf("Error when sending verification email to: '%v', %v", emailAddr, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if res.Data.Error != "" {
		log.Printf("Error when sending verification email to: '%v', %v", emailAddr, res.Data.Error)
		http.Error(w, res.Data.Error, http.StatusInternalServerError)
		return
	}
	log.Printf("Verification email sent to '%v'", emailAddr)

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
	err = pg.QueryRow(context.Background(), dbQuery, verifyToken).Scan(&foundToken)
	if err != nil {
		log.Printf("Looking for existing token '%v' failed: %v", verifyToken, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if foundToken == 0 {
		log.Printf("A token '%s' has been submitted, but it's not present in the database", verifyToken)
		http.Error(w, "That token value isn't known to us.  Broken email link?", http.StatusBadRequest)
		return
	}

	// Update the token status in the database
	dbQuery = `
		UPDATE potential_customers
		SET token_verified = true
		WHERE token = $1`
	commandTag, err := pg.Exec(context.Background(), dbQuery, verifyToken)
	if err != nil {
		log.Printf("Updating token status failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if numRows := commandTag.RowsAffected(); numRows != 1 {
		log.Printf("Wrong number of rows (%v) affected updating token '%s' status", numRows, verifyToken)
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
	var emailAddress string
	dbQuery = `
		SELECT email
		FROM potential_customers
		WHERE token = $1`
	err = pg.QueryRow(context.Background(), dbQuery, verifyToken).Scan(&emailAddress)
	if err != nil {
		msg := fmt.Sprintf("Retrieving email address for token '%v' failed: %v", verifyToken, err)
		log.Printf(msg)
		emailAlert("Error when verifying token for Newdash interest", msg)
		return
	}
	if emailAddress == "" {
		msg := fmt.Sprintf("A token '%v' was verified, but its email address wasn't able to be retrieved", foundToken)
		emailAlert("Error when verifying token for Newdash interest", msg)
		return
	}

	// Send an email to the user, letting them know their registration has been confirmed
	confirmEmail := smtp2go.Email{
		From:     "Newdash <reply@newdash.io>",
		To:       []string{fmt.Sprintf("<%s>", emailAddress)},
		Subject:  "Thank you for confirming your interest in Redash Hosting",
		TextBody: fmt.Sprintf("Thank you for confirming your interest in our Redash Hosting.  We'll contact you regarding it in the next few days."),
		HtmlBody: fmt.Sprintf("Thank you for confirming your interest in our Redash Hosting.  We'll contact you regarding it in the next few days."),
	}
	res, err := smtp2go.Send(&confirmEmail)
	if err != nil {
		log.Printf("Error when sending registration confirmation email to: '%v', %v", emailAddress, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if res.Data.Error != "" {
		log.Printf("Error when sending registration confirmation email to: '%v', %v", emailAddress, res.Data.Error)
		http.Error(w, res.Data.Error, http.StatusInternalServerError)
		return
	}
	log.Printf("Registration confirmation email sent to '%v'", emailAddress)

	// Send an email alerting us to the newly registered interest
	content := fmt.Sprintf("Someone has registered their interest in Newdash hosting: %v", emailAddress)
	emailAlert("New verified interest in Newdash hosting", content)
}

func emailAlert(subject, content string) {
	email := smtp2go.Email{
		From:     "Newdash <reply@newdash.io>",
		To:       []string{fmt.Sprintf("<%s>", alertEmail)},
		Subject:  subject,
		TextBody: content,
		HtmlBody: content,
	}
	res, err := smtp2go.Send(&email)
	if err != nil {
		log.Printf("Error when sending email alert: %v", err)
		return
	}
	if res.Data.Error != "" {
		log.Printf("Error when sending email alert: %v", res.Data.Error)
		return
	}
	log.Printf("Alert email sent to '%v'", alertEmail)
}
