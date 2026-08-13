package nginx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PradhaID/go-nginx-proxy/internal/config"
)

func newTestService() *Service {
	return NewService(&config.Config{})
}

func TestExtractUserAgent(t *testing.T) {
	line := `1.2.3.4 - - [13/Aug/2026:10:00:00 +0700] "GET / HTTP/1.1" 200 1234 "-" "Mozilla/5.0 (X11; Linux x86_64) Firefox/128.0"`
	if ua := extractUserAgent(line); ua != "Mozilla/5.0 (X11; Linux x86_64) Firefox/128.0" {
		t.Fatalf("got %q", ua)
	}
	if ua := extractUserAgent(`not a log line`); ua != "" {
		t.Fatalf("expected empty, got %q", ua)
	}
}

func TestExtractDomain(t *testing.T) {
	cases := []struct{ line, want string }{
		{`pradha.id 104.22.93.4 [13/Aug/2026:10:31:49 +0700] "GET / HTTP/1.1" 200 16 "-" "Mozilla/5.0"`, "pradha.id"},
		{`1.2.3.4 - - [13/Aug/2026:10:31:49 +0700] "GET / HTTP/1.1" 200 16 "-" "Mozilla/5.0"`, ""},
		{`2001:db8::1 - - [13/Aug/2026:10:31:49 +0700] "GET / HTTP/1.1" 200 16 "-" "-"`, ""},
		{``, ""},
	}
	for _, c := range cases {
		if got := ExtractDomain(c.line); got != c.want {
			t.Fatalf("ExtractDomain(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

func TestLogDirectives(t *testing.T) {
	m := NewManager(&config.Config{Available: t.TempDir(), Enabled: t.TempDir()})
	dir := m.cfg.Available
	os.WriteFile(filepath.Join(dir, "own.conf"), []byte(`server {
    listen 443 ssl;
    server_name own.example.com;
    access_log /var/log/nginx/own.example.com.access.log proxy_logged;
    access_log off;
    error_log /var/log/nginx/own.example.com.error.log warn;
    location / { proxy_pass http://127.0.0.1:8080; }
}`), 0o644)
	os.WriteFile(filepath.Join(dir, "fallback.conf"), []byte(`server {
    server_name fallback.example.com;
    access_log off;
    location / { proxy_pass http://127.0.0.1:8080; }
}`), 0o644)

	a, e := m.LogDirectives("own")
	if a != "/var/log/nginx/own.example.com.access.log" || e != "/var/log/nginx/own.example.com.error.log" {
		t.Fatalf("own: got %q %q", a, e)
	}
	a, e = m.LogDirectives("fallback")
	if a != "" || e != "" {
		t.Fatalf("fallback: expected empty, got %q %q", a, e)
	}
	a, e = m.LogDirectives("missing")
	if a != "" || e != "" {
		t.Fatalf("missing: expected empty, got %q %q", a, e)
	}
}

func TestClassifyAccessLine(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{`1.1.1.1 - - [13/Aug/2026:10:00:00 +0700] "GET / HTTP/1.1" 200 1 "-" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/151.0 Safari/537.36"`, "human"},
		{`1.1.1.1 - - [13/Aug/2026:10:00:00 +0700] "GET /robots.txt HTTP/1.1" 200 1 "-" "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"`, "bot"},
		{`1.1.1.1 - - [13/Aug/2026:10:00:00 +0700] "GET / HTTP/1.1" 200 1 "-" "curl/8.5.0"`, "bot"},
		{`1.1.1.1 - - [13/Aug/2026:10:00:00 +0700] "GET / HTTP/1.1" 200 1 "-" "-"`, "other"},
	}
	for _, c := range cases {
		if got := ClassifyAccessLine(c.line); got != c.want {
			t.Fatalf("ClassifyAccessLine(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

func TestTailLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	content := ""
	for i := 0; i < 10; i++ {
		content += "line " + string(rune('a'+i)) + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newTestService()
	tail, err := s.TailLog(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 3 || tail[0] != "line h" {
		t.Fatalf("got %v", tail)
	}
	all, err := s.TailLog(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 10 {
		t.Fatalf("expected 10, got %d", len(all))
	}
}

func TestStreamLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := newTestService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got := make(chan string, 10)
	go func() {
		_ = s.StreamLog(ctx, path, func(line string) error {
			got <- line
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("first\nsecond\n")
	f.Close()

	select {
	case l := <-got:
		if l != "first" {
			t.Fatalf("expected first, got %q", l)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for streamed line")
	}
	select {
	case l := <-got:
		if l != "second" {
			t.Fatalf("expected second, got %q", l)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for second line")
	}

	cancel()
	// no more lines
	select {
	case l := <-got:
		t.Fatalf("unexpected line after cancel: %q", l)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestStreamLogRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := newTestService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got := make(chan string, 10)
	go func() {
		_ = s.StreamLog(ctx, path, func(line string) error {
			got <- line
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)
	// rotate: rename then recreate (like logrotate)
	if err := os.Rename(path, filepath.Join(dir, "access.log.1")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fresh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case l := <-got:
		if l != "fresh" {
			t.Fatalf("expected fresh after rotation, got %q", l)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for rotated log line")
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("more\n")
	f.Close()

	select {
	case l := <-got:
		if l != "more" {
			t.Fatalf("expected more, got %q", l)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for appended line after rotation")
	}
	_ = strings.TrimSpace
}
