package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-Id")

		if requestID == "" || len(requestID) > 128 {
			requestID = newRequestID()
		}

		writer.Header().Set("X-Request-Id", requestID)
		next.ServeHTTP(writer, request)
	})
}

func newRequestID() string {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return ""
	}

	return hex.EncodeToString(randomBytes)
}
