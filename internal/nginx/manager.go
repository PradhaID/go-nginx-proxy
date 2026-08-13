package nginx

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PradhaID/go-nginx-proxy/internal/config"
)

var (
	ErrNilSite       = errors.New("site is nil")
	ErrInvalidDomain = errors.New("invalid domain name")
	ErrInvalidPath   = errors.New("location path must start with /")
	ErrNotFound      = errors.New("site not found")
	ErrExists        = errors.New("site already exists")
)

type Manager struct {
	cfg *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) FileName(domain string) string {
	return domain + ".conf"
}

func (m *Manager) AvailablePath(domain string) string {
	return filepath.Join(m.cfg.Available, m.FileName(domain))
}

func (m *Manager) EnabledPath(domain string) string {
	return filepath.Join(m.cfg.Enabled, m.FileName(domain))
}

func (m *Manager) List() ([]*Site, error) {
	entries, err := os.ReadDir(m.cfg.Available)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sites := make([]*Site, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		domain := strings.TrimSuffix(e.Name(), ".conf")
		site, err := m.load(domain)
		if err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, nil
}

func (m *Manager) Get(domain string) (*Site, error) {
	path := m.AvailablePath(domain)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m.load(domain)
}

// LogDirectives returns the access_log and error_log paths declared in the
// site's config file, or "" when the site does not declare one (so it falls
// back to the default logs). "access_log off;" is treated as unset.
func (m *Manager) LogDirectives(domain string) (accessLog, errorLog string) {
	data, err := os.ReadFile(m.AvailablePath(domain))
	if err != nil {
		return "", ""
	}
	return findLogDirective(data, "access_log"), findLogDirective(data, "error_log")
}

func findLogDirective(data []byte, name string) string {
	re := regexp.MustCompile(`(?m)^\s*` + name + `\s+([^;]+);`)
	for _, m := range re.FindAllStringSubmatch(string(data), -1) {
		fields := strings.Fields(m[1])
		if len(fields) == 0 || fields[0] == "off" {
			continue
		}
		return fields[0]
	}
	return ""
}

func (m *Manager) load(domain string) (*Site, error) {
	path := m.AvailablePath(domain)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	site := &Site{
		Domain:   domain,
		Enabled:  m.IsEnabled(domain),
		HasCert:  m.HasCert(domain),
		External: true,
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "# managed-by: go-nginx-proxy" {
			site.External = false
			continue
		}
		if strings.HasPrefix(line, "# meta: ") {
			raw := strings.TrimPrefix(line, "# meta: ")
			if err := json.Unmarshal([]byte(raw), site); err != nil {
				site.External = true
				continue
			}
			site.External = false
			if site.Domain == "" {
				site.Domain = domain
			}
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	site.Enabled = m.IsEnabled(domain)
	site.HasCert = m.HasCert(domain)
	return site, nil
}

func (m *Manager) Create(s *Site) error {
	if err := s.Validate(); err != nil {
		return err
	}
	path := m.AvailablePath(s.Domain)
	if _, err := os.Stat(path); err == nil {
		return ErrExists
	}
	if err := os.MkdirAll(m.cfg.Available, 0o755); err != nil {
		return err
	}
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	return m.write(s)
}

func (m *Manager) Update(domain string, s *Site) error {
	if err := s.Validate(); err != nil {
		return err
	}
	old, err := m.Get(domain)
	if err != nil {
		return err
	}
	s.CreatedAt = old.CreatedAt
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = time.Now()
	if s.Domain != domain {
		if err := m.Delete(domain); err != nil {
			return err
		}
	}
	return m.write(s)
}

func (m *Manager) write(s *Site) error {
	fullchain, privkey := m.cfg.CertPath(s.Domain)
	if !s.SSL || !m.HasCert(s.Domain) {
		fullchain, privkey = "", ""
	}
	content, err := renderSite(s, fullchain, privkey)
	if err != nil {
		return err
	}
	path := m.AvailablePath(s.Domain)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return os.Rename(tmp, path)
}

func (m *Manager) Delete(domain string) error {
	if err := m.Disable(domain); err != nil {
		return err
	}
	err := os.Remove(m.AvailablePath(domain))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (m *Manager) Enable(domain string) error {
	src := m.AvailablePath(domain)
	dst := m.EnabledPath(domain)
	if _, err := os.Stat(src); err != nil {
		return ErrNotFound
	}
	if err := os.MkdirAll(m.cfg.Enabled, 0o755); err != nil {
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink(src, dst); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Disable(domain string) error {
	err := os.Remove(m.EnabledPath(domain))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (m *Manager) IsEnabled(domain string) bool {
	fi, err := os.Lstat(m.EnabledPath(domain))
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func (m *Manager) HasCert(domain string) bool {
	_, priv := m.cfg.CertPath(domain)
	_, err := os.Stat(priv)
	return err == nil
}
