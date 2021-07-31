package main

import (
	"fmt"
	"net/http"
	"time"
)

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
