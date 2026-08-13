package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PradhaID/go-nginx-proxy/internal/config"
)

func TestRenderSite(t *testing.T) {
	site := &Site{
		Domain:            "example.com",
		Domains:           []string{"www.example.com"},
		SSL:               true,
		RedirectToHTTPS:   true,
		ClientMaxBodySize: "10m",
		Locations: []Location{
			{Path: "/", ProxyPass: "http://127.0.0.1:3000", Websocket: true},
		},
	}
	out, err := renderSite(site, "/etc/letsencrypt/live/example.com/fullchain.pem", "/etc/letsencrypt/live/example.com/privkey.pem")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"server_name example.com www.example.com;",
		"listen 443 ssl;",
		"return 301 https://$host$request_uri;",
		"proxy_pass http://127.0.0.1:3000;",
		"client_max_body_size 10m;",
		"proxy_set_header Upgrade $http_upgrade;",
		"# meta:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered config missing %q:\n%s", want, out)
		}
	}
}

func TestManagerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Available: filepath.Join(dir, "available"),
		Enabled:   filepath.Join(dir, "enabled"),
		CertDir:   filepath.Join(dir, "certs"),
	}
	m := NewManager(cfg)

	site := &Site{
		Domain:      "test.dev",
		SSL:         false,
		Locations:   []Location{{Path: "/", ProxyPass: "http://127.0.0.1:8080"}},
		ExtraConfig: "add_header X-Foo bar;",
	}
	if err := m.Create(site); err != nil {
		t.Fatal(err)
	}

	// symlink should exist after enable
	if err := m.Enable("test.dev"); err != nil {
		t.Fatal(err)
	}
	if !m.IsEnabled("test.dev") {
		t.Fatal("expected site enabled")
	}

	got, err := m.Get("test.dev")
	if err != nil {
		t.Fatal(err)
	}
	if got.Domain != "test.dev" || len(got.Locations) != 1 || got.External {
		t.Fatalf("round-trip failed: %+v", got)
	}
	if got.Locations[0].ProxyPass != "http://127.0.0.1:8080" {
		t.Fatalf("location mismatch: %+v", got.Locations[0])
	}

	// written file should have meta comment
	data, err := os.ReadFile(m.AvailablePath("test.dev"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# managed-by: go-nginx-proxy") {
		t.Fatalf("missing managed-by header:\n%s", data)
	}

	if err := m.Disable("test.dev"); err != nil {
		t.Fatal(err)
	}
	if m.IsEnabled("test.dev") {
		t.Fatal("expected site disabled")
	}

	if err := m.Delete("test.dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.AvailablePath("test.dev")); !os.IsNotExist(err) {
		t.Fatal("expected file removed")
	}
}
