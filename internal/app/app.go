package app

import (
	"context"
	"geoNotifications/internal/di"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run() {
	container, err := di.Build()
	if err != nil {
		log.Fatalf("failed to build dependencies: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := container.Start(ctx); err != nil {
		log.Fatalf("failed to start application: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	container.Stop(shutdownCtx)
	log.Println("Application stopped")
}
