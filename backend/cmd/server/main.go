package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/jd-clone/backend/internal/config"
	"github.com/CodingFervor/jd-clone/backend/internal/db"
	"github.com/CodingFervor/jd-clone/backend/internal/handler"
	"github.com/CodingFervor/jd-clone/backend/internal/repository"
	"github.com/CodingFervor/jd-clone/backend/internal/seed"
	"github.com/CodingFervor/jd-clone/backend/internal/server"
)

func main() {
	cfg := config.Load()
	gin.SetMode(gin.ReleaseMode)

	if err := db.Init(cfg.DBPath); err != nil {
		os.Stderr.WriteString("init db: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer db.Close()

	// Ensure the catalog has data on first run (idempotent).
	seed.Run(db.DB)

	h := handler.New(cfg.JWTSecret,
		repository.NewUserRepo(db.DB),
		repository.NewCategoryRepo(db.DB),
		repository.NewProductRepo(db.DB),
		repository.NewCartRepo(db.DB),
		repository.NewOrderRepo(db.DB),
		repository.NewReviewRepo(db.DB),
		repository.NewAddressRepo(db.DB),
	)

	r := server.New(h, cfg.AllowedOrigins)
	addr := ":" + strconv.Itoa(cfg.Port)
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		os.Stderr.WriteString("jd-clone server listening on " + addr + "\n")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			os.Stderr.WriteString("server failed: " + err.Error() + "\n")
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		os.Stderr.WriteString("server shutdown error: " + err.Error() + "\n")
	}
}
