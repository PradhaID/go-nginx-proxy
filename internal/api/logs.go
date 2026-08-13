package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/PradhaID/go-nginx-proxy/internal/nginx"
)

func (s *Server) logPath(kind string) string {
	if kind == "error" {
		return s.cfg.ErrorLog
	}
	return s.cfg.AccessLog
}

func logKind(kind, line string) string {
	if kind == "access" {
		return nginx.ClassifyAccessLine(line)
	}
	return ""
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
	lines := 200
	if v := r.URL.Query().Get("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
			lines = n
		}
	}

	path := s.logPath(kind)
	raw, err := s.service.TailLog(path, lines)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]nginx.LogLine, 0, len(raw))
	for _, ln := range raw {
		out = append(out, nginx.LogLine{Text: ln, Kind: logKind(kind, ln)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"type": kind, "path": path, "lines": out})
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
	path := s.logPath(kind)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if raw, err := s.service.TailLog(path, 200); err == nil {
		out := make([]nginx.LogLine, 0, len(raw))
		for _, ln := range raw {
			out = append(out, nginx.LogLine{Text: ln, Kind: logKind(kind, ln)})
		}
		s.writeSSE(w, flusher, "snapshot", out)
	}

	ctx := r.Context()
	if err := s.service.StreamLog(ctx, path, func(text string) error {
		data, err := json.Marshal(nginx.LogLine{Text: text, Kind: logKind(kind, text)})
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: log\ndata: %s\n\n", data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}); err != nil {
		log.Printf("log stream %s: %v", kind, err)
	}
}
