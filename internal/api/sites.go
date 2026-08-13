package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/PradhaID/go-nginx-proxy/internal/nginx"
)

func (s *Server) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleSitesList(w, r)
	case http.MethodPost:
		s.handleSiteCreate(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSitesList(w http.ResponseWriter, r *http.Request) {
	sites, err := s.manager.List()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if sites == nil {
		sites = []*nginx.Site{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": sites})
}

func (s *Server) handleSiteCreate(w http.ResponseWriter, r *http.Request) {
	var site nginx.Site
	if err := decodeBody(r, &site); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := s.manager.Create(&site); err != nil {
		if errors.Is(err, nginx.ErrExists) {
			writeErr(w, http.StatusConflict, err)
			return
		}
		if errors.Is(err, nginx.ErrInvalidDomain) {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, site)
}

func (s *Server) handleSiteGet(w http.ResponseWriter, r *http.Request, domain string) {
	site, err := s.manager.Get(domain)
	if err != nil {
		if errors.Is(err, nginx.ErrNotFound) {
			writeErr(w, http.StatusNotFound, err)
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, site)
}

func (s *Server) handleSiteUpdate(w http.ResponseWriter, r *http.Request, domain string) {
	var site nginx.Site
	if err := decodeBody(r, &site); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := s.manager.Update(domain, &site); err != nil {
		if errors.Is(err, nginx.ErrNotFound) {
			writeErr(w, http.StatusNotFound, err)
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, site)
}

func (s *Server) handleSiteDelete(w http.ResponseWriter, r *http.Request, domain string) {
	if err := s.manager.Delete(domain); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "domain": domain})
}

func (s *Server) handleSiteEnable(w http.ResponseWriter, r *http.Request, domain string) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if err := s.manager.Enable(domain); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled", "domain": domain})
}

func (s *Server) handleSiteDisable(w http.ResponseWriter, r *http.Request, domain string) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if err := s.manager.Disable(domain); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled", "domain": domain})
}

func (s *Server) handleSiteTest(w http.ResponseWriter, r *http.Request, domain string) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if _, err := s.manager.Get(domain); err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	out, err := s.service.Test(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}

func decodeBody(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if strings.TrimSpace(r.Header.Get("Content-Length")) != "" && dec.More() {
		return errors.New("unexpected trailing data")
	}
	return nil
}
