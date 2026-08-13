package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PradhaID/go-nginx-proxy/internal/nginx"
)

type subscriber struct {
	ch chan []byte
}

type realtimeHub struct {
	mu   sync.Mutex
	subs map[*subscriber]struct{}
}

func newRealtimeHub() *realtimeHub {
	return &realtimeHub{subs: make(map[*subscriber]struct{})}
}

func (h *realtimeHub) subscribe() *subscriber {
	s := &subscriber{ch: make(chan []byte, 8)}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s
}

func (h *realtimeHub) unsubscribe(s *subscriber) {
	h.mu.Lock()
	delete(h.subs, s)
	h.mu.Unlock()
}

func (h *realtimeHub) publish(event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data))
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		select {
		case s.ch <- msg:
		default:
			// slow subscriber, drop this update
		}
	}
}

// handleEvents streams status + sites updates over SSE.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub := s.realtime.subscribe()
	defer s.realtime.unsubscribe(sub)

	// initial state
	s.writeSSE(w, flusher, "status", s.service.Status(r.Context()))
	s.writeSSE(w, flusher, "sites", s.siteList())

	ctx := r.Context()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-sub.ch:
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	flusher.Flush()
}

func (s *Server) siteList() map[string]any {
	sites, err := s.manager.List()
	if err != nil {
		log.Printf("realtime: list sites: %v", err)
		return map[string]any{"sites": []*nginx.Site{}}
	}
	if sites == nil {
		sites = []*nginx.Site{}
	}
	return map[string]any{"sites": sites}
}

// startWatcher polls the config directories and broadcasts changes to
// SSE subscribers. Falls back to a periodic status refresh so UIs stay
// fresh even when nginx state changes without any file change.
func (s *Server) startWatcher() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		last := ""
		ticks := 0
		for range ticker.C {
			sig := s.fsSignature()
			ticks++
			if sig != last {
				last = sig
				ticks = 0
				s.broadcastAll()
			} else if ticks >= 8 { // ~16s heartbeat
				ticks = 0
				s.broadcastStatus()
			}
		}
	}()
}

func (s *Server) broadcastAll() {
	s.broadcastStatus()
	s.realtime.publish("sites", s.siteList())
}

func (s *Server) broadcastStatus() {
	s.realtime.publish("status", s.service.Status(context.Background()))
}

func (s *Server) fsSignature() string {
	return s.dirSignature(s.cfg.Available) + "|" + s.dirSignature(s.cfg.Enabled)
}

func (s *Server) dirSignature(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "err:" + err.Error()
	}
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		parts = append(parts, e.Name()+"@"+info.ModTime().Format(time.RFC3339Nano))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
