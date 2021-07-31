package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx"
)

// ConnectPostgreSQL creates a connection pool to the PostgreSQL server
func ConnectPostgreSQL() (err error) {
	// PostgreSQL configuration info
	pgConfig := new(pgx.ConnConfig)
	pgConfig.Host = "/var/run/postgresql"
	pgConfig.Database = "newdash_interest"
	pgConfig.User = "newdash"

	pgPoolConfig := pgx.ConnPoolConfig{*pgConfig, 20, nil, 2 * time.Second}
	pg, err = pgx.NewConnPool(pgPoolConfig)
	if err != nil {
		return fmt.Errorf("Couldn't connect to PostgreSQL server: %v\n", err)
	}

	// Log successful connection
	log.Printf("Connected to PostgreSQL server: %v:%v\n", "localhost", 5432)

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
