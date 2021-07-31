package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// MainHandler renders the main page
func MainHandler(w http.ResponseWriter, r *http.Request) {
	// Render the page
	t := tmpl.Lookup("mainPage")
	err := t.Execute(w, nil)
	if err != nil {
		log.Printf("Error: %s", err)
	}
}

// SubscribeHandler renders the main page
func SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Generate a random token - maybe copy the kuuid code from dbhub.io?
	verifyToken := "abc123"
	verifyURL := fmt.Sprintf("http://%v:%v/verify?token=%v", hostName, port, verifyToken)
	//verifyURL := fmt.Sprintf("https://%v:%v/verify?token=%v", hostName, port, verifyToken) // TODO: Use this https url instead

	// TODO: Store the provided email address and token in the database

	// Send verification email
	from := mail.NewEmail("Newdash.io", "interest@newdash.io")
	subject := "Please verify your email address"
	to := mail.NewEmail("", "justin@postgresql.org") // TODO: This should be the email address provided by the user
	plainTextContent := fmt.Sprintf("Please visit this url to confirm your email address: %v", verifyURL)
	htmlContent := fmt.Sprintf(`Please visit this url to verify your email address: <a href="%v">%v</a>`, verifyURL, verifyURL)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(sendGridKey)
	_, err := client.Send(message)
	if err != nil {
		log.Println(err)
	}
	fmt.Println("Verification email sent")

	// Render the page
	t := tmpl.Lookup("subscribePage")
	err = t.Execute(w, nil)
	if err != nil {
		log.Printf("Error: %s", err)
	}
}

// VerifyHandler renders the main page
func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract the token from the link

	// TODO: Make sure the token is found in the database

	// TODO: Thank the user for registering their interest in newdash.io

	// Render the page
	t := tmpl.Lookup("verifyPage")
	err := t.Execute(w, nil)
	if err != nil {
		log.Printf("Error: %s", err)
	}
}
