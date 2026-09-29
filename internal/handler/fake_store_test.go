package handler

import (
	"context"

	"github.com/marcosarl1/Circuito-API/internal/repository"
	"github.com/marcosarl1/Circuito-API/internal/service"
)

type fakeEventStore struct {
	events map[string]service.Event
}

func newFakeEventStore() *fakeEventStore {
	return &fakeEventStore{
		events: make(map[string]service.Event),
	}
}

func (store *fakeEventStore) ListEvents(
	requestContext context.Context,
	page int64,
	size int64,
	estado string,
	search string,
) ([]service.Event, int64, error) {
	eventos := make([]service.Event, 0, len(store.events))

	for _, evento := range store.events {
		eventos = append(eventos, evento)
	}

	return eventos, int64(len(eventos)), nil
}

func (store *fakeEventStore) FindEvent(
	requestContext context.Context,
	eventID string,
) (*service.Event, error) {
	normalizedID, err := service.NormalizeEventID(eventID)
	if err != nil {
		return nil, err
	}

	event, exists := store.events[normalizedID]
	if !exists {
		return nil, repository.ErrEventNotFound
	}

	return &event, nil
}

func (store *fakeEventStore) CreateEvent(
	requestContext context.Context, newEvent service.Event,
) (*service.Event, error) {
	store.events[newEvent.ID] = newEvent
	return &newEvent, nil
}

func (store *fakeEventStore) UpdateEvent(
	requestContext context.Context,
	eventID string,
	updates map[string]any,
) (*service.Event, error) {
	normalizedID, err := service.NormalizeEventID(eventID)
	if err != nil {
		return nil, err
	}

	event, exists := store.events[normalizedID]
	if !exists {
		return nil, repository.ErrEventNotFound
	}

	if nomeEvento, ok := updates["nome_evento"].(string); ok {
		event.NomeEvento = nomeEvento
	}

	if cidade, ok := updates["cidade"].(string); ok {
		event.Cidade = cidade
	}

	if estado, ok := updates["estado"].(string); ok {
		event.Estado = estado
	}

	if organizador, ok := updates["organizador"].(string); ok {
		event.Organizador = organizador
	}

	if siteColeta, ok := updates["site_coleta"].(string); ok {
		event.SiteColeta = siteColeta
	}

	if dataRealizacao, ok := updates["data_realizacao"].(string); ok {
		event.DataRealizacao = dataRealizacao
	}

	if distancias, ok := updates["distancias"].([]string); ok {
		event.Distancias = distancias
	}

	if horario, ok := updates["horario"].(string); ok {
		event.Horario = &horario
	}

	if patrocinado, ok := updates["patrocinado"].(bool); ok {
		event.Patrocinado = patrocinado
	}

	event.Normalize()
	store.events[normalizedID] = event

	return &event, nil
}

func (store *fakeEventStore) DeleteEvent(
	requestContext context.Context,
	eventID string,
) (bool, error) {
	normalizedID, err := service.NormalizeEventID(eventID)
	if err != nil {
		return false, err
	}

	if _, exists := store.events[normalizedID]; !exists {
		return false, nil
	}

	delete(store.events, normalizedID)

	return true, nil
}
