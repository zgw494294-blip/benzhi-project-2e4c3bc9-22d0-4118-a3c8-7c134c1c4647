package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

var decodedBodyCache sync.Map

type Server struct {
	App *application.Service
	Mux *http.ServeMux
}

func New(app *application.Service) *Server {
	s := &Server{App: app, Mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) Handler() http.Handler { return withRequestID(s.Mux) }
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", requestID(r))
		next.ServeHTTP(w, r)
	})
}
func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return "request-local"
}
func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return domain.Invalid("读取请求体失败", "")
	}
	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" {
		cached, _ := decodedBodyCache.LoadOrStore(key, append([]byte(nil), raw...))
		raw = cached.([]byte)
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return domain.Invalid("请求体不是合法 JSON", "")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	var method *methodError
	var route *notFoundError
	if errors.As(err, &method) {
		status = http.StatusMethodNotAllowed
	}
	if errors.As(err, &route) {
		status = http.StatusNotFound
	}
	var de *domain.DomainError
	if errors.As(err, &de) {
		switch de.Code {
		case domain.ErrInvalid:
			status = http.StatusBadRequest
		case domain.ErrNotFound:
			status = http.StatusNotFound
		case domain.ErrConflict, domain.ErrVersion:
			status = http.StatusConflict
		case domain.ErrState, domain.ErrQuality:
			status = http.StatusUnprocessableEntity
		}
	}
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": codeOf(err), "message": err.Error()}})
}
func codeOf(err error) string {
	var method *methodError
	var route *notFoundError
	if errors.As(err, &method) {
		return "method_not_allowed"
	}
	if errors.As(err, &route) {
		return "route_not_found"
	}
	var de *domain.DomainError
	if errors.As(err, &de) {
		return string(de.Code)
	}
	return "internal"
}
func pathParts(path string) []string { return strings.Split(strings.Trim(path, "/"), "/") }
