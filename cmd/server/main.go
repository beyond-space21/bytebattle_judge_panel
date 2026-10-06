package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/judge/competition/internal/api"
	"github.com/judge/competition/internal/auth"
	"github.com/judge/competition/internal/config"
	"github.com/judge/competition/internal/judge"
	"github.com/judge/competition/internal/store"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer st.Close()

	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		log.Fatalf("hash admin password: %v", err)
	}
	if err := st.EnsureAdmin(ctx, cfg.AdminUsername, hash); err != nil {
		log.Fatalf("ensure admin: %v", err)
	}
	log.Printf("admin user ready: %s", cfg.AdminUsername)

	staticFS := locateUI()
	j := judge.New(cfg.Judge0URL, cfg.PythonLanguageID)
	srv := api.New(cfg, st, j, staticFS)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func locateUI() fs.FS {
	candidates := []string{"web/dist", "dist"}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append([]string{filepath.Join(dir, "web/dist"), filepath.Join(dir, "dist")}, candidates...)
	}
	for _, c := range candidates {
		if info, err := os.Stat(filepath.Join(c, "index.html")); err == nil && !info.IsDir() {
			log.Printf("serving web UI from %s", c)
			return os.DirFS(c)
		}
	}
	log.Printf("warning: no web UI found (run: make web)")
	return nil
}
