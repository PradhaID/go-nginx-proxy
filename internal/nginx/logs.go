package nginx

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type LogLine struct {
	Text   string `json:"text"`
	Domain string `json:"domain,omitempty"` // access: host from the proxy_logged format
	Kind   string `json:"kind,omitempty"`   // access: bot | human | other
}

var ipv4Token = regexp.MustCompile(`^[\d.]+$`)

func isIPToken(tok string) bool {
	// IPv6 addresses always contain ':', IPv4 addresses are digits+dots
	return strings.Contains(tok, ":") || ipv4Token.MatchString(tok)
}

// ExtractDomain returns the $host field of a line written with the
// proxy_logged format (domain first), or "" for lines that do not carry one
// (e.g. the default combined format, which starts with the client IP).
func ExtractDomain(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 || isIPToken(fields[0]) {
		return ""
	}
	return fields[0]
}

// TailLog returns up to `lines` lines from the end of the given log file.
func (s *Service) TailLog(path string, lines int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		all = append(all, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if lines > 0 && len(all) > lines {
		all = all[len(all)-lines:]
	}
	return all, nil
}

// StreamLog emits new lines appended to the log at `path` (following rotation
// and truncation) until ctx is cancelled or send returns an error.
func (s *Service) StreamLog(ctx context.Context, path string, send func(string) error) error {
	var f *os.File
	var pos int64
	var dev, ino uint64
	var pending []byte

	for {
		if ctx.Err() != nil {
			return nil
		}
		fi, err := os.Stat(path)
		if err != nil {
			if f != nil {
				f.Close()
				f = nil
			}
			if os.IsNotExist(err) {
				if !sleep(ctx) {
					return nil
				}
				continue
			}
			return err
		}
		curDev, curIno := statIDs(fi)
		if f == nil || curIno != ino || curDev != dev {
			first := f == nil
			if f != nil {
				f.Close()
				f = nil
			}
			f, err = os.Open(path)
			if err != nil {
				return err
			}
			dev, ino = curDev, curIno
			if first {
				// initial open: skip existing content, only follow new lines
				pos = fi.Size()
			} else {
				// rotation: read the fresh file from the top
				pos = 0
			}
			pending = pending[:0]
		}
		if fi.Size() < pos {
			pos = 0
			pending = pending[:0]
		}

		buf := make([]byte, 64*1024)
		n, _ := f.ReadAt(buf, pos)
		if n > 0 {
			pos += int64(n)
			chunk := append(pending, buf[:n]...)
			raw := strings.Split(string(chunk), "\n")
			if len(raw) > 0 {
				pending = []byte(raw[len(raw)-1])
				for _, ln := range raw[:len(raw)-1] {
					text := strings.TrimRight(ln, "\r")
					if text == "" {
						continue
					}
					if err := send(text); err != nil {
						return err
					}
				}
			}
		}

		if !sleep(ctx) {
			return nil
		}
	}
}

func sleep(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Second):
		return true
	}
}

func statIDs(fi os.FileInfo) (dev, ino uint64) {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Dev), uint64(st.Ino)
	}
	return 0, 0
}

// extractUserAgent pulls the $http_user_agent field out of a line in nginx
// combined log format: host - user [date] "request" status bytes "referer" "ua".
func extractUserAgent(line string) string {
	parts := strings.Split(line, `"`)
	if len(parts) >= 6 {
		return parts[5]
	}
	return ""
}

var botUAPattern = regexp.MustCompile(`(?i)(googlebot|bingbot|bingpreview|msnbot|slurp|duckduckbot|yandex|baiduspider|sogou|exabot|facebookexternalhit|facebot|twitterbot|linkedinbot|pinterest|semrush|ahrefs|mj12|dotbot|petalbot|bytespider|amazonbot|ccbot|applebot|chatgpt|gptbot|claudebot|anthropic|perplexitybot|seekr|cohere|crawler|spider|\bbot\b|curl|wget|python-requests|python-urllib|go-http-client|httpx|httpie|postmanruntime|node-fetch|axios|okhttp|java/|libwww-perl|lwp-|apache-httpclient|php-crawler|headlesschrome|phantomjs|puppeteer|playwright|selenium|monitor|statuscake|uptimerobot|uptime-?robot|newrelic|datadog|cloudflare|prism|nagios|zabbix|nmap|masscan|zgrab|ipscan)`)

// ClassifyAccessLine returns "bot" when the User-Agent matches known bots,
// "human" for browser traffic, or "other" when no UA is present.
func ClassifyAccessLine(line string) string {
	ua := extractUserAgent(line)
	if ua == "" || ua == "-" {
		return "other"
	}
	if botUAPattern.MatchString(ua) {
		return "bot"
	}
	return "human"
}
