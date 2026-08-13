package nginx

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/PradhaID/go-nginx-proxy/internal/config"
	"github.com/PradhaID/go-nginx-proxy/internal/shell"
)

type Status struct {
	Running bool   `json:"running"`
	Active  string `json:"active"`
	Version string `json:"version"`
	Pid     int    `json:"pid"`
	Configs int    `json:"configs"`
	Enabled int    `json:"enabled"`
}

type Service struct {
	cfg   *config.Config
	shell *shell.Runner
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg, shell: shell.New(cfg.Sudo)}
}

func (s *Service) Version(ctx context.Context) string {
	out, err := s.shell.Run(ctx, s.cfg.NginxBin, "-v")
	out = strings.TrimSpace(out)
	if err != nil {
		return strings.TrimPrefix(out, "nginx version: ")
	}
	return strings.TrimPrefix(out, "nginx version: ")
}

func (s *Service) Status(ctx context.Context) Status {
	st := Status{Version: s.Version(ctx)}
	if s.cfg.UseSystemctl {
		out, err := s.shell.Run(ctx, s.cfg.SystemctlBin, "is-active", s.cfg.ServiceName)
		st.Active = strings.TrimSpace(out)
		st.Running = err == nil && st.Active == "active"
	} else {
		out, _ := s.shell.Run(ctx, "pgrep", "-f", "^"+s.cfg.NginxBin)
		st.Active = "unknown"
		st.Running = strings.TrimSpace(out) != ""
	}
	if entries, err := os.ReadDir(s.cfg.Enabled); err == nil {
		st.Enabled = len(entries)
	}
	if entries, err := os.ReadDir(s.cfg.Available); err == nil {
		st.Configs = len(entries)
	}
	return st
}

func (s *Service) Start(ctx context.Context) (string, error) {
	if s.cfg.UseSystemctl {
		return s.shell.Run(ctx, s.cfg.SystemctlBin, "start", s.cfg.ServiceName)
	}
	return s.shell.Run(ctx, s.cfg.NginxBin)
}

func (s *Service) Stop(ctx context.Context) (string, error) {
	if s.cfg.UseSystemctl {
		return s.shell.Run(ctx, s.cfg.SystemctlBin, "stop", s.cfg.ServiceName)
	}
	return s.shell.Run(ctx, s.cfg.NginxBin, "-s", "quit")
}

func (s *Service) Restart(ctx context.Context) (string, error) {
	if s.cfg.UseSystemctl {
		return s.shell.Run(ctx, s.cfg.SystemctlBin, "restart", s.cfg.ServiceName)
	}
	if _, err := s.shell.Run(ctx, s.cfg.NginxBin, "-s", "quit"); err != nil {
		return "", err
	}
	return s.shell.Run(ctx, s.cfg.NginxBin)
}

func (s *Service) Reload(ctx context.Context) (string, error) {
	if s.cfg.UseSystemctl {
		return s.shell.Run(ctx, s.cfg.SystemctlBin, "reload", s.cfg.ServiceName)
	}
	return s.shell.Run(ctx, s.cfg.NginxBin, "-s", "reload")
}

func (s *Service) Test(ctx context.Context) (string, error) {
	out, err := s.shell.Run(ctx, s.cfg.NginxBin, "-t", "-c", s.cfg.NginxConfig)
	if err != nil {
		return strings.TrimSpace(out), fmt.Errorf("nginx config test failed: %s", strings.TrimSpace(out))
	}
	return strings.TrimSpace(out), nil
}
