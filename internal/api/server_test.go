package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/PradhaID/go-nginx-proxy/internal/config"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	access := filepath.Join(dir, "access.log")
	errorLog := filepath.Join(dir, "error.log")
	os.WriteFile(access, []byte(
		`1.1.1.1 - - [13/Aug/2026:10:00:00 +0700] "GET / HTTP/1.1" 200 1 "-" "curl/8.5.0"`+"\n"+
			`2.2.2.2 - - [13/Aug/2026:10:00:01 +0700] "GET / HTTP/1.1" 200 1 "-" "Mozilla/5.0 (X11; Linux x86_64) Firefox/128.0"`+"\n"), 0o644)
	os.WriteFile(errorLog, []byte("2026/08/13 10:00:00 [error] 123#0: connect() failed\n"), 0o644)
	cfg := &config.Config{
		Username:  "admin",
		Password:  "secret",
		Available: filepath.Join(dir, "available"),
		Enabled:   filepath.Join(dir, "enabled"),
		CertDir:   filepath.Join(dir, "certs"),
		AccessLog: access,
		ErrorLog:  errorLog,
	}
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

func TestLogs(t *testing.T) {
	ts := newTestServer(t)
	do := func(q string) (*http.Response, map[string]any) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/logs"+q, nil)
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

	res, m := do("?type=access")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lines := m["lines"].([]any)
	if len(lines) != 2 {
		t.Fatalf("expected 2 access lines, got %d", len(lines))
	}
	first := lines[0].(map[string]any)
	if first["kind"] != "bot" {
		t.Fatalf("expected first line kind=bot, got %v", first["kind"])
	}
	second := lines[1].(map[string]any)
	if second["kind"] != "human" {
		t.Fatalf("expected second line kind=human, got %v", second["kind"])
	}

	res, m = do("?type=error")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	el := m["lines"].([]any)
	if len(el) != 1 {
		t.Fatalf("expected 1 error line, got %d", len(el))
	}

	res, m = do("?type=access&lines=1")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if n := len(m["lines"].([]any)); n != 1 {
		t.Fatalf("expected 1 line with lines=1, got %d", n)
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
