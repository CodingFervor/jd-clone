package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/jd-clone/backend/internal/repository"
)

// ===================== Gift cards (礼品卡) =====================

// SetGiftCard attaches the gift-card (礼品卡) repo.
func (h *Handler) SetGiftCard(gc *repository.GiftCardRepo) {
	h.GiftCard = gc
}

// GenerateGiftCard: POST /gift-cards/generate (admin) — issues a new card.
// Body: { "amount": 100.0 }
func (h *Handler) GenerateGiftCard(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入有效的礼品卡面值"})
		return
	}
	gc, err := h.GiftCard.Generate(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gc, "message": "礼品卡已生成"})
}

// RedeemGiftCard: POST /gift-cards/redeem (auth) — redeems a card for the user.
// Body: { "code": "ABCD2345..." }
func (h *Handler) RedeemGiftCard(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入礼品卡卡号"})
		return
	}
	gc, err := h.GiftCard.Redeem(req.Code, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gc, "message": "兑换成功"})
}

// ListGiftCards: GET /gift-cards (auth) — the calling user's redeemed cards.
func (h *Handler) ListGiftCards(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	list, err := h.GiftCard.ListByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
