package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[httpx] encode response: %v", err)
	}
}

func Error(w http.ResponseWriter, status int, err error) {
	resp := ErrorResponse{Error: http.StatusText(status)}
	if err != nil {
		resp.Message = err.Error()
	}
	JSON(w, status, resp)
}

func ErrNotFound(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "resource not found"
	}
	JSON(w, http.StatusNotFound, ErrorResponse{
		Error:   http.StatusText(http.StatusNotFound),
		Message: msg,
	})
}

func ErrBadRequest(w http.ResponseWriter, msg string) {
	JSON(w, http.StatusBadRequest, ErrorResponse{
		Error:   http.StatusText(http.StatusBadRequest),
		Message: msg,
	})
}

func ErrUnauthorized(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "unauthorized"
	}
	JSON(w, http.StatusUnauthorized, ErrorResponse{
		Error:   http.StatusText(http.StatusUnauthorized),
		Message: msg,
	})
}

func ErrForbidden(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "forbidden"
	}
	JSON(w, http.StatusForbidden, ErrorResponse{
		Error:   http.StatusText(http.StatusForbidden),
		Message: msg,
	})
}

func ErrConflict(w http.ResponseWriter, msg string) {
	JSON(w, http.StatusConflict, ErrorResponse{
		Error:   http.StatusText(http.StatusConflict),
		Message: msg,
	})
}

var validate = validator.New()

func BindAndValidate(r *http.Request, dest interface{}) error {
	if r.Body == nil {
		return errors.New("empty request body")
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		return err
	}
	if err := validate.Struct(dest); err != nil {
		return err
	}
	return nil
}

func ValidationErrors(err error) map[string]string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return nil
	}
	out := make(map[string]string, len(ve))
	for _, f := range ve {
		field := strings.ToLower(f.Field()[:1]) + f.Field()[1:]
		var msg string
		switch f.Tag() {
		case "required":
			msg = "required"
		case "email":
			msg = "must be a valid email address"
		case "min":
			msg = "must be at least " + f.Param() + " characters"
		case "max":
			msg = "must be at most " + f.Param() + " characters"
		case "uuid":
			msg = "must be a valid UUID"
		case "oneof":
			msg = "must be one of: " + f.Param()
		default:
			msg = f.Error()
		}
		out[field] = msg
	}
	return out
}

func ParseUUIDParam(r *http.Request, key string) (uuid.UUID, error) {
	raw := strings.TrimPrefix(r.URL.Path, "/")
	_ = raw
	fromCtx := r.Context().Value(key)
	if s, ok := fromCtx.(string); ok {
		return uuid.Parse(s)
	}
	return uuid.Nil, errors.New("param not found: " + key)
}
