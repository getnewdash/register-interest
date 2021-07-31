package main

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	gz "github.com/NYTimes/gziphandler"
	"github.com/jackc/pgx"
)

var (
	// Host name and port to listen on
	hostName string
	port     int

	// HTTPS certificate paths
	certPath, certKey string

	// The request log
	requestLog string
	reqLog     *os.File

	// PostgreSQL connection pool handle
	pg *pgx.ConnPool

	// The Sendgrid API key
	sendGridKey string

	// Our parsed HTML templates
	tmpl *template.Template
)

func main() {
	// Load the required values from environment variables (easy for working with Jenkins)
	var err error
	var httpsEnabled, ok bool

	// HTTPS Certificate pieces
	certPath, ok = os.LookupEnv("HTTPS_CERT_PATH")
	if !ok {
		log.Println("HTTPS_CERT_PATH not set")
	}
	certKey, ok = os.LookupEnv("HTTPS_CERT_KEY")
	if !ok {
		log.Println("HTTPS_CERT_KEY not set")
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
		log.Fatal("PORT not set")
	}
	if p == "" {
		log.Fatal("PORT is empty")
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

	// Sendgrid API
	sendGridKey, ok = os.LookupEnv("SENDGRID_API_KEY")
	if !ok {
		log.Fatal("SENDGRID_API_KEY not set")
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
		log.Fatalf("Error when opening request log: %s\n", err)
	}
	defer reqLog.Close()
	log.Printf("Request log opened: %s\n", requestLog)

	// Register page handlers
	http.Handle("/", gz.GzipHandler(logReq(MainHandler)))
	http.Handle("/subscribe", gz.GzipHandler(logReq(SubscribeHandler)))
	http.Handle("/verify", gz.GzipHandler(logReq(VerifyHandler)))

	// CSS
	http.Handle("/css/shared.css", gz.GzipHandler(logReq(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("css", "shared.css"))
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
	log.Printf("WebUI server starting on https://%s:%v\n", "localhost", port)
	if httpsEnabled {
		err = srv.ListenAndServeTLS(certPath, certKey)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		log.Fatalln(err)
	}

	// Disconnect from PostgreSQL
	pg.Close()
}
