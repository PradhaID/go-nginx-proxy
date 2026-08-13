package nginx

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

var siteTemplate = template.Must(template.New("site").Funcs(template.FuncMap{
	"joinDomains": func(t templateData) string { return strings.Join(t.AllDomains(), " ") },
	"indent": func(n int, s string) string {
		if s == "" {
			return ""
		}
		pad := strings.Repeat("    ", n)
		lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
		for i := range lines {
			lines[i] = pad + lines[i]
		}
		return strings.Join(lines, "\n")
	},
}).Parse(`# managed-by: go-nginx-proxy
# meta: {{ .Meta }}
{{- if and .SSL .RedirectToHTTPS }}

server {
    listen 80;
    listen [::]:80;
    server_name {{ joinDomains . }};

    return 301 https://$host$request_uri;
}
{{- end }}

server {
{{- if .SSL }}
    listen 443 ssl;
    listen [::]:443 ssl;
{{- if .Fullchain }}
    ssl_certificate {{ .Fullchain }};
    ssl_certificate_key {{ .Privkey }};
{{- else }}
    # certificate not issued yet — generate one via the Cert panel
{{- end }}
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
{{- else }}
    listen 80;
    listen [::]:80;
{{- end }}
    server_name {{ joinDomains . }};
{{- if .ClientMaxBodySize }}
    client_max_body_size {{ .ClientMaxBodySize }};
{{- end }}
{{- if .WebRoot }}
    root {{ .WebRoot }};
{{- end }}
{{- range .Locations }}
    location {{ .Path }} {
        proxy_pass {{ .ProxyPass }};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
{{- if .Websocket }}
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
{{- end }}
{{- if .Extra }}
{{ indent 1 .Extra }}
{{- end }}
    }
{{- end }}
{{- if .ExtraConfig }}

{{ indent 1 .ExtraConfig }}
{{- end }}
}
`))

type templateData struct {
	*Site
	Fullchain string
	Privkey   string
}

func renderSite(s *Site, fullchain, privkey string) (string, error) {
	var buf bytes.Buffer
	if err := siteTemplate.Execute(&buf, templateData{Site: s, Fullchain: fullchain, Privkey: privkey}); err != nil {
		return "", fmt.Errorf("render nginx config: %w", err)
	}
	return buf.String(), nil
}
