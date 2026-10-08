package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/marcosarl1/Circuito-API/internal/repository"
	"github.com/marcosarl1/Circuito-API/internal/service"
)

func RunScrape(jobStore JobStore, newID func() string, trigger func(context.Context) error) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if jobStore == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}
		job, err := jobStore.AcquireScrapeJob(request.Context(), newID(), service.NowISO())
		if err != nil {
			if errors.Is(err, repository.ErrScrapeInProgress) {
				writeError(writer, http.StatusConflict, "Scrape já está em andamento")
				return
			}
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		if trigger != nil {
			if err := trigger(request.Context()); err != nil {
				slog.Error("scrape trigger failed, abandoning job", "job_id", job.JobID, "err", err)
				if abandonErr := jobStore.AbandonScrapeJob(request.Context(), job.JobID, "não foi possível iniciar o worker"); abandonErr != nil {
					slog.Error("scrape abandon failed", "job_id", job.JobID, "err", abandonErr)
				}
				writeError(writer, http.StatusServiceUnavailable, "Serviço de scraping indisponível")
				return
			}
		}
		writeJSON(writer, http.StatusAccepted, map[string]string{"job_id": job.JobID})
	}
}

func ScrapeStatus(jobStore JobStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if jobStore == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}
		job, err := jobStore.GetScrapeJob(request.Context(), request.PathValue("id"))
		if err != nil {
			if errors.Is(err, repository.ErrScrapeJobNotFound) {
				writeError(writer, http.StatusNotFound, "Job não encontrado")
				return
			}
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		writeJSON(writer, http.StatusOK, job)
	}
}

func ScrapeLastRun(jobStore JobStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if jobStore == nil {
			writeError(writer, http.StatusServiceUnavailable, "Banco de dados indisponível")
			return
		}
		finishedAt, err := jobStore.GetLastScrapeRun(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "Erro interno")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"finished_at": finishedAt})
	}
}

func ScrapeImport() http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writeError(writer, http.StatusNotImplemented, "Importação indisponível: worker de scraping não configurado")
	}
}
