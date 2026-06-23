package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/config"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/server"
)

func main() {
	cfg := config.Default()

	srv := server.NewServer(cfg)

	httpServer := &http.Server{
		Addr:    cfg.Port,
		Handler: srv.Handler(),
	}

	// Start server in a goroutine
	go func() {
		log.Printf("hfs-v2 starting on %s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}

	log.Println("server stopped")
}
