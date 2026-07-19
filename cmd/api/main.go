package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/DGreegman/vaultpay/internal/config"
	"github.com/DGreegman/vaultpay/internal/db"
	"github.com/DGreegman/vaultpay/internal/server"
	"github.com/DGreegman/vaultpay/internal/session"
	"github.com/DGreegman/vaultpay/internal/token"
	"github.com/DGreegman/vaultpay/internal/user"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.New(ctx, db.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatalf("Failed, to connect to database: %v", err)
	}

	defer pool.Close()

	userRepo := user.NewPostgresRepository(pool)
	userService := user.NewService(userRepo)
	tokenManager := token.NewManager(cfg.JWTSecret)
	sessionRepo := session.NewPostgresRepository(pool)
	sessionService := session.NewService(sessionRepo)

	srv := server.New(cfg, pool, userService, tokenManager, sessionService)

	// Run the server in a goroutine so main can waiit for signal
	go func () {
		if err := srv.Listen(); err != nil {
			log.Fatalf("server failed: %v", err)
		}
	}()

	log.Printf("Vaultpay api is listening on : %s (env=%s)", cfg.Port, cfg.AppEnv)

	// Block until we receive a shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<- quit

	log.Println("Shutdown down...")

	if err := srv.Shutdown(); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}

	log.Println("Shutdown Complete")
}