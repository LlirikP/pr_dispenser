package handlers

import (
	"encoding/json"
	"net/http"
)

func RespondWithError(w http.ResponseWriter, code, msg string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": msg,
		},
	})
}
