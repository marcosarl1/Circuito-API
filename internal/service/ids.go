package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var idSuffix = regexp.MustCompile(`^\d{4}$`)

func NormalizeEventID(v any) (string, error) {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return "", fmt.Errorf("empty id")
		}
		return s, nil
	case int:
		return fmt.Sprintf("%d", t), nil
	case int32:
		return fmt.Sprintf("%d", t), nil
	case int64:
		return fmt.Sprintf("%d", t), nil
	case float64:
		return fmt.Sprintf("%.0f", t), nil
	case fmt.Stringer:
		s := strings.TrimSpace(t.String())
		if s == "" {
			return "", fmt.Errorf("empty id")
		}
		return s, nil
	default:
		return "", fmt.Errorf("unsupported id type %T", v)
	}
}

func EventPrefix(t time.Time) string {
	return t.UTC().Format("200601")
}

func BuildEventID(prefix string, seq int64) (string, error) {
	if len(prefix) != 6 {
		return "", fmt.Errorf("invalid prefix %q", prefix)
	}
	if seq < 1 || seq > 9999 {
		return "", fmt.Errorf("seq out of range: %d", seq)
	}
	if !idSuffix.MatchString(fmt.Sprintf("%04d", seq)) {
		return "", fmt.Errorf("invalid seq: %d", seq)
	}
	return fmt.Sprintf("%s%04d", prefix, seq), nil
}
