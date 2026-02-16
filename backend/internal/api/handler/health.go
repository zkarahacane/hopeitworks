package handler

import (
	"encoding/json"
	"net/http"
)

// HandleHealthz returns HTTP 200 with {"status":"ok"}.
func HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
