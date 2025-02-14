package middleware

import (
	"log"
	"net/http"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s - Host: %s, Path: %s", r.Method, r.URL.Path, r.Host, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}