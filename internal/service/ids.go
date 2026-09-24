package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var idSuffix = regexp.MustCompile(`^\d{4}$`)

func NormalizeEventID(rawValue any) (string, error) {
	switch typedValue := rawValue.(type) {
	case string:
		trimmedID := strings.TrimSpace(typedValue)
		if trimmedID == "" {
			return "", fmt.Errorf("empty id")
		}
		return trimmedID, nil
	case int:
		return fmt.Sprintf("%d", typedValue), nil
	case int32:
		return fmt.Sprintf("%d", typedValue), nil
	case int64:
		return fmt.Sprintf("%d", typedValue), nil
	case float64:
		return fmt.Sprintf("%.0f", typedValue), nil
	case fmt.Stringer:
		trimmedID := strings.TrimSpace(typedValue.String())
		if trimmedID == "" {
			return "", fmt.Errorf("empty id")
		}
		return trimmedID, nil
	default:
		return "", fmt.Errorf("unsupported id type %T", rawValue)
	}
}

func EventPrefix(referenceTime time.Time) string {
	return referenceTime.UTC().Format("200601")
}

func BuildEventID(prefix string, sequence int64) (string, error) {
	if len(prefix) != 6 {
		return "", fmt.Errorf("invalid prefix %q", prefix)
	}
	if sequence < 1 || sequence > 9999 {
		return "", fmt.Errorf("seq out of range: %d", sequence)
	}
	if !idSuffix.MatchString(fmt.Sprintf("%04d", sequence)) {
		return "", fmt.Errorf("invalid seq: %d", sequence)
	}
	return fmt.Sprintf("%s%04d", prefix, sequence), nil
}
