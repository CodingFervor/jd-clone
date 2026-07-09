package handler

import (
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ===================== Lottery wheel (积分大转盘) =====================

// ListLotteryPrizes: GET /lottery/prizes (public) — the wheel segments.
func (h *Handler) ListLotteryPrizes(c *gin.Context) {
	list, err := h.Lottery.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	points := 0
	if uid, ok := h.currentUserID(c, true); ok && uid > 0 {
		points = h.Shop.AvailablePoints(uid)
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "points": points})
}

// SpinLottery: POST /lottery/spin (requires auth) — picks a weighted-random
// prize and deducts the points cost from the user.
func (h *Handler) SpinLottery(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	prizes, err := h.Lottery.List()
	if err != nil || len(prizes) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "奖品未配置"})
		return
	}
	// Determine the cost from the first prize (all segments share the cost).
	cost := prizes[0].PointsCost
	if cost <= 0 {
		cost = 50
	}
	if h.Shop.AvailablePoints(uid) < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "积分不足"})
		return
	}
	// Weighted random selection over the probability weights.
	total := 0
	for _, p := range prizes {
		if p.Probability > 0 {
			total += p.Probability
		}
	}
	won := prizes[0]
	if total > 0 {
		roll := rand.Intn(total)
		cum := 0
		for _, p := range prizes {
			if p.Probability <= 0 {
				continue
			}
			cum += p.Probability
			if roll < cum {
				won = p
				break
			}
		}
	}
	// Deduct the cost by recording a points spend so the spin reduces the
	// user's available balance (mirrors how redemptions subtract points).
	_ = h.Shop.SpendPoints(uid, cost, "lottery:"+won.Name)

	c.JSON(http.StatusOK, gin.H{
		"data":   won,
		"points": h.Shop.AvailablePoints(uid),
	})
}
