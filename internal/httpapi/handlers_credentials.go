package httpapi

import "net/http"

func (s *Server) HandleCredential(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if r.Method != "GET" || len(parts) != 4 || parts[3] != "verify" {
		writeError(w, errNotFound())
		return
	}
	out, e := s.App.VerifyCredential(r.Context(), parts[2])
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
