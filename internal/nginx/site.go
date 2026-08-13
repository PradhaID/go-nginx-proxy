package nginx

import (
	"encoding/json"
	"regexp"
	"time"
)

type Location struct {
	Path      string `json:"path"`
	ProxyPass string `json:"proxyPass"`
	Websocket bool   `json:"websocket"`
	Extra     string `json:"extra"`
}

type Site struct {
	Domain            string     `json:"domain"`
	Domains           []string   `json:"domains"`
	SSL               bool       `json:"ssl"`
	RedirectToHTTPS   bool       `json:"redirectToHttps"`
	ClientMaxBodySize string     `json:"clientMaxBodySize"`
	WebRoot           string     `json:"webRoot"`
	Locations         []Location `json:"locations"`
	ExtraConfig       string     `json:"extraConfig"`
	Enabled           bool       `json:"enabled"`
	HasCert           bool       `json:"hasCert"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	External          bool       `json:"external"`
}

var hostnameRe = regexp.MustCompile(`^(?i)([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$`)

func (s *Site) AllDomains() []string {
	if s == nil {
		return nil
	}
	seen := map[string]bool{s.Domain: true}
	out := []string{s.Domain}
	for _, d := range s.Domains {
		if d != "" && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

func (s *Site) Validate() error {
	if s == nil {
		return ErrNilSite
	}
	if !hostnameRe.MatchString(s.Domain) {
		return ErrInvalidDomain
	}
	for _, l := range s.Locations {
		if l.Path == "" || l.Path[0] != '/' {
			return ErrInvalidPath
		}
	}
	return nil
}

func (s *Site) Meta() string {
	meta := map[string]any{
		"version":           1,
		"domain":            s.Domain,
		"domains":           s.Domains,
		"ssl":               s.SSL,
		"redirectToHttps":   s.RedirectToHTTPS,
		"clientMaxBodySize": s.ClientMaxBodySize,
		"webRoot":           s.WebRoot,
		"locations":         s.Locations,
		"extraConfig":       s.ExtraConfig,
		"createdAt":         s.CreatedAt,
		"updatedAt":         s.UpdatedAt,
	}
	b, _ := json.Marshal(meta)
	return string(b)
}
