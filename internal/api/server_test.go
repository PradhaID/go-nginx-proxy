package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/PradhaID/go-nginx-proxy/internal/config"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		Username:  "admin",
		Password:  "secret",
		Available: filepath.Join(dir, "available"),
		Enabled:   filepath.Join(dir, "enabled"),
		CertDir:   filepath.Join(dir, "certs"),
	}
	cfg.Available = filepath.Join(dir, "available")
	cfg.Enabled = filepath.Join(dir, "enabled")
	srv := New(cfg, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestAuth(t *testing.T) {
	ts := newTestServer(t)

	res, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/health", nil)
	req.SetBasicAuth("admin", "secret")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestSSE(t *testing.T) {
	ts := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/events", nil)
	req.SetBasicAuth("admin", "secret")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}
	if res.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatalf("expected X-Accel-Buffering: no, got %q", res.Header.Get("X-Accel-Buffering"))
	}

	buf := make([]byte, 512)
	n, err := res.Body.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("read: %v", err)
	}
	body := string(buf[:n])
	if len(body) == 0 {
		t.Fatal("expected initial SSE payload")
	}
	if !bytes.Contains(buf[:n], []byte("event: status")) {
		t.Fatalf("expected event: status in initial payload, got %q", body)
	}
}

func TestSitesCRUD(t *testing.T) {
	ts := newTestServer(t)
	do := func(method, path string, body any) (*http.Response, map[string]any) {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req, _ := http.NewRequest(method, ts.URL+path, &buf)
		req.SetBasicAuth("admin", "secret")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var m map[string]any
		_ = json.NewDecoder(res.Body).Decode(&m)
		return res, m
	}

	site := map[string]any{
		"domain": "app.example.com",
		"locations": []map[string]any{
			{"path": "/", "proxyPass": "http://127.0.0.1:3000", "websocket": true},
		},
	}

	res, m := do(http.MethodPost, "/api/sites", site)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d (%v)", res.StatusCode, m)
	}

	res, _ = do(http.MethodPost, "/api/sites/app.example.com/enable", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("enable: expected 200, got %d", res.StatusCode)
	}

	res, m = do(http.MethodGet, "/api/sites", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", res.StatusCode)
	}
	list := m["sites"].([]any)
	if len(list) != 1 {
		t.Fatalf("expected 1 site, got %d", len(list))
	}

	res, _ = do(http.MethodPut, "/api/sites/app.example.com", map[string]any{
		"domain":            "app.example.com",
		"clientMaxBodySize": "20m",
		"locations":         site["locations"],
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update: expected 200, got %d", res.StatusCode)
	}

	res, _ = do(http.MethodDelete, "/api/sites/app.example.com", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d", res.StatusCode)
	}

	res, _ = do(http.MethodGet, "/api/sites/app.example.com", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", res.StatusCode)
	}
}
