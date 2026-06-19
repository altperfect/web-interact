package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webhook-site/internal/app"
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := app.OpenStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := store.Cleanup(ctx, cfg.RetentionDays); err != nil {
		log.Printf("cleanup: %v", err)
	}

	svc := app.New(cfg, store)
	go svc.RunCleanup(ctx)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           svc,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errs := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", cfg.Addr)
		errs <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-errs:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
