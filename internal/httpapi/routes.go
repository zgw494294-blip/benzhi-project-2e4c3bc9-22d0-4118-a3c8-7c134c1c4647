package httpapi

import "net/http"

func (s *Server) routes() {
	s.Mux.HandleFunc("/v1/batches", s.HandleBatches)
	s.Mux.HandleFunc("/v1/batches/", s.HandleBatchAction)
	s.Mux.HandleFunc("/v1/credentials/", s.HandleCredential)
	s.Mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
}
