package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
	"github.com/CodingFervor/jd-clone/backend/internal/repository"
)

// SetAfterSale attaches the refund + coupon repos.
func (h *Handler) SetAfterSale(refund *repository.RefundRepo, coupon *repository.CouponRepo) {
	h.Refund = refund
	h.Coupon = coupon
}

// ---- Refund ----

func (h *Handler) CreateRefund(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		OrderID int64  `json:"order_id" binding:"required"`
		Reason  string `json:"reason" binding:"required"`
		Type    string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	o, err := h.Order.Get(req.OrderID, uid)
	if err != nil || o == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}
	rf := &model.Refund{OrderID: req.OrderID, UserID: uid, Reason: req.Reason, Type: req.Type, Amount: o.Total, Status: "pending"}
	if err := h.Refund.Create(rf); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "申请失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rf})
}

func (h *Handler) ListRefunds(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	list, err := h.Refund.ListByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ApproveRefund is an admin action to approve/reject a refund.
func (h *Handler) ApproveRefund(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	if err := h.Refund.UpdateStatus(id, req.Status, req.Note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已处理"})
}

// ---- Coupon ----

func (h *Handler) ListCoupons(c *gin.Context) {
	uid, _ := h.currentUserID(c, true)
	list, err := h.Coupon.ListAvailable(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) ClaimCoupon(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	cid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.Coupon.Claim(uid, cid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "领取成功"})
}

func (h *Handler) ListMyCoupons(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	list, err := h.Coupon.ListUserCoupons(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ---- FTS5 Search ----

func (h *Handler) FtsSearch(c *gin.Context) {
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"data": []any{}, "total": 0})
		return
	}
	ids, err := h.Refund.SearchProducts(q, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索失败"})
		return
	}
	// Load product rows for the matched IDs.
	products := []model.Product{}
	for _, id := range ids {
		p, _ := h.Product.Get(id)
		if p != nil {
			products = append(products, *p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": products, "total": len(products)})
}

func (h *Handler) FtsSuggest(c *gin.Context) {
	prefix := c.Query("q")
	if prefix == "" {
		c.JSON(http.StatusOK, gin.H{"data": []string{}})
		return
	}
	suggestions, err := h.Refund.Suggest(prefix, 10)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": suggestions})
}
