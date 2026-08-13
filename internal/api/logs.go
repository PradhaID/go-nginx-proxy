package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/PradhaID/go-nginx-proxy/internal/nginx"
)

func logKind(kind, line string) string {
	if kind == "access" {
		return nginx.ClassifyAccessLine(line)
	}
	return ""
}

// logResolve returns the log file to read for a domain + kind. Domains that
// declare their own access_log/error_log use that file; everything else falls
// back to the default logs.
func (s *Server) logResolve(domain, kind string) string {
	if domain == "" {
		if kind == "error" {
			return s.cfg.ErrorLog
		}
		return s.cfg.AccessLog
	}
	accessLog, errorLog := s.manager.LogDirectives(domain)
	if kind == "error" {
		if errorLog != "" {
			return errorLog
		}
		return s.cfg.ErrorLog
	}
	if accessLog != "" {
		return accessLog
	}
	return s.cfg.AccessLog
}

// logNeedsDomainFilter reports whether lines read from `path` must be filtered
// by domain. Only needed when a domain falls back to the shared default access
// log (whose proxy_logged format carries $host first).
func (s *Server) logNeedsDomainFilter(domain, path, kind string) bool {
	if domain == "" || kind == "error" {
		return false
	}
	if accessLog, _ := s.manager.LogDirectives(domain); accessLog != "" {
		return false // dedicated file, all lines belong to the domain
	}
	return path == s.cfg.AccessLog
}

func (s *Server) decorateLine(kind, domain, line string) nginx.LogLine {
	l := nginx.LogLine{Text: line, Kind: logKind(kind, line)}
	if kind != "access" {
		return l
	}
	if d := nginx.ExtractDomain(line); d != "" {
		l.Domain = d
	} else if domain != "" {
		l.Domain = domain
	}
	return l
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	kind := r.URL.Query().Get("type")
	if kind == "" {
		kind = "access"
	}
	domain := r.URL.Query().Get("domain")
	lines := 200
	if v := r.URL.Query().Get("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
			lines = n
		}
	}

	path := s.logResolve(domain, kind)
	raw, err := s.service.TailLog(path, lines)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	filter := s.logNeedsDomainFilter(domain, path, kind)
	out := make([]nginx.LogLine, 0, len(raw))
	for _, ln := range raw {
		if filter && nginx.ExtractDomain(ln) != domain {
			continue
		}
		out = append(out, s.decorateLine(kind, domain, ln))
	}
	writeJSON(w, http.StatusOK, map[string]any{"type": kind, "domain": domain, "path": path, "lines": out})
}

// handleLogStream tails a log file live over SSE. It first sends a snapshot of
// recent lines, then streams new lines as they are appended.
func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}
	kind := r.URL.Query().Get("type")
	if kind == "" {
		kind = "access"
	}
	domain := r.URL.Query().Get("domain")
	path := s.logResolve(domain, kind)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	filter := s.logNeedsDomainFilter(domain, path, kind)
	if raw, err := s.service.TailLog(path, 200); err == nil {
		out := make([]nginx.LogLine, 0, len(raw))
		for _, ln := range raw {
			if filter && nginx.ExtractDomain(ln) != domain {
				continue
			}
			out = append(out, s.decorateLine(kind, domain, ln))
		}
		s.writeSSE(w, flusher, "snapshot", out)
	}

	ctx := r.Context()
	if err := s.service.StreamLog(ctx, path, func(text string) error {
		if filter && nginx.ExtractDomain(text) != domain {
			return nil
		}
		data, err := json.Marshal(s.decorateLine(kind, domain, text))
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: log\ndata: %s\n\n", data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}); err != nil {
		log.Printf("log stream %s %s: %v", kind, domain, err)
	}
}

// handleLogDomains lists every config domain with the access/error log file
// it writes to (its own declaration, or the default fallback).
func (s *Server) handleLogDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	sites, err := s.manager.List()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	type domainLog struct {
		Domain    string `json:"domain"`
		AccessLog string `json:"accessLog"`
		ErrorLog  string `json:"errorLog"`
	}
	out := make([]domainLog, 0, len(sites))
	for _, site := range sites {
		a, e := s.manager.LogDirectives(site.Domain)
		if a == "" {
			a = s.cfg.AccessLog
		}
		if e == "" {
			e = s.cfg.ErrorLog
		}
		out = append(out, domainLog{Domain: site.Domain, AccessLog: a, ErrorLog: e})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Domain < out[j].Domain })
	writeJSON(w, http.StatusOK, map[string]any{"domains": out})
}
