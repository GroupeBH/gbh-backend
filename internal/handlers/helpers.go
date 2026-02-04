package handlers

import (
	"encoding/json"
	"net/http"
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
