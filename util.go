package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

// ConnectPostgreSQL creates a connection to the PostgreSQL server
func ConnectPostgreSQL() (err error) {
	// PostgreSQL configuration info
	os.Setenv("PGHOST", "/var/run/postgresql")
	os.Setenv("PGDATABASE", "newdash_interest")
	os.Setenv("PGUSER", "newdash")
	os.Setenv("PGPASSFILE", "/home/newdash/.pgpass")
	pg, err = pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return
	}

	// Log successful connection
	log.Printf("Connected to PostgreSQL server: %v:%v", "localhost", 5432)

	return nil
}

// Wrapper function to log incoming https requests.
func logReq(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Write request details to the request log
		fmt.Fprintf(reqLog, "%v - %s [%s] \"%s %s %s\" \"-\" \"-\" \"%s\" \"%s\"\n", r.RemoteAddr,
			"-", time.Now().Format(time.RFC3339Nano), r.Method, r.URL, r.Proto, r.Referer(), r.Header.Get("User-Agent"))

		// Call the original function
		fn(w, r)
	}
}
