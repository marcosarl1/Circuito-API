package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/marcosarl1/Circuito-API/internal/repository"
	"github.com/marcosarl1/Circuito-API/internal/service"
)

func ListEvents(store *repository.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if store == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}
		query := request.URL.Query()
		page, _ := strconv.ParseInt(query.Get("page"), 10, 64)
		size, _ := strconv.ParseInt(query.Get("size"), 10, 64)
		eventos, total, err := store.ListEvents(request.Context(), page, size, query.Get("estado"), query.Get("q"))
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"eventos": eventos, "total": total})
	}
}

func GetEvent(store *repository.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if store == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}
		eventID := request.PathValue("id")
		evento, err := store.FindEvent(request.Context(), eventID)
		if err != nil {
			writeError(writer, http.StatusNotFound, "Evento não encontrado")
			return
		}
		writeJSON(writer, http.StatusOK, evento)
	}
}

func CreateEvent(store *repository.Store, nextID func(context.Context) (string, error)) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if store == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}
		var newEvent service.Event
		if err := json.NewDecoder(request.Body).Decode(&newEvent); err != nil {
			writeError(writer, http.StatusBadRequest, "Corpo inválido")
			return
		}
		eventID, err := nextID(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		newEvent.ID = eventID
		createdEvent, err := store.CreateEvent(request.Context(), newEvent)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		writeJSON(writer, http.StatusCreated, createdEvent)
	}
}

func UpdateEvent(store *repository.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if store == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}

		var updateRequest service.UpdateEventRequest
		if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
			writeError(writer, http.StatusBadRequest, "Body inválido")
			return
		}

		updates := make(map[string]any)

		if updateRequest.NomeEvento != nil {
			updates["nome_evento"] = *updateRequest.NomeEvento
		}
		if updateRequest.Cidade != nil {
			updates["cidade"] = *updateRequest.Cidade
		}
		if updateRequest.Estado != nil {
			updates["estado"] = *updateRequest.Estado
		}
		if updateRequest.DataRealizacao != nil {
			updates["data_realizacao"] = *updateRequest.DataRealizacao
		}

		if len(updates) == 0 {
			writeError(writer, http.StatusBadRequest, "Nenhum campo para atualizar")
			return
		}

		eventID := request.PathValue("id")

		updatedEvent, err := store.UpdateEvent(request.Context(), eventID, updates)
		if err != nil {
			writeError(writer, http.StatusNotFound, "Evento não encontrado")
			return
		}

		writeJSON(writer, http.StatusOK, updatedEvent)
	}
}

func DeleteEvent(store *repository.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if store == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}
		eventID := request.PathValue("id")
		deleted, err := store.DeleteEvent(request.Context(), eventID)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		if !deleted {
			writeError(writer, http.StatusNotFound, "Evento não encontrado")
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}
}
