package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/jd-clone/backend/internal/handler"
)

// New builds the gin engine with the full route table.
func New(h *handler.Handler, allowedOrigins string) *gin.Engine {
	r := gin.Default()
	r.Use(corsMiddleware(allowedOrigins))

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Serve uploaded images from the local data/images directory.
	r.Static("/images", "data/images")

	api := r.Group("/api/v1")
	{
		// Public auth
		api.POST("/auth/login", h.Login)
		api.POST("/auth/register", h.Register)

		// Public catalog
		api.GET("/categories", h.ListCategories)
		api.GET("/products", h.ListProducts)
		api.GET("/products/seckill", h.ListSeckill)
		api.GET("/products/:id", h.GetProduct)
		api.GET("/products/:id/reviews", h.ListReviews)
		api.GET("/products/:id/skus", h.ListSKUs)
		api.GET("/products/:id/price-history", h.ListPriceHistory)
		api.GET("/products/:id/restock", h.CheckRestock)
		api.GET("/products/:id/qa", h.ListQA)

		// Public bundles + shop ratings
		api.GET("/bundles", h.ListBundles)
		api.GET("/shops/:name/ratings", h.ShopRatingSummary)

		// Public FTS5 search + suggest
		api.GET("/search", h.FtsSearch)
		api.GET("/search/suggest", h.FtsSuggest)

		// Public flash-sale (秒杀) listing
		api.GET("/seckill", h.ListSeckillDeals)

		// Public group-buy (拼团) listing
		api.GET("/group-buys", h.ListGroupBuys)

		// Public presale (预售定金) listing
		api.GET("/presales", h.ListPresales)

		// Public shipment tracking by tracking number
		api.GET("/shipments/track", h.TrackByNo)

		// Authenticated (the handler validates the token per-request)
		auth := api.Group("/")
		auth.Use(authMiddleware(h))
		{
			auth.GET("/auth/profile", h.Profile)

			// Shop rating create
			auth.POST("/shops/:name/ratings", h.CreateShopRating)

			// Restock alerts (到货通知)
			auth.POST("/products/:id/restock", h.SubscribeRestock)
			auth.DELETE("/products/:id/restock", h.UnsubscribeRestock)

			// Product Q&A (商品问答)
			auth.POST("/products/:id/qa", h.AskQA)
			auth.POST("/qa/:id/answer", h.AnswerQA)

			auth.GET("/cart", h.ListCart)
			auth.POST("/cart", h.AddCart)
			auth.PUT("/cart/:id", h.UpdateCart)
			auth.DELETE("/cart/:id", h.DeleteCart)

			auth.GET("/orders", h.ListOrders)
			auth.POST("/orders", h.CreateOrder)
			auth.POST("/orders/:id/pay", h.PayOrder)

			// Payment flow (sandbox)
			auth.POST("/payments", h.CreatePayment)
			auth.POST("/payments/:id/confirm", h.ConfirmPayment)
			auth.GET("/payments/order/:id", h.GetPayment)

			// Shipment flow
			auth.POST("/orders/:id/ship", h.ShipOrder)
			auth.GET("/orders/:id/track", h.TrackOrder)
			auth.POST("/orders/:id/ship/advance", h.AdvanceShipment)

			auth.POST("/reviews", h.CreateReview)

			auth.GET("/addresses", h.ListAddresses)
			auth.POST("/addresses", h.CreateAddress)
			auth.PUT("/addresses/:id", h.UpdateAddress)
			auth.DELETE("/addresses/:id", h.DeleteAddress)

			// Image upload (used by admin product form + review photos)
			auth.POST("/upload", h.UploadImage)

			// Favorites / wishlist
			auth.GET("/favorites", h.ListFavorites)
			auth.POST("/favorites/:id", h.ToggleFavorite)
			auth.GET("/favorites/:id/check", h.CheckFavorite)

			// Flash-sale grab (秒杀抢购)
			auth.POST("/seckill/:id/grab", h.GrabSeckill)

			// Group buy join (拼团参团)
			auth.POST("/group-buys/:id/join", h.JoinGroupBuy)

			// Presale deposit (预售定金)
			auth.POST("/presales/:id/deposit", h.PayPresaleDeposit)

			// Browse history
			auth.GET("/history", h.ListHistory)
			auth.DELETE("/history", h.ClearHistory)

			// Daily check-in
			auth.POST("/checkin", h.DoCheckIn)
			auth.GET("/checkin/status", h.CheckInStatus)

			// Points mall (积分商城)
			auth.GET("/points/shop", h.ListPointShop)
			auth.POST("/points/shop/:id/redeem", h.RedeemPoints)
			auth.GET("/points/redemptions", h.ListRedemptions)

			// Review replies
			auth.POST("/reviews/reply", h.ReplyReview)
			auth.POST("/reviews/:id/useful", h.MarkReviewUseful)

			// Edit profile
			auth.PUT("/auth/profile", h.UpdateProfile)

			// Confirm receipt (order lifecycle)
			auth.POST("/orders/:id/confirm", h.ConfirmOrder)
			// Cancel order (取消订单)
			auth.POST("/orders/:id/cancel", h.CancelOrder)

			// After-sale refunds
			auth.POST("/refunds", h.CreateRefund)
			auth.GET("/refunds", h.ListRefunds)
			auth.POST("/refunds/:id/approve", h.ApproveRefund)

			// Coupons
			auth.GET("/coupons", h.ListCoupons)
			auth.POST("/coupons/:id/claim", h.ClaimCoupon)
			auth.GET("/coupons/mine", h.ListMyCoupons)

			// Admin product management
			auth.POST("/admin/products", h.AdminCreateProduct)
			auth.PUT("/admin/products/:id", h.AdminUpdateProduct)
			auth.DELETE("/admin/products/:id", h.AdminDeleteProduct)
			auth.POST("/admin/products/:id/skus", h.CreateSKU)
		}
	}
	return r
}

// authMiddleware ensures a valid Bearer token is present. It does NOT enforce
// roles — admin endpoints are guarded only by token presence (demo).
func authMiddleware(h *handler.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}
		c.Next()
	}
}

func corsMiddleware(allowed string) gin.HandlerFunc {
	allowAll := strings.TrimSpace(allowed) == "*" || allowed == ""
	origins := map[string]bool{}
	for _, o := range strings.Split(allowed, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = true
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		switch {
		case allowAll:
			c.Header("Access-Control-Allow-Origin", "*")
		case origin != "" && origins[origin]:
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
