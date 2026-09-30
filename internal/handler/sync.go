package handler

import (
	"errors"
	"net/http"

	"github.com/marcosarl1/Circuito-API/internal/storage"
)

func SyncBucketStatus(syncer *storage.BucketSync) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		inProgress := syncer != nil && syncer.InProgress()
		writeJSON(writer, http.StatusOK, map[string]bool{"in_progress": inProgress})
	}
}

func SyncBucket(syncer *storage.BucketSync) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if syncer == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}
		result, err := syncer.Trigger(request.Context())
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrBucketNotConfigured):
				writeError(writer, http.StatusInternalServerError, "AWS_BUCKET_NAME não configurado")
			case errors.Is(err, storage.ErrSyncInProgress):
				writeError(writer, http.StatusConflict, "Sincronização já em andamento")
			default:
				writeError(writer, http.StatusInternalServerError, "Erro interno")
			}
			return
		}
		writeJSON(writer, http.StatusOK, result)
	}
}
