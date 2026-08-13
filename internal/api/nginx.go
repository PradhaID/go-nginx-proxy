package api

import (
	"net/http"
)

func (s *Server) handleNginxStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.service.Status(r.Context()))
}

func (s *Server) handleNginxStart(w http.ResponseWriter, r *http.Request) {
	s.nginxAction(w, r, s.handleNginxStartRun)
}

func (s *Server) handleNginxStartRun(w http.ResponseWriter, r *http.Request) {
	out, err := s.service.Start(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}

func (s *Server) handleNginxStop(w http.ResponseWriter, r *http.Request) {
	s.nginxAction(w, r, s.handleNginxStopRun)
}

func (s *Server) handleNginxStopRun(w http.ResponseWriter, r *http.Request) {
	out, err := s.service.Stop(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}

func (s *Server) handleNginxRestart(w http.ResponseWriter, r *http.Request) {
	s.nginxAction(w, r, s.handleNginxRestartRun)
}

func (s *Server) handleNginxRestartRun(w http.ResponseWriter, r *http.Request) {
	out, err := s.service.Restart(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}

func (s *Server) handleNginxReload(w http.ResponseWriter, r *http.Request) {
	s.nginxAction(w, r, s.handleNginxReloadRun)
}

func (s *Server) handleNginxReloadRun(w http.ResponseWriter, r *http.Request) {
	out, err := s.service.Reload(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}

func (s *Server) handleNginxTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	out, err := s.service.Test(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "output": out, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "output": out})
}

type actionFunc func(w http.ResponseWriter, r *http.Request)

func (s *Server) nginxAction(w http.ResponseWriter, r *http.Request, fn actionFunc) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	fn(w, r)
}
