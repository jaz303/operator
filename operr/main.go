package operr

import (
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrInputMappingFailed = errors.New("input mapping failed")
	ErrOperationFailed    = errors.New("operation failed")
)

func DefaultErrorMapper(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]any{
		"error": err,
	})
}
