package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/PradhaID/go-nginx-proxy/internal/api"
	"github.com/PradhaID/go-nginx-proxy/internal/config"
	"github.com/PradhaID/go-nginx-proxy/internal/web"
)

func main() {
	cfg := config.Load()

	if cfg.Password == "" {
		log.Println("warning: PROXY_PASSWORD is not set, authentication disabled")
	}
	if cfg.Password == "changeme" {
		log.Println("warning: using default password (changeme), set PROXY_PASSWORD")
	}

	var static http.Handler
	if cfg.WebDir != "" {
		abs, err := filepath.Abs(cfg.WebDir)
		if err != nil {
			log.Fatalf("invalid PROXY_WEB_DIR: %v", err)
		}
		log.Printf("serving static files from %s", abs)
		static = spaHandler(abs)
	} else {
		static = web.Handler()
	}

	srv := api.New(cfg, static)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("go-nginx-proxy listening on %s", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func spaHandler(dir string) http.Handler {
	fs := http.Dir(dir)
	fileServer := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, r.URL.Path)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
