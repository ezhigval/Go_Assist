package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"modulr/controlplane"
)

func main() {
	addr := getEnv("CONTROL_PLANE_ADDR", ":8080")
	statePath := getEnv("CONTROL_PLANE_STATE_PATH", "data/controlplane/snapshot.json")
	service, err := controlplane.NewPersistentService(statePath)
	if err != nil {
		log.Fatalf("controlplane: init service failed: %v", err)
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           controlplane.NewHandler(service),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("controlplane: listening on %s (state: %s)", addr, statePath)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("controlplane: shutdown signal %s", sig)
	case err := <-errCh:
		log.Fatalf("controlplane: listen failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("controlplane: shutdown failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
