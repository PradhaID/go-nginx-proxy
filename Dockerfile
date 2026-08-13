# syntax=docker/dockerfile:1
FROM node:22-alpine AS web
WORKDIR /src
COPY web/package*.json ./
RUN npm install
COPY web/ .
RUN npm run build

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY --from=web /internal/web/dist ./internal/web/dist
RUN go build -o /go-nginx-proxy ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache nginx certbot py3-pip \
    && pip install --no-cache-dir certbot-dns-cloudflare \
    && mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled /etc/letsencrypt
COPY --from=build /go-nginx-proxy /usr/local/bin/go-nginx-proxy
# make sure nginx loads configs from sites-enabled
COPY deploy/nginx.conf /etc/nginx/http.d/default.conf
ENV PROXY_ADDR=:8080 \
    PROXY_NGINX_AVAILABLE_DIR=/etc/nginx/sites-available \
    PROXY_NGINX_ENABLED_DIR=/etc/nginx/sites-enabled \
    PROXY_USE_SYSTEMCTL=false
EXPOSE 8080
CMD ["go-nginx-proxy"]
