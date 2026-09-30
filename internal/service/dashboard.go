package service

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CountItem struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type EstadoCount struct {
	Estado string `json:"estado"`
	Count  int    `json:"count"`
}

type CidadeCount struct {
	Cidade string `json:"cidade"`
	Count  int    `json:"count"`
}

type DistanciaCount struct {
	Distancia string `json:"distancia"`
	Count     int    `json:"count"`
}

type OrganizadorCount struct {
	Organizador string `json:"organizador"`
	Count       int    `json:"count"`
}

type FonteCount struct {
	Fonte string `json:"fonte"`
	Count int    `json:"count"`
}

type DensidadeItem struct {
	Data  string `json:"data"`
	Count int    `json:"count"`
}

type StatusInscricoes struct {
	Abertas    int `json:"abertas"`
	EmBreve    int `json:"emBreve"`
	Encerradas int `json:"encerradas"`
}

type ScraperHealthItem struct {
	Fonte          string  `json:"fonte"`
	Display        string  `json:"display"`
	Count          int     `json:"count"`
	SemLink        int     `json:"semLink"`
	SemRegulamento int     `json:"semRegulamento"`
	SemImagem      int     `json:"semImagem"`
	SemPreco       int     `json:"semPreco"`
	MaxDataColeta  *string `json:"maxDataColeta"`
}

type ProximoEventoItem struct {
	ID              string   `json:"_id"`
	NomeEvento      string   `json:"nome_evento"`
	DataRealizacao  string   `json:"data_realizacao"`
	DatasRealizacao []string `json:"datas_realizacao"`
	Cidade          string   `json:"cidade"`
	Estado          string   `json:"estado"`
	Organizador     string   `json:"organizador"`
}

type DashboardStats struct {
	Total            int                 `json:"total"`
	Ativos           int                 `json:"ativos"`
	Passados         int                 `json:"passados"`
	Proximos30d      int                 `json:"proximos30d"`
	Proximos90d      int                 `json:"proximos90d"`
	SemPreco         int                 `json:"semPreco"`
	Patrocinados     int                 `json:"patrocinados"`
	SemImagem        int                 `json:"semImagem"`
	SemLink          int                 `json:"semLink"`
	SemRegulamento   int                 `json:"semRegulamento"`
	ValorMedio       float64             `json:"valorMedio"`
	Lote1Count       int                 `json:"lote1Count"`
	PorMes           []CountItem         `json:"porMes"`
	PorEstado        []EstadoCount       `json:"porEstado"`
	PorCidade        []CidadeCount       `json:"porCidade"`
	PorDistancia     []DistanciaCount    `json:"porDistancia"`
	PorOrganizador   []OrganizadorCount  `json:"porOrganizador"`
	PorFonte         []FonteCount        `json:"porFonte"`
	Densidade        []DensidadeItem     `json:"densidade"`
	Choques          int                 `json:"choques"`
	StatusInscricoes StatusInscricoes    `json:"statusInscricoes"`
	ComPercurso      int                 `json:"comPercurso"`
	ComKits          int                 `json:"comKits"`
	PorHorario       []CountItem         `json:"porHorario"`
	PorKit           []CountItem         `json:"porKit"`
	ScraperHealth    []ScraperHealthItem `json:"scraperHealth"`
	ProximosEventos  []ProximoEventoItem `json:"proximosEventos"`
}

var mesesPortugues = map[string]int{
	"janeiro": 1, "fevereiro": 2, "marco": 3, "março": 3,
	"abril": 4, "maio": 5, "junho": 6, "julho": 7,
	"agosto": 8, "setembro": 9, "outubro": 10,
	"novembro": 11, "dezembro": 12,
}

var precoValorPattern = regexp.MustCompile(`R\$\s*([0-9.,]+)`)
var distanciaPattern = regexp.MustCompile(`(?i)\d+(?:[.,]\d+)?\s*KM`)
var horarioPatternFull = regexp.MustCompile(`^\d{2}:\d{2}$`)

func truncateToDay(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func parseDataRealizacao(raw string, datas []time.Time) *time.Time {
	if len(datas) > 0 {
		day := truncateToDay(datas[0])
		return &day
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		if parsed, err := time.Parse(time.RFC3339, strings.Replace(trimmed, "Z", "+00:00", 1)); err == nil && parsed.Year() > 1900 {
			day := truncateToDay(parsed)
			return &day
		}
		if parsed, err := time.Parse("2006-01-02", trimmed); err == nil && parsed.Year() > 1900 {
			day := truncateToDay(parsed)
			return &day
		}
		parts := strings.Fields(strings.ToLower(trimmed))
		if len(parts) >= 5 {
			if day, err := strconv.Atoi(parts[0]); err == nil {
				if month, ok := mesesPortugues[parts[2]]; ok {
					if year, err := strconv.Atoi(parts[4]); err == nil && day >= 1 && year > 1900 {
						parsed := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
						return &parsed
					}
				}
			}
		}
	}
	return nil
}

func parsePrecoValor(text string) (float64, bool) {
	match := precoValorPattern.FindStringSubmatch(text)
	if match == nil {
		return 0, false
	}
	cleaned := match[1]
	if strings.Contains(cleaned, ".") && strings.Contains(cleaned, ",") {
		cleaned = strings.ReplaceAll(cleaned, ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	} else if strings.Contains(cleaned, ".") {
		parts := strings.Split(cleaned, ".")
		if len(parts[len(parts)-1]) != 2 {
			cleaned = strings.ReplaceAll(cleaned, ".", "")
		}
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	} else {
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func normalizeDistanciaLabel(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > 30 {
		return "", false
	}
	if strings.Contains(trimmed, "Art.") || strings.Contains(trimmed, "CAPÍTULO") {
		return "", false
	}
	matches := distanciaPattern.FindAllString(trimmed, -1)
	if len(matches) == 0 {
		return "", false
	}
	normalized := strings.ToUpper(strings.ReplaceAll(matches[0], " ", ""))
	normalized = strings.ReplaceAll(normalized, "\t", "")
	normalized = strings.ReplaceAll(normalized, ",", ".")
	if len(normalized) > 10 {
		return "", false
	}
	return normalized, true
}

func capitalizeWords(text string) string {
	words := strings.Fields(strings.ToLower(text))
	for index, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
			words[index] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

func isValidLocalLargada(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 10 || len(trimmed) > 120 {
		return false
	}
	if strings.Contains(trimmed, "Art.") || strings.Contains(trimmed, "CAPÍTULO") {
		return false
	}
	lowered := strings.ToLower(trimmed)
	if strings.HasPrefix(lowered, "com ") || strings.HasPrefix(lowered, "para ") || strings.HasPrefix(lowered, "com pelo") {
		return false
	}
	if len(strings.Fields(trimmed)) < 2 {
		return false
	}
	return true
}

type counter struct {
	counts map[string]int
	order  []string
}

func newCounter() *counter {
	return &counter{counts: map[string]int{}}
}

func (c *counter) add(key string) {
	if _, exists := c.counts[key]; !exists {
		c.order = append(c.order, key)
	}
	c.counts[key]++
}

func (c *counter) sorted() []string {
	keys := append([]string{}, c.order...)
	sort.SliceStable(keys, func(i, j int) bool {
		return c.counts[keys[i]] > c.counts[keys[j]]
	})
	return keys
}

func ComputeDashboardStats(eventos []Event, now time.Time) DashboardStats {
	today := truncateToDay(now)
	in30Days := today.AddDate(0, 0, 30)
	in90Days := today.AddDate(0, 0, 90)
	in7Days := today.AddDate(0, 0, 7)

	stats := DashboardStats{
		PorMes:          []CountItem{},
		PorEstado:       []EstadoCount{},
		PorCidade:       []CidadeCount{},
		PorDistancia:    []DistanciaCount{},
		PorOrganizador:  []OrganizadorCount{},
		PorFonte:        []FonteCount{},
		Densidade:       []DensidadeItem{},
		PorHorario:      []CountItem{},
		PorKit:          []CountItem{},
		ScraperHealth:   []ScraperHealthItem{},
		ProximosEventos: []ProximoEventoItem{},
	}

	porMes := newCounter()
	porEstado := newCounter()
	porCidade := newCounter()
	porDistancia := newCounter()
	porOrganizador := newCounter()
	porFonte := newCounter()
	densidade := newCounter()
	porHorario := newCounter()
	porKit := newCounter()
	cidadeDisplay := map[string]string{}
	organizadorDisplay := map[string]string{}
	fonteDisplay := map[string]string{}

	type healthAggregate struct {
		display        string
		count          int
		semLink        int
		semRegulamento int
		semImagem      int
		semPreco       int
		maxDataColeta  *string
	}
	healthMap := map[string]*healthAggregate{}
	healthOrder := []string{}

	var precos []float64
	var candidatos []ProximoEventoItem
	var datasOrdenacao []time.Time

	for _, evento := range eventos {
		stats.Total++

		eventDate := parseDataRealizacao(evento.DataRealizacao, evento.DatasRealizacao)

		var eventDay *time.Time
		if eventDate != nil {
			day := truncateToDay(*eventDate)
			eventDay = &day
			monthKey := day.Format("2006-01")
			porMes.add(monthKey)
			densidade.add(day.Format("2006-01-02"))

			if !day.Before(today) {
				stats.Ativos++
				if !day.After(in30Days) {
					stats.Proximos30d++
				}
				if !day.After(in90Days) {
					stats.Proximos90d++
				}
			} else {
				stats.Passados++
			}

			if day.Before(today) {
				stats.StatusInscricoes.Encerradas++
			} else if !day.After(in7Days) {
				stats.StatusInscricoes.EmBreve++
			} else {
				stats.StatusInscricoes.Abertas++
			}
		} else {
			stats.StatusInscricoes.Encerradas++
		}

		if len(evento.PrecosEntries) == 0 {
			stats.SemPreco++
		} else {
			for _, entry := range evento.PrecosEntries {
				if text, ok := entry.(string); ok && strings.Contains(strings.ToLower(text), "lote 1") {
					stats.Lote1Count++
					break
				}
			}
			for _, entry := range evento.PrecosEntries {
				if text, ok := entry.(string); ok {
					if value, found := parsePrecoValor(text); found {
						precos = append(precos, value)
					}
				}
			}
		}

		if evento.Horario != nil {
			horario := strings.TrimSpace(*evento.Horario)
			if horarioPatternFull.MatchString(horario) {
				porHorario.add(horario)
			}
		}

		if evento.Percurso != nil {
			local := strings.TrimSpace(evento.Percurso.LocalLargada)
			trajeto := ""
			if evento.Percurso.Trajeto != nil {
				trajeto = strings.TrimSpace(*evento.Percurso.Trajeto)
			}
			valid := false
			if local != "" && isValidLocalLargada(local) {
				valid = true
			}
			if trajeto != "" && len(trajeto) <= 500 &&
				!strings.Contains(trajeto, "Art.") && !strings.Contains(trajeto, "CAPÍTULO") {
				valid = true
			}
			if valid {
				stats.ComPercurso++
			}
		}

		if len(evento.Kits) > 0 {
			validKits := 0
			for _, kit := range evento.Kits {
				nome := strings.TrimSpace(kit.Nome)
				if nome == "" || len(nome) > 50 || strings.Contains(nome, "Art.") {
					continue
				}
				validKits++
				porKit.add(nome)
			}
			if validKits > 0 {
				stats.ComKits++
			}
		}

		if evento.Patrocinado {
			stats.Patrocinados++
		}
		hasImagem := evento.URLImagem != nil && strings.TrimSpace(*evento.URLImagem) != ""
		if !hasImagem {
			stats.SemImagem++
		}
		hasLink := evento.URLInscricao != nil && strings.TrimSpace(*evento.URLInscricao) != ""
		if !hasLink {
			stats.SemLink++
		}
		linkEdital := ""
		if evento.LinkEdital != nil {
			linkEdital = *evento.LinkEdital
		}
		hasRegulamento := linkEdital != "" && linkEdital != "edital não encontrado"
		if !hasRegulamento {
			stats.SemRegulamento++
		}

		estado := strings.ToUpper(strings.TrimSpace(evento.Estado))
		if estado == "" {
			estado = "—"
		}
		porEstado.add(estado)

		cidadeRaw := strings.TrimSpace(evento.Cidade)
		if cidadeRaw == "" {
			cidadeRaw = "—"
		}
		cidadeKey := strings.ToLower(cidadeRaw)
		if _, exists := cidadeDisplay[cidadeKey]; !exists {
			cidadeDisplay[cidadeKey] = capitalizeWords(cidadeRaw)
		}
		porCidade.add(cidadeKey)

		for _, distancia := range evento.Distancias {
			if normalized, ok := normalizeDistanciaLabel(distancia); ok {
				porDistancia.add(normalized)
			}
		}

		organizadorRaw := strings.TrimSpace(evento.Organizador)
		if organizadorRaw == "" {
			organizadorRaw = "—"
		}
		organizadorKey := strings.ToLower(organizadorRaw)
		if _, exists := organizadorDisplay[organizadorKey]; !exists {
			organizadorDisplay[organizadorKey] = organizadorRaw
		}
		porOrganizador.add(organizadorKey)

		fonteRaw := strings.TrimSpace(evento.SiteColeta)
		if fonteRaw == "" {
			fonteRaw = "—"
		}
		fonteKey := strings.ToLower(fonteRaw)
		if _, exists := fonteDisplay[fonteKey]; !exists {
			fonteDisplay[fonteKey] = fonteRaw
		}
		porFonte.add(fonteKey)

		if fonteRaw != "—" && !strings.Contains(fonteKey, "manual") && !strings.Contains(fonteKey, "ticketsports") {
			semLinkFlag := 0
			if !hasLink {
				semLinkFlag = 1
			}
			semRegulamentoFlag := 0
			if !hasRegulamento {
				semRegulamentoFlag = 1
			}
			semImagemFlag := 0
			if !hasImagem {
				semImagemFlag = 1
			}
			semPrecoFlag := 0
			if len(evento.PrecosEntries) == 0 {
				semPrecoFlag = 1
			}
			var dataColeta *string
			if evento.DataColeta != nil {
				formatted := evento.DataColeta.Format(time.RFC3339)
				dataColeta = &formatted
			}
			aggregate, exists := healthMap[fonteKey]
			if !exists {
				aggregate = &healthAggregate{display: fonteRaw}
				healthMap[fonteKey] = aggregate
				healthOrder = append(healthOrder, fonteKey)
			}
			aggregate.count++
			aggregate.semLink += semLinkFlag
			aggregate.semRegulamento += semRegulamentoFlag
			aggregate.semImagem += semImagemFlag
			aggregate.semPreco += semPrecoFlag
			if dataColeta != nil && (aggregate.maxDataColeta == nil || *dataColeta > *aggregate.maxDataColeta) {
				aggregate.maxDataColeta = dataColeta
			}
		}

		if eventDay != nil && !eventDay.Before(today) && !eventDay.After(in30Days) {
			datasISO := make([]string, 0, len(evento.DatasRealizacao))
			for _, data := range evento.DatasRealizacao {
				datasISO = append(datasISO, data.Format(time.RFC3339))
			}
			candidatos = append(candidatos, ProximoEventoItem{
				ID:              evento.ID,
				NomeEvento:      evento.NomeEvento,
				DataRealizacao:  evento.DataRealizacao,
				DatasRealizacao: datasISO,
				Cidade:          evento.Cidade,
				Estado:          evento.Estado,
				Organizador:     evento.Organizador,
			})
			datasOrdenacao = append(datasOrdenacao, *eventDay)
		}
	}

	if len(precos) > 0 {
		sum := 0.0
		for _, value := range precos {
			sum += value
		}
		average := sum / float64(len(precos))
		stats.ValorMedio = float64(int(average*100+0.5)) / 100
	}

	for _, key := range sortedKeys(porMes.counts) {
		stats.PorMes = append(stats.PorMes, CountItem{Label: key, Count: porMes.counts[key]})
	}
	for _, key := range porEstado.sorted() {
		stats.PorEstado = append(stats.PorEstado, EstadoCount{Estado: key, Count: porEstado.counts[key]})
	}
	for _, key := range porCidade.sorted() {
		stats.PorCidade = append(stats.PorCidade, CidadeCount{Cidade: cidadeDisplay[key], Count: porCidade.counts[key]})
	}
	for _, key := range porDistancia.sorted() {
		stats.PorDistancia = append(stats.PorDistancia, DistanciaCount{Distancia: key, Count: porDistancia.counts[key]})
	}
	for _, key := range porOrganizador.sorted() {
		stats.PorOrganizador = append(stats.PorOrganizador, OrganizadorCount{Organizador: organizadorDisplay[key], Count: porOrganizador.counts[key]})
	}
	for _, key := range porFonte.sorted() {
		stats.PorFonte = append(stats.PorFonte, FonteCount{Fonte: fonteDisplay[key], Count: porFonte.counts[key]})
	}
	for _, key := range sortedKeys(densidade.counts) {
		stats.Densidade = append(stats.Densidade, DensidadeItem{Data: key, Count: densidade.counts[key]})
		if densidade.counts[key] > 1 {
			stats.Choques++
		}
	}
	for _, key := range porHorario.sorted() {
		stats.PorHorario = append(stats.PorHorario, CountItem{Label: key, Count: porHorario.counts[key]})
	}
	for _, key := range porKit.sorted() {
		stats.PorKit = append(stats.PorKit, CountItem{Label: key, Count: porKit.counts[key]})
	}

	sort.SliceStable(healthOrder, func(i, j int) bool {
		left := healthMap[healthOrder[i]].maxDataColeta
		right := healthMap[healthOrder[j]].maxDataColeta
		leftValue := ""
		if left != nil {
			leftValue = *left
		}
		rightValue := ""
		if right != nil {
			rightValue = *right
		}
		return leftValue > rightValue
	})
	for _, key := range healthOrder {
		aggregate := healthMap[key]
		stats.ScraperHealth = append(stats.ScraperHealth, ScraperHealthItem{
			Fonte:          key,
			Display:        aggregate.display,
			Count:          aggregate.count,
			SemLink:        aggregate.semLink,
			SemRegulamento: aggregate.semRegulamento,
			SemImagem:      aggregate.semImagem,
			SemPreco:       aggregate.semPreco,
			MaxDataColeta:  aggregate.maxDataColeta,
		})
	}

	indices := make([]int, len(candidatos))
	for index := range indices {
		indices[index] = index
	}
	sort.SliceStable(indices, func(i, j int) bool {
		return datasOrdenacao[indices[i]].Before(datasOrdenacao[indices[j]])
	})
	for position, index := range indices {
		if position >= 5 {
			break
		}
		stats.ProximosEventos = append(stats.ProximosEventos, candidatos[index])
	}

	return stats
}

func sortedKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
