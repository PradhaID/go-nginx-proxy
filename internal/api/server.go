package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/PradhaID/go-nginx-proxy/internal/certbot"
	"github.com/PradhaID/go-nginx-proxy/internal/config"
	"github.com/PradhaID/go-nginx-proxy/internal/nginx"
)

type Server struct {
	cfg      *config.Config
	manager  *nginx.Manager
	service  *nginx.Service
	certbot  *certbot.Certbot
	realtime *realtimeHub
	mux      *http.ServeMux
}

func New(cfg *config.Config, staticFS http.Handler) *Server {
	s := &Server{
		cfg:      cfg,
		manager:  nginx.NewManager(cfg),
		service:  nginx.NewService(cfg),
		certbot:  certbot.New(cfg),
		realtime: newRealtimeHub(),
		mux:      http.NewServeMux(),
	}

	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/events", s.handleEvents)
	s.mux.HandleFunc("/api/nginx/status", s.handleNginxStatus)
	s.mux.HandleFunc("/api/nginx/start", s.handleNginxStart)
	s.mux.HandleFunc("/api/nginx/stop", s.handleNginxStop)
	s.mux.HandleFunc("/api/nginx/restart", s.handleNginxRestart)
	s.mux.HandleFunc("/api/nginx/reload", s.handleNginxReload)
	s.mux.HandleFunc("/api/nginx/test", s.handleNginxTest)

	s.mux.HandleFunc("/api/sites", s.handleSites)
	s.mux.HandleFunc("/api/sites/", s.handleSite)

	s.mux.HandleFunc("/api/logs", s.handleLogs)
	s.mux.HandleFunc("/api/logs/stream", s.handleLogStream)

	if staticFS != nil {
		s.mux.Handle("/", staticFS)
	} else {
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}
	s.startWatcher()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.auth(s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": version,
	})
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Password == "" {
			next.ServeHTTP(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != s.cfg.Username || pass != s.cfg.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="go-nginx-proxy"`)
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleSite routes /api/sites/{domain}[/{action}[/{subaction}]]
func (s *Server) handleSite(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	parts := strings.SplitN(path, "/", 3)
	domain := parts[0]
	if domain == "" {
		http.Error(w, `{"error":"domain required"}`, http.StatusBadRequest)
		return
	}
	action := ""
	sub := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	if len(parts) > 2 {
		sub = parts[2]
	}

	switch {
	case action == "":
		switch r.Method {
		case http.MethodGet:
			s.handleSiteGet(w, r, domain)
		case http.MethodPut:
			s.handleSiteUpdate(w, r, domain)
		case http.MethodDelete:
			s.handleSiteDelete(w, r, domain)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	case action == "enable":
		s.handleSiteEnable(w, r, domain)
	case action == "disable":
		s.handleSiteDisable(w, r, domain)
	case action == "test":
		s.handleSiteTest(w, r, domain)
	case action == "cert" && sub == "":
		switch r.Method {
		case http.MethodGet:
			s.handleCertStatus(w, r, domain)
		case http.MethodPost:
			s.handleCertIssue(w, r, domain)
		case http.MethodDelete:
			s.handleCertDelete(w, r, domain)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	case action == "cert" && sub == "renew":
		s.handleCertRenew(w, r, domain)
	default:
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	log.Printf("error: %v", err)
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

const version = "0.1.0"
