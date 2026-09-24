package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/marcosarl1/Circuito-API/internal/repository"
)

func ListEvents(store *repository.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Banco de dados indisponível"})
			return
		}
		q := r.URL.Query()
		page, _ := strconv.ParseInt(q.Get("page"), 10, 64)
		size, _ := strconv.ParseInt(q.Get("size"), 10, 64)
		items, total, err := store.ListEvents(r.Context(), page, size, q.Get("estado"), q.Get("q"))
		if err != nil {
			http.Error(w, `{"detail":"erro interno"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"eventos": items, "total": total})
	}
}

func GetEvent(store *repository.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		e, err := store.FindEvent(r.Context(), id)
		if err != nil {
			http.Error(w, `{"detail":"Evento não encontrado"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(e)
	}
}
