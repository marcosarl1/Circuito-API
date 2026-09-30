package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var monthNames = [...]string{
	"",
	"Janeiro",
	"Fevereiro",
	"Março",
	"Abril",
	"Maio",
	"Junho",
	"Julho",
	"Agosto",
	"Setembro",
	"Outubro",
	"Novembro",
	"Dezembro",
}

var pricePattern = regexp.MustCompile(`R\$\s*[\d.,]+`)

func formatDataRealizacao(value time.Time) string {
	return fmt.Sprintf(
		"%02d de %s de %d",
		value.Day(),
		monthNames[value.Month()],
		value.Year(),
	)
}

func splitCategorias(value string) []string {
	categorias := make([]string, 0, 2)

	for _, item := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			categorias = append(categorias, trimmed)
		}
	}

	return categorias
}

func extractPriceFromText(text string) string {
	match := pricePattern.FindString(text)
	if match == "" {
		return text
	}

	pricePart := strings.TrimSpace(match)
	labelPart := strings.TrimSpace(strings.ReplaceAll(text, match, ""))
	labelPart = strings.Trim(labelPart, " -—|")

	if labelPart == "" {
		return pricePart
	}

	return fmt.Sprintf("%s — %s", strings.ToUpper(labelPart), pricePart)
}

func formatBRL(value float64) string {
	formatted := fmt.Sprintf("%.2f", value)

	parts := strings.SplitN(formatted, ".", 2)
	integerPart := parts[0]
	decimalPart := "00"

	if len(parts) == 2 {
		decimalPart = parts[1]
	}

	negative := strings.HasPrefix(integerPart, "-")
	integerPart = strings.TrimPrefix(integerPart, "-")

	var builder strings.Builder

	for index, digit := range integerPart {
		if index > 0 && (len(integerPart)-index)%3 == 0 {
			builder.WriteByte('.')
		}
		builder.WriteRune(digit)
	}

	result := "R$ " + builder.String() + "," + decimalPart
	if negative {
		return "-" + result
	}

	return result
}

func formatPriceList(entries []any, rawPrice *string) []string {
	if len(entries) == 0 {
		return formatPriceFromRaw(rawPrice)
	}

	priceList := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, entry := range entries {
		var formatted string

		switch typedEntry := entry.(type) {
		case string:
			formatted = formatTextEntry(typedEntry)
		case map[string]any:
			formatted = formatMapEntry(typedEntry)
		default:
			formatted = strings.TrimSpace(fmt.Sprintf("%v", typedEntry))
		}

		if formatted == "" {
			continue
		}

		if _, exists := seen[formatted]; exists {
			continue
		}

		seen[formatted] = struct{}{}
		priceList = append(priceList, formatted)
	}

	if len(priceList) == 0 {
		return formatPriceFromRaw(rawPrice)
	}

	return priceList
}

func formatPriceFromRaw(rawPrice *string) []string {
	if rawPrice == nil {
		return []string{}
	}

	priceList := make([]string, 0, 2)

	for _, part := range strings.Split(*rawPrice, ";") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			priceList = append(priceList, trimmed)
		}
	}

	return priceList
}

func formatTextEntry(text string) string {
	trimmed := strings.TrimSpace(text)

	if trimmed == "" {
		return ""
	}

	if strings.Contains(trimmed, "|") {
		parts := strings.SplitN(trimmed, "|", 2)
		if len(parts) == 2 {
			pricePart := strings.TrimSpace(parts[0])
			labelPart := strings.TrimSpace(parts[1])

			if labelPart == "" {
				return pricePart
			}

			return fmt.Sprintf(
				"%s — %s",
				strings.ToUpper(labelPart),
				pricePart,
			)
		}
	}

	return extractPriceFromText(trimmed)
}

func formatMapEntry(entry map[string]any) string {
	return ""
}
