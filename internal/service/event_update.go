package service

func (request UpdateEventRequest) ToUpdates() (map[string]any, error) {
	updates := make(map[string]any)

	if request.NomeEvento != nil {
		updates["nome_evento"] = *request.NomeEvento
	}
	if request.Cidade != nil {
		updates["cidade"] = *request.Cidade
	}
	if request.Estado != nil {
		updates["estado"] = *request.Estado
	}
	if request.Organizador != nil {
		updates["organizador"] = *request.Organizador
	}
	if request.SiteColeta != nil {
		updates["site_coleta"] = *request.SiteColeta
	}
	if request.DataColeta != nil {
		updates["data_coleta"] = *request.DataColeta
	}
	if request.DataRealizacao != nil {
		updates["data_realizacao"] = *request.DataRealizacao
	}
	if request.DatasRealizacao != nil {
		updates["datas_realizacao"] = *request.DatasRealizacao
	}
	if request.Distancias != nil {
		updates["distancias"] = normalizeDistancias(*request.Distancias)
	}
	if request.Horario != nil {
		if err := validateHorario(request.Horario); err != nil {
			return nil, err
		}
		updates["horario"] = *request.Horario
	}
	if request.URLInscricao != nil {
		updates["url_inscricao"] = *request.URLInscricao
	}
	if request.URLImagem != nil {
		updates["url_imagem"] = *request.URLImagem
	}
	if request.LinkEdital != nil {
		updates["link_edital"] = *request.LinkEdital
	}
	if request.Categorias != nil {
		updates["categorias"] = *&request.Categorias
	}
	if request.CategoriasPrem != nil {
		updates["categorias_premiadas"] = *request.CategoriasPrem
	}
	if request.Preco != nil {
		updates["preco"] = *request.Preco
	}
	if request.PrecosEntries != nil {
		updates["precos_entries"] = parsePrecosEntries(*request.PrecosEntries)
	}
	if request.Patrocinado != nil {
		updates["patrocinado"] = *request.Patrocinado
	}
	if request.Percurso != nil {
		updates["percurso"] = *request.Percurso
	}
	if request.Kits != nil {
		updates["kits"] = *request.Kits
	}

	return updates, nil
}
