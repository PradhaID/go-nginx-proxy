package certbot

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PradhaID/go-nginx-proxy/internal/config"
	"github.com/PradhaID/go-nginx-proxy/internal/shell"
)

type Certbot struct {
	cfg   *config.Config
	shell *shell.Runner
}

type Status struct {
	Issued        bool      `json:"issued"`
	Domain        string    `json:"domain"`
	NotAfter      time.Time `json:"notAfter"`
	NotBefore     time.Time `json:"notBefore"`
	ExpiresInDays int       `json:"expiresInDays"`
	RenewAt       string    `json:"renewAt"`
}

func New(cfg *config.Config) *Certbot {
	return &Certbot{cfg: cfg, shell: shell.New(cfg.Sudo)}
}

func (c *Certbot) hasProvider() string {
	if c.cfg.CloudflareToken != "" {
		return "cloudflare"
	}
	return "webroot"
}

func (c *Certbot) ensureCloudflareCreds() error {
	if c.cfg.CloudflareToken == "" {
		return nil
	}
	if _, err := os.Stat(c.cfg.CloudflareCredFile); err == nil {
		return nil
	}
	dir := filepath.Dir(c.cfg.CloudflareCredFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("dns_cloudflare_api_token = %s\n", c.cfg.CloudflareToken)
	tmp := c.cfg.CloudflareCredFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, c.cfg.CloudflareCredFile); err != nil {
		return err
	}
	return nil
}

func (c *Certbot) baseArgs(domains []string) ([]string, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("no domains provided")
	}
	args := []string{"certonly", "--non-interactive", "--agree-tos", "--no-eff-email"}
	for _, d := range domains {
		args = append(args, "-d", d)
	}
	if c.cfg.CertbotEmail != "" {
		args = append(args, "--email", c.cfg.CertbotEmail)
	}
	if c.cfg.CertbotStaging {
		args = append(args, "--staging")
	}
	switch c.hasProvider() {
	case "cloudflare":
		if err := c.ensureCloudflareCreds(); err != nil {
			return nil, err
		}
		args = append(args,
			"--dns-cloudflare",
			"--dns-cloudflare-credentials", c.cfg.CloudflareCredFile,
			"--dns-cloudflare-propagation-seconds", fmt.Sprint(c.cfg.CloudflarePropagation),
		)
	default:
		args = append(args, "--webroot", "-w", webroot)
	}
	return args, nil
}

func (c *Certbot) Issue(ctx context.Context, domains []string) (string, error) {
	args, err := c.baseArgs(domains)
	if err != nil {
		return "", err
	}
	return c.shell.Run(ctx, c.cfg.CertbotBin, args...)
}

func (c *Certbot) Renew(ctx context.Context, domains []string) (string, error) {
	args := []string{"renew", "--non-interactive", "--keep-until-expiring", "--no-random-sleep-on-renew"}
	if c.cfg.CertbotStaging {
		args = append(args, "--staging")
	}
	args = append(args, "--cert-name", domains[0])
	return c.shell.Run(ctx, c.cfg.CertbotBin, args...)
}

func (c *Certbot) Status(domain string) Status {
	st := Status{Domain: domain}
	fullchain := filepath.Join(c.cfg.CertDir, domain, "fullchain.pem")
	data, err := os.ReadFile(fullchain)
	if err != nil {
		return st
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return st
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return st
	}
	st.Issued = true
	st.NotAfter = cert.NotAfter
	st.NotBefore = cert.NotBefore
	st.ExpiresInDays = int(time.Until(cert.NotAfter).Hours() / 24)
	renew := cert.NotAfter.AddDate(0, 0, -30)
	st.RenewAt = renew.Format(time.RFC3339)
	return st
}

func (c *Certbot) Delete(ctx context.Context, domain string) (string, error) {
	return c.shell.Run(ctx, c.cfg.CertbotBin, "delete", "--cert-name", domain)
}

const webroot = "/var/www/html"
