package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func extractInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func writeCachedJSON(w http.ResponseWriter, status int, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(payload)
}

func encodeJSON(payload interface{}) ([]byte, error) {
	return json.Marshal(payload)
}

func parseDurationParam(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("invalid duration")
	}
	if v < 15 || v > 240 || v%15 != 0 {
		return 0, errors.New("invalid duration")
	}
	return v, nil
}

func normalizeStringList(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		clean := strings.TrimSpace(item)
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}

	if len(out) == 0 {
		return []string{}
	}
	return out
}
