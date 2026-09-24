package service

import "time"

type Event struct {
	ID              string      `bson:"_id" json:"id"`
	NomeEvento      string      `bson:"nome_evento" json:"nome_evento"`
	Cidade          string      `bson:"cidade" json:"cidade"`
	Estado          string      `bson:"estado" json:"estado"`
	DataRealizacao  string      `bson:"data_realizacao" json:"data_realizacao"`
	DatasRealizacao []time.Time `bson:"datas_realizacao" json:"-"`
}

type UpdateEventRequest struct {
	NomeEvento     *string `json:"nome_evento"`
	Cidade         *string `json:"cidade"`
	Estado         *string `json:"estado"`
	DataRealizacao *string `json:"data_realizacao"`
}

type Page struct {
	Eventos    []Event `json:"eventos"`
	Total      int64   `json:"total"`
	TotalPages int64   `json:"total_pages"`
	Page       int64   `json:"page"`
	Size       int64   `json:"size"`
	HasNext    bool    `json:"has_next"`
	HasPrev    bool    `json:"has_prev"`
}
