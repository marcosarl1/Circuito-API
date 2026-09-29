package service

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func normalizeDistancias(value any) []string {
	switch typedValue := value.(type) {
	case string:
		if strings.TrimSpace(typedValue) == "" {
			return []string{}
		}
		parts := strings.Split(typedValue, ",")
		distancias := make([]string, 0, len(parts))

		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				distancias = append(distancias, trimmed)
			}
		}
		return distancias
	case []string:
		distancias := make([]string, 0, len(typedValue))

		for _, item := range typedValue {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				distancias = append(distancias, trimmed)
			}
		}
		return distancias

	case []any:
		distancias := make([]string, 0, len(typedValue))
		for _, item := range typedValue {
			trimmed := strings.TrimSpace(toLegacyString(item))
			if trimmed != "" {
				distancias = append(distancias, trimmed)
			}
		}
		return distancias

	default:
		return []string{}
	}
}

func toLegacyString(value any) string {
	switch typedValue := value.(type) {
	case string:
		return typedValue
	case int:
		return strconv.Itoa(typedValue)
	case int32:
		return strconv.FormatInt(int64(typedValue), 10)
	case int64:
		return strconv.FormatInt(typedValue, 10)
	case float64:
		return strconv.FormatFloat(typedValue, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typedValue)
	default:
		return ""
	}
}

func parsePrecosEntries(value any) []any {
	switch typedValue := value.(type) {
	case nil:
		return []any{}
	case string:
		trimmed := strings.TrimSpace(typedValue)
		if trimmed == "" {
			return []any{}
		}

		var parsed any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return []any{}
		}

		return normalizePrecosParsed(parsed)

	case []any:
		return typedValue

	case []string:
		entries := make([]any, 0, len(typedValue))
		for _, item := range typedValue {
			entries = append(entries, item)
		}
		return entries

	default:
		return []any{typedValue}
	}
}

func normalizePrecosParsed(parsed any) []any {
	switch typedValue := parsed.(type) {
	case []any:
		return typedValue
	case string:
		return []any{typedValue}
	case nil:
		return []any{}
	default:
		return []any{typedValue}
	}
}

var horarioPattern = regexp.MustCompile(`^\d{2}:\d{2}$`)

func validateHorario(value *string) error {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	if !horarioPattern.MatchString(trimmed) {
		return errors.New("horario must match HH:MM format")
	}

	_, err := time.Parse("15:04", trimmed)
	if err != nil {
		return err
	}

	*value = trimmed

	return nil
}
