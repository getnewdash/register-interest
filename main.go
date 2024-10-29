package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	gz "github.com/NYTimes/gziphandler"
	"github.com/jackc/pgx/v5"
	"gopkg.in/go-playground/validator.v8"
)

var (
	// Email address to send alerts to. eg for new sign ups, any errors, etc.
	alertEmail string

	// Output extra debugging information?
	// To enable, define "DEBUG" in the environment with any value that's not an empty string
	debug bool

	// Host name and port to listen on
	hostName string
	port     int

	// HTTPS certificate paths
	certPath, certKey string
	httpsEnabled      bool

	// The request log
	requestLog string
	reqLog     *os.File

	// PostgreSQL connection handle
	pg *pgx.Conn

	// Our parsed HTML templates
	tmpl *template.Template

	// Used for validating email addresses
	validate *validator.Validate
)

func main() {
	// Load the required values from environment variables (easy for working with CI systems)
	var err error
	var ok bool

	// HTTPS Certificate pieces
	certPath, ok = os.LookupEnv("HTTPS_CERT_PATH")
	if !ok {
		log.Println("HTTPS_CERT_PATH not set, https is disabled")
	}
	certKey, ok = os.LookupEnv("HTTPS_CERT_KEY")
	if !ok {
		log.Println("HTTPS_CERT_KEY not set, https is disabled")
	}
	if certPath != "" && certKey != "" {
		httpsEnabled = true
		log.Println("HTTPS enabled")
	}

	// Host:port to listen on
	hostName, ok = os.LookupEnv("HOSTNAME")
	if !ok {
		log.Fatal("HOSTNAME not set")
	}
	p, ok := os.LookupEnv("PORT")
	if !ok {
		log.Println("PORT not set, using default")
		if httpsEnabled {
			p = "443"
		} else {
			// Non HTTPS default is 8080, as we're assuming local development setup
			p = "8080"
		}
	}
	if p == "" {
		log.Println("PORT is empty, using default")
		if httpsEnabled {
			p = "443"
		} else {
			// Non HTTPS default is 8080, as we're assuming local development setup
			p = "8080"
		}
	}
	port, err = strconv.Atoi(p)
	if err != nil {
		log.Fatal(err)
	}

	// Path to the request log
	requestLog, ok = os.LookupEnv("REQUEST_LOG")
	if !ok {
		log.Fatal("REQUEST_LOG not set")
	}

	// Email address to send alerts too
	alertEmail, ok = os.LookupEnv("ALERT_EMAIL")
	if !ok {
		log.Fatal("ALERT_EMAIL not set")
	}

	// SMTP2Go API key
	_, ok = os.LookupEnv("SMTP2GO_API_KEY")
	if !ok {
		log.Fatal("SMTP2GO_API_KEY not set")
	}

	// Switch on debug mode?
	z, _ := os.LookupEnv("DEBUG")
	if z != "" {
		debug = true
		log.Println("Enabling debug logging")
	}

	// Connect to PostgreSQL server
	err = ConnectPostgreSQL()
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Parse our template files
	tmpl = template.Must(template.New("templates").ParseGlob(filepath.Join("templates", "*.html")))

	// Open the request log for writing
	reqLog, err = os.OpenFile(requestLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0750)
	if err != nil {
		log.Fatalf("Error when opening request log: %s", err)
	}
	defer reqLog.Close()
	log.Printf("Request log opened: %s", requestLog)

	// Set up validation
	config := &validator.Config{TagName: "validate"} // TODO: What does the 'TagName: validate' as shown in all the examples actually do?
	validate = validator.New(config)

	// Register page handlers
	http.Handle("/", gz.GzipHandler(logReq(MainHandler)))
	http.Handle("/sub", gz.GzipHandler(logReq(SubscribeHandler)))
	http.Handle("/ver", gz.GzipHandler(logReq(VerifyHandler)))
	http.Handle("/verify", gz.GzipHandler(logReq(VerifyHandler)))

	// Static files
	http.Handle("/js/main.min.js", gz.GzipHandler(logReq(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("js", "main.min.js"))
	})))
	http.Handle("/css/shared.css", gz.GzipHandler(logReq(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("css", "shared.css"))
	})))
	http.Handle("/image/github.png", gz.GzipHandler(logReq(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("image", "github.png"))
	})))

	// Set up the web server
	srv := &http.Server{
		Addr: fmt.Sprintf(":%v", port),
	}
	if httpsEnabled {
		srv.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12, // TLS 1.2 is now the lowest acceptable level
		}
	}

	// Start web server
	if httpsEnabled {
		log.Printf("WebUI server starting on https://%s:%v", "localhost", port)
		err = srv.ListenAndServeTLS(certPath, certKey)
	} else {
		log.Printf("WebUI server starting on http://%s:%v", "localhost", port)
		err = srv.ListenAndServe()
	}
	if err != nil {
		log.Fatalln(err)
	}

	// Disconnect from PostgreSQL
	pg.Close(context.Background())
}
