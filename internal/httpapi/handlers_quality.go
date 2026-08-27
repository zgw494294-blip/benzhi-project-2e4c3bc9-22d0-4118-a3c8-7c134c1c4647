package httpapi

import (
	"net/http"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
)

func (s *Server) HandleAddPage(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeError(w, errMethod())
		return
	}
	var c application.AddPageCommand
	if e := decode(r, &c); e != nil {
		writeError(w, e)
		return
	}
	meta, e := parseMutationMetadata(r, c.ExpectedVersion, true)
	if e != nil {
		writeError(w, e)
		return
	}
	c.BatchID = id
	c.ExpectedVersion = meta.ExpectedVersion
	c.IdempotencyKey = meta.IdempotencyKey
	out, e := s.App.AddPage(r.Context(), c)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (s *Server) HandleAddOCR(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeError(w, errMethod())
		return
	}
	var c application.AddOCRCommand
	if e := decode(r, &c); e != nil {
		writeError(w, e)
		return
	}
	meta, e := parseMutationMetadata(r, c.ExpectedVersion, true)
	if e != nil {
		writeError(w, e)
		return
	}
	c.BatchID = id
	c.ExpectedVersion = meta.ExpectedVersion
	c.IdempotencyKey = meta.IdempotencyKey
	out, e := s.App.AddOCR(r.Context(), c)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (s *Server) HandleQuality(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeError(w, errMethod())
		return
	}
	var c application.QualityCommand
	if r.ContentLength != 0 {
		if e := decode(r, &c); e != nil {
			writeError(w, e)
			return
		}
	}
	meta, e := parseMutationMetadata(r, c.ExpectedVersion, true)
	if e != nil {
		writeError(w, e)
		return
	}
	c.BatchID = id
	c.ExpectedVersion = meta.ExpectedVersion
	c.IdempotencyKey = meta.IdempotencyKey
	out, e := s.App.QualityCheck(r.Context(), c)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (s *Server) HandleResolve(w http.ResponseWriter, r *http.Request, batchID, issueID string) {
	if r.Method != "POST" {
		writeError(w, errMethod())
		return
	}
	var c application.ResolveCommand
	if e := decode(r, &c); e != nil {
		writeError(w, e)
		return
	}
	meta, e := parseMutationMetadata(r, c.ExpectedVersion, true)
	if e != nil {
		writeError(w, e)
		return
	}
	c.BatchID = batchID
	c.IssueID = issueID
	c.ExpectedVersion = meta.ExpectedVersion
	c.IdempotencyKey = meta.IdempotencyKey
	out, e := s.App.ResolveIssue(r.Context(), c)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (s *Server) HandleReview(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeError(w, errMethod())
		return
	}
	var c application.ReviewCommand
	if e := decode(r, &c); e != nil {
		writeError(w, e)
		return
	}
	meta, e := parseMutationMetadata(r, c.ExpectedVersion, true)
	if e != nil {
		writeError(w, e)
		return
	}
	c.BatchID = id
	c.ExpectedVersion = meta.ExpectedVersion
	c.IdempotencyKey = meta.IdempotencyKey
	out, e := s.App.Review(r.Context(), c)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (s *Server) HandleFreeze(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeError(w, errMethod())
		return
	}
	var c application.FreezeCommand
	if e := decode(r, &c); e != nil {
		writeError(w, e)
		return
	}
	meta, e := parseMutationMetadata(r, c.ExpectedVersion, true)
	if e != nil {
		writeError(w, e)
		return
	}
	c.BatchID = id
	c.ExpectedVersion = meta.ExpectedVersion
	c.IdempotencyKey = meta.IdempotencyKey
	out, e := s.App.Freeze(r.Context(), c)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
