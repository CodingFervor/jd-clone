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
	// Attach the SKU / Payment / Shipment enhancement repos.
	h.SetEnhance(
		repository.NewSKURepo(db.DB),
		repository.NewPaymentRepo(db.DB),
		repository.NewShipmentRepo(db.DB),
	)
	// Attach the Refund / Coupon after-sale repos.
	h.SetAfterSale(
		repository.NewRefundRepo(db.DB),
		repository.NewCouponRepo(db.DB),
	)
	// Attach the favorite (wishlist) repo.
	h.SetUserExtra(repository.NewFavoriteRepo(db.DB))
	// Attach the browse-history + check-in + points-mall repos.
	shopRepo := repository.NewPointShopRepo(db.DB)
	shopRepo.SeedPointShop()
	h.SetHistory(repository.NewHistoryRepo(db.DB), repository.NewCheckInRepo(db.DB), shopRepo)
	// Attach + seed the flash-sale (秒杀) repo.
	seckillRepo := repository.NewSeckillRepo(db.DB)
	seckillRepo.SeedSeckill()
	h.SetSeckill(seckillRepo)
	// Attach + seed the group-buy (拼团) repo.
	groupBuyRepo := repository.NewGroupBuyRepo(db.DB)
	groupBuyRepo.SeedGroupBuys()
	h.SetGroupBuy(groupBuyRepo)
	// Attach + seed the presale (预售定金) repo.
	presaleRepo := repository.NewPresaleRepo(db.DB)
	presaleRepo.SeedPresales()
	h.SetPresale(presaleRepo)
	// Attach + seed the price-history (比价历史) repo.
	priceHistoryRepo := repository.NewPriceHistoryRepo(db.DB)
	priceHistoryRepo.SeedPriceHistory()
	h.SetPriceHistory(priceHistoryRepo)

	// Ensure the images directory exists for the static file server.
	_ = os.MkdirAll("data/images", 0o755)

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
