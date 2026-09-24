package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/repository"
)

func Health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func Ready(store *repository.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if store == nil || store.Client == nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "reason": "db not connected"})
			return
		}
		pingContext, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()
		if err := store.Client.Ping(pingContext, nil); err != nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "reason": "ping failed"})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
	}
}
