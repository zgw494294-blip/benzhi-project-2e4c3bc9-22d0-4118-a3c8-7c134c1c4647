package httpapi

import (
	"net/http"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
)

func (s *Server) HandleBatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		writeError(w, errMethod())
		return
	}
	var req application.CreateBatchCommand
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	meta, err := parseMutationMetadata(r, 0, false)
	if err != nil {
		writeError(w, err)
		return
	}
	req.IdempotencyKey = meta.IdempotencyKey
	out, err := s.App.CreateBatch(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (s *Server) HandleBatchAction(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) < 3 {
		writeError(w, errNotFound())
		return
	}
	batchID := parts[2]
	if len(parts) == 3 && r.Method == "GET" {
		b, e := s.App.GetBatch(r.Context(), batchID)
		if e != nil {
			writeError(w, e)
			return
		}
		writeJSON(w, 200, b)
		return
	}
	if len(parts) < 4 {
		writeError(w, errNotFound())
		return
	}
	action := parts[3]
	switch action {
	case "pages":
		s.HandleAddPage(w, r, batchID)
	case "ocr-runs":
		s.HandleAddOCR(w, r, batchID)
	case "quality-check":
		s.HandleQuality(w, r, batchID)
	case "review":
		s.HandleReview(w, r, batchID)
	case "freeze":
		s.HandleFreeze(w, r, batchID)
	case "issues":
		if len(parts) < 6 || parts[5] != "resolve" {
			writeError(w, errNotFound())
			return
		}
		s.HandleResolve(w, r, batchID, parts[4])
	default:
		writeError(w, errNotFound())
	}
}
func errMethod() error   { return &methodError{} }
func errNotFound() error { return &notFoundError{} }

type methodError struct{}

func (*methodError) Error() string { return "method not allowed" }

type notFoundError struct{}

func (*notFoundError) Error() string { return "route not found" }
