package middleware

import (
	"net/http"
	"time"
)

func Timeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, `{"error":"request timeout"}`)
	}
}
