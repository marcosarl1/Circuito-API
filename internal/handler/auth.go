package handler

import (
	"crypto/subtle"
	"net/http"
)

func RequireAPIKey(expectedKey string) func(http.HandlerFunc) http.HandlerFunc {
	return func(protectedHandler http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			if expectedKey == "" {
				writeError(writer, http.StatusInternalServerError, "API_KEY não configurada")
				return
			}
			providedKey := request.Header.Get("X-API-Key")
			if subtle.ConstantTimeCompare([]byte(providedKey), []byte(expectedKey)) != 1 {
				writeError(writer, http.StatusUnauthorized, "Chave API inválida")
				return
			}
			protectedHandler(writer, request)
		}
	}
}
