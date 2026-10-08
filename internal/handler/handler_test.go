package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcosarl1/Circuito-API/internal/service"
)

var errTriggerDown = errors.New("trigger down")

func TestHealthReturnsOK(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	recorder := httptest.NewRecorder()

	Health(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected application/json, got %q", contentType)
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", response["status"])
	}
}
func TestReadyReturnsServiceUnavailableWithoutStore(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)

	recorder := httptest.NewRecorder()

	Ready(nil)(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["status"] != "not_ready" {
		t.Fatalf("expected status=not_ready, got %q", response["status"])
	}
}

func TestRequireAPIKeyRejectsMissingKey(t *testing.T) {
	protectedHandler := RequireAPIKey("expected-key")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/eventos", nil)

	recorder := httptest.NewRecorder()

	protectedHandler(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["detail"] != "Chave API inválida" {
		t.Fatalf("unexpected detail: %q", response["detail"])
	}
}

func TestRequireAPIKeyAllowsValidKey(t *testing.T) {
	protectedHandler := RequireAPIKey("expected-key")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/eventos", nil)
	request.Header.Set("X-API-Key", "expected-key")

	recorder := httptest.NewRecorder()

	protectedHandler(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestGetEventReturnsServiceUnavailableWithoutStore(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/eventos/2026090001", nil)

	recorder := httptest.NewRecorder()

	GetEvent(nil)(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["detail"] != "Banco de dados indisponível" {
		t.Fatalf("unexpected detail: %q", response["detail"])
	}
}

func newTestRequest(
	method string,
	path string,
	body string,
) *http.Request {
	request := httptest.NewRequest(method, path, nil)

	if body != "" {
		request.Body = io.NopCloser(strings.NewReader(body))
	}
	return request
}

func TestGetEventReturnsEvent(t *testing.T) {
	store := newFakeEventStore()
	store.events["2026090001"] = service.Event{
		ID:         "2026090001",
		NomeEvento: "Corrida Teste",
		Cidade:     "João Pessoa",
		Estado:     "PB",
	}

	request := newTestRequest(http.MethodGet, "/api/v1/eventos/2026090001", "")
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	GetEvent(store)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var evento service.Event
	if err := json.NewDecoder(recorder.Body).Decode(&evento); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if evento.ID != "2026090001" {
		t.Fatalf("expected id 2026090001, got %q", evento.ID)
	}
}

func TestGetEventReturnsNotFound(t *testing.T) {
	store := newFakeEventStore()

	request := newTestRequest(http.MethodGet, "/api/v1/eventos/2026090001", "")
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	GetEvent(store)(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expecteed status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestCreateEventReturnsCreated(t *testing.T) {
	store := newFakeEventStore()

	nextID := func(requestContext context.Context) (string, error) {
		return "2026090001", nil
	}

	request := newTestRequest(http.MethodPost, "/api/v1/eventos",
		`{
			"nome_evento": "Corrida teste",
			"cidade": "Joao Pessoa",
			"estado": "PB"
			}`,
	)

	recorder := httptest.NewRecorder()

	CreateEvent(store, nextID)(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var evento service.Event
	if err := json.NewDecoder(recorder.Body).Decode(&evento); err != nil {
		t.Fatalf("could not decode response, got %v", err)
	}

	if evento.ID != "2026090001" {
		t.Fatalf("expected id 2026090001, got %q", evento.ID)
	}
}

func TestUpdateEventReturnsUpdatedEvent(t *testing.T) {
	store := newFakeEventStore()
	store.events["2026090001"] = service.Event{
		ID:         "2026090001",
		NomeEvento: "Nome antiga",
		Cidade:     "Cidade antiga",
	}

	request := newTestRequest(http.MethodPatch, "/api/v1/eventos/2026090001",
		`{
			"nome_evento": "Nome novo",
			"cidade": "Cidade nova"
		}`,
	)
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	UpdateEvent(store)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var evento service.Event
	if err := json.NewDecoder(recorder.Body).Decode(&evento); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if evento.NomeEvento != "Nome novo" {
		t.Fatalf("expected nome_evento Nome Novo, got %q", evento.NomeEvento)
	}

	if evento.Cidade != "Cidade nova" {
		t.Fatalf("expected cidade Cidade nova, got %q", evento.Cidade)
	}
}

func TestDeleteEventReturnsNoContent(t *testing.T) {
	store := newFakeEventStore()
	store.events["2026090001"] = service.Event{
		ID: "2026090001",
	}

	request := newTestRequest(http.MethodDelete, "/api/v1/eventos/2026090001", "")
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	DeleteEvent(store)(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if _, exists := store.events["2026090001"]; exists {
		t.Fatal("expected evento to be deleted")
	}
}

func TestEventIDCanBeUsedAcrossLifecycle(t *testing.T) {
	store := newFakeEventStore()

	nextID := func(requestContext context.Context) (string, error) {
		return "2026090001", nil
	}

	createRequest := newTestRequest(http.MethodPost, "/api/v1/eventos",
		`{
			"nome_evento": "Corrida teste",
			"cidade": "Cidade teste",
			"estado": "PB"
		}`)

	createRecorder := httptest.NewRecorder()
	CreateEvent(store, nextID)(createRecorder, createRequest)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create: expected status %d, got %d; body=%s", http.StatusCreated, createRecorder.Code, createRecorder.Body.String())
	}

	var createdEvent service.Event
	if err := json.NewDecoder(createRecorder.Body).Decode(&createdEvent); err != nil {
		t.Fatalf("create: could not decode response: %v", err)
	}

	if createdEvent.ID != "2026090001" {
		t.Fatalf("create: expected id 2026090001, got %q", createdEvent.ID)
	}

	listRequest := newTestRequest(http.MethodGet, "/api/v1/eventos?page=1&size=20", "")
	listRecorder := httptest.NewRecorder()

	ListEvents(store)(listRecorder, listRequest)

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list: expected status %d, got %d; body=%s", http.StatusOK, listRecorder.Code, listRecorder.Body.String())
	}

	var page service.Page
	if err := json.NewDecoder(listRecorder.Body).Decode(&page); err != nil {
		t.Fatalf("list: could not decode response: %v", err)
	}

	if len(page.Eventos) != 1 {
		t.Fatalf("list: expected 1 event, got %d", len(page.Eventos))
	}

	listedID := page.Eventos[0].ID
	if listedID != createdEvent.ID {
		t.Fatalf("list: id mismatch: created %q listed=%q", createdEvent.ID, listedID)
	}

	getRequest := newTestRequest(http.MethodGet, "/api/v1/eventos/"+listedID, "")
	getRequest.SetPathValue("id", listedID)
	getRecorder := httptest.NewRecorder()

	GetEvent(store)(getRecorder, getRequest)

	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get: expected status %d, got %d; body=%s", http.StatusOK, getRecorder.Code, getRecorder.Body.String())
	}

	updateRequest := newTestRequest(http.MethodPatch, "/api/v1/eventos/"+listedID,
		`{
			"cidade": "Campina Grande"
		}`)
	updateRequest.SetPathValue("id", listedID)
	updateRecorder := httptest.NewRecorder()

	UpdateEvent(store)(updateRecorder, updateRequest)

	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update: expected status %d, got %d; body=%s", http.StatusOK, updateRecorder.Code, updateRecorder.Body.String())
	}

	deleteRequest := newTestRequest(http.MethodDelete, "/api/v1/eventos/"+listedID, "")
	deleteRequest.SetPathValue("id", listedID)
	deleteRecorder := httptest.NewRecorder()

	DeleteEvent(store)(deleteRecorder, deleteRequest)

	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("delete: expected status %d, got %d; body=%s", http.StatusNoContent, deleteRecorder.Code, deleteRecorder.Body.String())
	}

	finalGetRequest := newTestRequest(http.MethodGet, "/api/v1/eventos"+listedID, "")
	finalGetRequest.SetPathValue("id", listedID)
	finalGetRecorder := httptest.NewRecorder()

	GetEvent(store)(finalGetRecorder, finalGetRequest)

	if finalGetRecorder.Code != http.StatusNotFound {
		t.Fatalf("final get: expected status %d, got %d; body=%s", http.StatusNotFound, finalGetRecorder.Code, finalGetRecorder.Body.String())
	}
}

func TestUpdateEventRejectsInvalidHorario(t *testing.T) {
	store := newFakeEventStore()
	store.events["2026090001"] = service.Event{
		ID: "2026090001",
	}
	request := newTestRequest(http.MethodPatch, "/api/v1/eventos/2026090001", `{"horario": "7:30"}`)
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	UpdateEvent(store)(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; boyd=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
}

func TestUpdateEventUpdatesExtendFields(t *testing.T) {
	store := newFakeEventStore()
	store.events["2026090001"] = service.Event{
		ID:         "2026090001",
		NomeEvento: "Evento",
	}

	request := newTestRequest(http.MethodPatch, "/api/v1/eventos/2026090001",
		`{
			"organizador": "Nova organização",
			"distancias": ["5 KM", "10 KM"],
			"horario": "07:30",
			"patrocinado": true
		}`)

	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	UpdateEvent(store)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
}

func TestRunScrapeAcceptsFirstJob(t *testing.T) {
	store := newFakeJobStore()

	request := newTestRequest(http.MethodPost, "/api/v1/scrape/run", "")
	recorder := httptest.NewRecorder()

	RunScrape(store, func() string { return "job-1" }, nil)(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d; body=%s", recorder.Code, recorder.Body.String())
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response["job_id"] != "job-1" {
		t.Fatalf("unexpected job_id: %q", response["job_id"])
	}
}

func TestRunScrapeRejectsConcurrentJob(t *testing.T) {
	store := newFakeJobStore()
	handler := RunScrape(store, func() string { return "job-1" }, nil)

	first := newTestRequest(http.MethodPost, "/api/v1/scrape/run", "")
	firstRecorder := httptest.NewRecorder()
	handler(firstRecorder, first)
	if firstRecorder.Code != http.StatusAccepted {
		t.Fatalf("first: expected 202, got %d", firstRecorder.Code)
	}

	second := newTestRequest(http.MethodPost, "/api/v1/scrape/run", "")
	secondRecorder := httptest.NewRecorder()
	handler(secondRecorder, second)
	if secondRecorder.Code != http.StatusConflict {
		t.Fatalf("second: expected 409, got %d; body=%s", secondRecorder.Code, secondRecorder.Body.String())
	}
}

func TestRunScrapeWithoutStore(t *testing.T) {
	request := newTestRequest(http.MethodPost, "/api/v1/scrape/run", "")
	recorder := httptest.NewRecorder()

	RunScrape(nil, func() string { return "job-1" }, nil)(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestScrapeStatusReturnsJob(t *testing.T) {
	store := newFakeJobStore()
	store.jobs["job-1"] = &service.ScrapeJob{JobID: "job-1", Status: service.JobStatusQueued, StartedAt: service.NowISO()}

	request := newTestRequest(http.MethodGet, "/api/v1/scrape/status/job-1", "")
	request.SetPathValue("id", "job-1")
	recorder := httptest.NewRecorder()

	ScrapeStatus(store)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", recorder.Code, recorder.Body.String())
	}

	var job service.ScrapeJob
	if err := json.NewDecoder(recorder.Body).Decode(&job); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if job.JobID != "job-1" || job.Status != service.JobStatusQueued {
		t.Fatalf("unexpected job: %+v", job)
	}
}

func TestScrapeStatusReturnsNotFound(t *testing.T) {
	store := newFakeJobStore()

	request := newTestRequest(http.MethodGet, "/api/v1/scrape/status/missing", "")
	request.SetPathValue("id", "missing")
	recorder := httptest.NewRecorder()

	ScrapeStatus(store)(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestScrapeLastRunReturnsNullWithoutRuns(t *testing.T) {
	store := newFakeJobStore()

	request := newTestRequest(http.MethodGet, "/api/v1/scrape/last-run", "")
	recorder := httptest.NewRecorder()

	ScrapeLastRun(store)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if finishedAt, exists := response["finished_at"]; !exists || finishedAt != nil {
		t.Fatalf("expected null finished_at, got %v", response)
	}
}

func TestScrapeImportReportsNotImplemented(t *testing.T) {
	request := newTestRequest(http.MethodPost, "/api/v1/scrape/import", "")
	recorder := httptest.NewRecorder()

	ScrapeImport()(recorder, request)

	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", recorder.Code)
	}
}

func TestSyncBucketStatusReportsIdle(t *testing.T) {
	request := newTestRequest(http.MethodGet, "/api/v1/sync-bucket/status", "")
	recorder := httptest.NewRecorder()

	SyncBucketStatus(nil)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response map[string]bool
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response["in_progress"] {
		t.Fatal("expected in_progress=false")
	}
}

func TestRunScrapeTriggersWorker(t *testing.T) {
	store := newFakeJobStore()
	triggered := false

	request := newTestRequest(http.MethodPost, "/api/v1/scrape/run", "")
	recorder := httptest.NewRecorder()

	RunScrape(store, func() string { return "job-1" }, func(context.Context) error {
		triggered = true
		return nil
	})(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d; body=%s", recorder.Code, recorder.Body.String())
	}
	if !triggered {
		t.Fatal("expected worker trigger to run")
	}
}

func TestRunScrapeAbandonsJobWhenTriggerFails(t *testing.T) {
	store := newFakeJobStore()

	request := newTestRequest(http.MethodPost, "/api/v1/scrape/run", "")
	recorder := httptest.NewRecorder()

	RunScrape(store, func() string { return "job-1" }, func(context.Context) error {
		return errTriggerDown
	})(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d; body=%s", recorder.Code, recorder.Body.String())
	}

	job, err := store.GetScrapeJob(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("job should still exist: %v", err)
	}
	if job.Status != service.JobStatusFailed {
		t.Fatalf("abandoned job should be failed, got %q", job.Status)
	}

	second := newTestRequest(http.MethodPost, "/api/v1/scrape/run", "")
	secondRecorder := httptest.NewRecorder()
	RunScrape(store, func() string { return "job-2" }, nil)(secondRecorder, second)
	if secondRecorder.Code != http.StatusAccepted {
		t.Fatalf("slot should be free after abandon, got %d", secondRecorder.Code)
	}
}

func awaitingJob(store *fakeJobStore, id string) {
	store.jobs[id] = &service.ScrapeJob{JobID: id, Status: service.JobStatusAwaitingImport, StartedAt: service.NowISO()}
	store.locked = true
}

func TestConfirmScrapeJobResumesImport(t *testing.T) {
	store := newFakeJobStore()
	awaitingJob(store, "job-1")
	triggered := false

	request := newTestRequest(http.MethodPost, "/api/v1/scrape/confirm/job-1", "")
	request.SetPathValue("id", "job-1")
	recorder := httptest.NewRecorder()

	ConfirmScrapeJob(store, func(context.Context) error {
		triggered = true
		return nil
	})(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d; body=%s", recorder.Code, recorder.Body.String())
	}
	if !triggered {
		t.Fatal("expected worker trigger to run")
	}
	job, _ := store.GetScrapeJob(context.Background(), "job-1")
	if job.Status != service.JobStatusQueued {
		t.Fatalf("expected back to queued, got %q", job.Status)
	}
}

func TestConfirmScrapeJobRejectsWrongState(t *testing.T) {
	store := newFakeJobStore()
	store.jobs["job-1"] = &service.ScrapeJob{JobID: "job-1", Status: service.JobStatusRunning}

	request := newTestRequest(http.MethodPost, "/api/v1/scrape/confirm/job-1", "")
	request.SetPathValue("id", "job-1")
	recorder := httptest.NewRecorder()

	ConfirmScrapeJob(store, nil)(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}

	missing := newTestRequest(http.MethodPost, "/api/v1/scrape/confirm/void", "")
	missing.SetPathValue("id", "void")
	missingRecorder := httptest.NewRecorder()
	ConfirmScrapeJob(store, nil)(missingRecorder, missing)
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", missingRecorder.Code)
	}
}

func TestCancelScrapeJobFreesSlot(t *testing.T) {
	store := newFakeJobStore()
	awaitingJob(store, "job-1")

	request := newTestRequest(http.MethodPost, "/api/v1/scrape/cancel/job-1", "")
	request.SetPathValue("id", "job-1")
	recorder := httptest.NewRecorder()

	CancelScrapeJob(store)(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}

	next := newTestRequest(http.MethodPost, "/api/v1/scrape/run", "")
	nextRecorder := httptest.NewRecorder()
	RunScrape(store, func() string { return "job-2" }, nil)(nextRecorder, next)
	if nextRecorder.Code != http.StatusAccepted {
		t.Fatalf("slot should be free after cancel, got %d", nextRecorder.Code)
	}
}
