package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Addr                  string
	Username              string
	Password              string
	Sudo                  bool
	WebDir                string
	Available             string
	Enabled               string
	NginxBin              string
	NginxConfig           string
	UseSystemctl          bool
	SystemctlBin          string
	ServiceName           string
	CertbotBin            string
	CertbotEmail          string
	CertbotStaging        bool
	CertDir               string
	CloudflareToken       string
	CloudflareCredFile    string
	CloudflarePropagation int
	AccessLog             string
	ErrorLog              string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func Load() *Config {
	return &Config{
		Addr:                  getenv("PROXY_ADDR", ":8080"),
		Username:              getenv("PROXY_USERNAME", "admin"),
		Password:              getenv("PROXY_PASSWORD", "changeme"),
		Sudo:                  getenvBool("PROXY_SUDO", false),
		WebDir:                os.Getenv("PROXY_WEB_DIR"),
		Available:             getenv("PROXY_NGINX_AVAILABLE_DIR", "/etc/nginx/sites-available"),
		Enabled:               getenv("PROXY_NGINX_ENABLED_DIR", "/etc/nginx/sites-enabled"),
		NginxBin:              getenv("PROXY_NGINX_BIN", "nginx"),
		NginxConfig:           getenv("PROXY_NGINX_CONFIG", "/etc/nginx/nginx.conf"),
		UseSystemctl:          getenvBool("PROXY_USE_SYSTEMCTL", true),
		SystemctlBin:          getenv("PROXY_SYSTEMCTL_BIN", "systemctl"),
		ServiceName:           getenv("PROXY_SERVICE_NAME", "nginx"),
		CertbotBin:            getenv("PROXY_CERTBOT_BIN", "certbot"),
		CertbotEmail:          os.Getenv("PROXY_CERTBOT_EMAIL"),
		CertbotStaging:        getenvBool("PROXY_CERTBOT_STAGING", false),
		CertDir:               getenv("PROXY_CERT_DIR", "/etc/letsencrypt/live"),
		CloudflareToken:       os.Getenv("PROXY_CLOUDFLARE_DNS_TOKEN"),
		CloudflareCredFile:    getenv("PROXY_CLOUDFLARE_CREDENTIALS_FILE", "/etc/letsencrypt/cloudflare.ini"),
		CloudflarePropagation: getenvInt("PROXY_CLOUDFLARE_PROPAGATION", 60),
		AccessLog:             getenv("PROXY_NGINX_ACCESS_LOG", "/var/log/nginx/access.log"),
		ErrorLog:              getenv("PROXY_NGINX_ERROR_LOG", "/var/log/nginx/error.log"),
	}
}

func (c *Config) CertPath(domain string) (fullchain, privkey string) {
	base := filepath.Join(c.CertDir, domain)
	return filepath.Join(base, "fullchain.pem"), filepath.Join(base, "privkey.pem")
}
