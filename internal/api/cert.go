package api

import (
	"encoding/json"
	"net/http"

	"github.com/PradhaID/go-nginx-proxy/internal/nginx"
)

type certRequest struct {
	Domains []string `json:"domains"`
}

func (s *Server) handleCertIssue(w http.ResponseWriter, r *http.Request, domain string) {
	if _, err := s.manager.Get(domain); err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	site := &nginx.Site{Domain: domain}
	if err := json.NewDecoder(r.Body).Decode(site); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	domains := site.AllDomains()
	out, err := s.certbot.Issue(r.Context(), domains)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	if _, err := s.service.Reload(r.Context()); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": "certificate issued but nginx reload failed: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}

func (s *Server) handleCertRenew(w http.ResponseWriter, r *http.Request, domain string) {
	site, err := s.manager.Get(domain)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	out, err := s.certbot.Renew(r.Context(), site.AllDomains())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	if _, err := s.service.Reload(r.Context()); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": "certificate renewed but nginx reload failed: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}

func (s *Server) handleCertStatus(w http.ResponseWriter, r *http.Request, domain string) {
	st := s.certbot.Status(domain)
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleCertDelete(w http.ResponseWriter, r *http.Request, domain string) {
	out, err := s.certbot.Delete(r.Context(), domain)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}
