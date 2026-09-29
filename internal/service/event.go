package service

import (
	"time"
)

type Percurso struct {
	LocalLargada string  `bson:"local_largada" json:"local_largada"`
	Trajeto      *string `bson:"trajeto" json:"trajeto"`
}

type Kit struct {
	Nome          string     `bson:"nome" json:"nome"`
	Itens         []string   `bson:"itens" json:"itens"`
	LocalRetirada *string    `bson:"local_retirada" json:"local_retirada"`
	DataRetirada  *time.Time `bson:"data_retirada" json:"data_retirada"`
}

type Event struct {
	ID               string      `bson:"_id" json:"id"`
	NomeEvento       string      `bson:"nome_evento" json:"nome_evento"`
	Cidade           string      `bson:"cidade" json:"cidade"`
	Estado           string      `bson:"estado" json:"estado"`
	Organizador      string      `bson:"organizador" json:"organizador"`
	SiteColeta       string      `bson:"site_coleta" json:"site_coleta"`
	DataColeta       *time.Time  `bson:"data_coleta" json:"data_coleta"`
	CreatedAt        *time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt        *time.Time  `bson:"updated_at" json:"updated_at"`
	DataRealizacao   string      `bson:"data_realizacao" json:"data_realizacao"`
	DatasRealizacao  []time.Time `bson:"datas_realizacao" json:"-"`
	Distancias       []string    `bson:"distancias" json:"distancias"`
	Horario          *string     `bson:"horario" json:"horario"`
	URLInscricao     *string     `bson:"url_inscricao" json:"url_inscricao"`
	URLImagem        *string     `bson:"url_imagem" json:"url_imagem"`
	LinkEdital       *string     `bson:"link_edital" json:"link_edital"`
	Categorias       []string    `bson:"categorias" json:"categorias"`
	CategoriasPrem   *string     `bson:"categorias_premiadas" json:"categorias_premiadas"`
	Preco            *string     `bson:"preco" json:"preco"`
	PrecosEntries    []any       `bson:"precos_entries" json:"precos_entries"`
	Patrocinado      bool        `bson:"patrocinado" json:"patrocinado"`
	Percurso         *Percurso   `bson:"percurso" json:"percurso"`
	Kits             []Kit       `bson:"kits" json:"kits"`
	CamposProtegidos []string    `bson:"campos_protegidos" json:"campos_protegidos"`
}

func (event *Event) Normalize() {
	event.Distancias = normalizeDistancias(event.Distancias)
	event.PrecosEntries = parsePrecosEntries(event.PrecosEntries)

	if event.Categorias == nil {
		event.Categorias = []string{}
	}
	if event.Kits == nil {
		event.Kits = []Kit{}
	}
	if event.CamposProtegidos == nil {
		event.CamposProtegidos = []string{}
	}
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
