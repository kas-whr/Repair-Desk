package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

const maxBodyBytes = 1 << 20

// ErrorBody is the single error format of the API:
// {"error": {"code": "...", "message": "...", "details": {...}}}
type ErrorBody struct {
	Error ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type listResponse[T any] struct {
	Items []T `json:"items"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func writeList[T any](w http.ResponseWriter, items []T) {
	if items == nil {
		items = []T{}
	}
	writeJSON(w, http.StatusOK, listResponse[T]{Items: items})
}

func writeErrorPayload(w http.ResponseWriter, status int, p ErrorPayload) {
	writeJSON(w, status, ErrorBody{Error: p})
}

// writeError maps domain errors to HTTP statuses; everything else is a 500.
func writeError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	if de, ok := domain.AsError(err); ok {
		writeErrorPayload(w, statusFor(de.Kind), ErrorPayload{Code: de.Code, Message: de.Message, Details: de.Details})
		return
	}
	logger.ErrorContext(r.Context(), "internal error", "error", err, "method", r.Method, "path", r.URL.Path)
	writeErrorPayload(w, http.StatusInternalServerError, ErrorPayload{Code: "INTERNAL_ERROR", Message: "internal server error"})
}

func statusFor(k domain.Kind) int {
	switch k {
	case domain.KindValidation:
		return http.StatusBadRequest
	case domain.KindNotFound:
		return http.StatusNotFound
	case domain.KindConflict:
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func errInvalidJSON(msg string) error {
	return &domain.Error{Kind: domain.KindValidation, Code: "INVALID_JSON", Message: msg}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var typeErr *json.UnmarshalTypeError
		var maxErr *http.MaxBytesError
		switch {
		case errors.Is(err, io.EOF):
			return errInvalidJSON("request body is empty")
		case errors.As(err, &typeErr):
			return domain.NewValidationError(map[string]string{typeErr.Field: fmt.Sprintf("must be of type %s", typeErr.Type)})
		case errors.As(err, &maxErr):
			return errInvalidJSON("request body is too large")
		default:
			return errInvalidJSON("malformed JSON: " + err.Error())
		}
	}
	if dec.More() {
		return errInvalidJSON("request body must contain a single JSON object")
	}
	return nil
}

func pathID(r *http.Request, name string) (string, error) {
	id := r.PathValue(name)
	if !domain.IsValidID(id) {
		return "", domain.NewValidationError(map[string]string{name: "must be a valid UUID"})
	}
	return id, nil
}
