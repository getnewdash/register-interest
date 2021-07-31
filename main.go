package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	gz "github.com/NYTimes/gziphandler"
)

var (
	// The request log
	reqLog *os.File

	// The Sendgrid API key
	sendGridKey string

	// Our parsed HTML templates
	tmpl *template.Template
)

const (
	certPath   = "/etc/letsencrypt/live/newdash.io/fullchain.pem"
	certKey    = "/etc/letsencrypt/live/newdash.io/privkey.pem"
	hostName   = "localhost" // TODO: Change this before putting into prod
	port       = 8443
	requestLog = "request.log"
	//requestLog = "/var/log/newdash/request.log"
)

func main() {
	// Make sure the Sendgrid API key is available
	var ok bool
	sendGridKey, ok = os.LookupEnv("SENDGRID_API_KEY")
	if !ok {
		log.Fatal("SENDGRID_API_KEY not set")
	}

	// Parse our template files
	tmpl = template.Must(template.New("templates").ParseGlob(filepath.Join("templates", "*.html")))

	// Open the request log for writing
	var err error
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

	// Start web server
	log.Printf("WebUI server starting on https://%s:%v\n", "localhost", port)
	srv := &http.Server{
		Addr: fmt.Sprintf(":%v", port),
		//TLSConfig: &tls.Config{
		//	MinVersion: tls.VersionTLS12, // TLS 1.2 is now the lowest acceptable level
		//},
	}
	err = srv.ListenAndServe()
	//err = srv.ListenAndServeTLS(certPath, certKey)
	if err != nil {
		log.Fatalln(err)
	}
}
