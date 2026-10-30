// Package httpx holds the shared HTTP response contract: the pinned error
// envelope (D17/D28) and strict JSON body decoding.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type errorEnvelope struct {
	Error errDetail `json:"error"`
}

type errDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes v as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes the standard error envelope.
func Error(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{Error: errDetail{Code: code, Message: message}})
}

// ErrBodyTooLarge is reported when the request body exceeds the size limit.
var ErrBodyTooLarge = errors.New("request body too large")

// DecodeJSON decodes a strict, size-limited JSON body into dst. On failure it
// writes a 400 envelope (D28: malformed JSON) and returns false.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB is generous for this API
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			Error(w, 400, "BAD_REQUEST", "Request body is too large")
			return false
		}
		Error(w, 400, "BAD_REQUEST", "Request body must be valid JSON")
		return false
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		Error(w, 400, "BAD_REQUEST", "Request body must contain a single JSON value")
		return false
	}
	return true
}
