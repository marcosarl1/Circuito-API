package service

import (
	"time"
)

type Percurso struct {
	LocalLargada string  `bson:"local_largada" json:"local_largada"`
	Trajeto      *string `bson:"trajeto" json:"trajeto"`
}

type Kit struct {
	Nome          string      `bson:"nome" json:"nome"`
	Itens         StringSlice `bson:"itens" json:"itens"`
	LocalRetirada *string     `bson:"local_retirada" json:"local_retirada"`
	DataRetirada  *time.Time  `bson:"data_retirada" json:"data_retirada"`
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
	Distancias       StringSlice `bson:"distancias" json:"distancias"`
	Horario          *string     `bson:"horario" json:"horario"`
	URLInscricao     *string     `bson:"url_inscricao" json:"url_inscricao"`
	URLImagem        *string     `bson:"url_imagem" json:"url_imagem"`
	LinkEdital       *string     `bson:"link_edital" json:"link_edital"`
	Categorias       StringSlice `bson:"categorias" json:"categorias"`
	CategoriasPrem   *string     `bson:"categorias_premiadas" json:"categorias_premiadas"`
	Preco            *string     `bson:"preco" json:"preco"`
	PrecosEntries    []any       `bson:"precos_entries" json:"precos_entries"`
	Patrocinado      bool        `bson:"patrocinado" json:"patrocinado"`
	Percurso         *Percurso   `bson:"percurso" json:"percurso"`
	Kits             []Kit       `bson:"kits" json:"kits"`
	CamposProtegidos StringSlice `bson:"campos_protegidos" json:"campos_protegidos"`
	ListaPrecos      []string    `bson:"-" json:"lista_precos"`
}

func (event *Event) Normalize() {
	if event.DataRealizacao == "" && len(event.DatasRealizacao) > 0 {
		event.DataRealizacao = formatDataRealizacao(event.DatasRealizacao[0])
	}

	event.Distancias = StringSlice(normalizeDistancias(event.Distancias))
	event.PrecosEntries = parsePrecosEntries(event.PrecosEntries)

	if event.Categorias == nil {
		event.Categorias = StringSlice{}
	}
	if event.Kits == nil {
		event.Kits = []Kit{}
	}
	if event.CamposProtegidos == nil {
		event.CamposProtegidos = StringSlice{}
	}

	event.ListaPrecos = formatPriceList(event.PrecosEntries, event.Preco)
}

type UpdateEventRequest struct {
	NomeEvento       *string      `json:"nome_evento"`
	Cidade           *string      `json:"cidade"`
	Estado           *string      `json:"estado"`
	Organizador      *string      `json:"organizador"`
	SiteColeta       *string      `json:"site_coleta"`
	DataColeta       *time.Time   `json:"data_coleta"`
	DataRealizacao   *string      `json:"data_realizacao"`
	DatasRealizacao  *[]time.Time `json:"datas_realizacao"`
	Distancias       *[]string    `json:"distancias"`
	Horario          *string      `json:"horario"`
	URLInscricao     *string      `json:"url_inscricao"`
	URLImagem        *string      `json:"url_imagem"`
	LinkEdital       *string      `json:"link_edital"`
	Categorias       *[]string    `json:"categorias"`
	CategoriasPrem   *string      `json:"categorias_premiadas"`
	Preco            *string      `json:"preco"`
	PrecosEntries    *[]any       `json:"precos_entries"`
	Patrocinado      *bool        `json:"patrocinado"`
	Percurso         *Percurso    `json:"percurso"`
	Kits             *[]Kit       `json:"kits"`
	CamposProtegidos *[]string    `json:"campos_protegidos"`
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
