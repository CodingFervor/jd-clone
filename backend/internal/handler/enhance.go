package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
	"github.com/CodingFervor/jd-clone/backend/internal/repository"
)

// Enhance methods are added to the existing Handler (which already holds the JWT
// secret + currentUserID) so payment/shipment endpoints can reuse auth.

// SetEnhance attaches the SKU/Payment/Shipment repos to the handler.
func (h *Handler) SetEnhance(sku *repository.SKURepo, pay *repository.PaymentRepo, ship *repository.ShipmentRepo) {
	h.SKU = sku
	h.Payment = pay
	h.Shipment = ship
}

// ---- SKU endpoints ----

// ListSKUs: GET /products/:id/skus
func (h *Handler) ListSKUs(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}
	skus, err := h.SKU.ListByProduct(pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": skus})
}

// CreateSKU: POST /admin/products/:id/skus
func (h *Handler) CreateSKU(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}
	var s model.SKU
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	s.ProductID = pid
	if err := h.SKU.Create(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s})
}

// SKUsForProduct returns SKUs for embedding in a product detail response.
func (h *Handler) SKUsForProduct(productID int64) []model.SKU {
	if h.SKU == nil {
		return nil
	}
	skus, _ := h.SKU.ListByProduct(productID)
	return skus
}

// ---- Payment endpoints ----

// CreatePayment: POST /payments  { order_id, method }
func (h *Handler) CreatePayment(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		OrderID int64  `json:"order_id" binding:"required"`
		Method  string `json:"method"`
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
	if o.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单状态不支持支付"})
		return
	}
	p, err := h.Payment.Create(req.OrderID, uid, o.Total, req.Method)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建支付失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"payment": p,
		"pay_url": "/api/v1/payments/" + strconv.FormatInt(p.ID, 10) + "/confirm",
		"message": "沙箱模式：调用 confirm 接口完成支付",
	})
}

// ConfirmPayment: POST /payments/:id/confirm
func (h *Handler) ConfirmPayment(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的支付ID"})
		return
	}
	p, err := h.Payment.MarkSuccess(pid)
	if err != nil || p == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "支付确认失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"payment": p, "message": "支付成功"})
}

// GetPayment: GET /payments/order/:id
func (h *Handler) GetPayment(c *gin.Context) {
	oid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的订单ID"})
		return
	}
	p, err := h.Payment.GetByOrder(oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"payment": p})
}

// ---- Shipment endpoints ----

// ShipOrder: POST /orders/:id/ship
func (h *Handler) ShipOrder(c *gin.Context) {
	oid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的订单ID"})
		return
	}
	// Load the order without the user-id scope (admin/seller action).
	o, err := h.Order.Get(oid, 0)
	if err != nil || o == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}
	if o.Status != "paid" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单尚未支付，不能发货"})
		return
	}
	s, err := h.Shipment.CreateForOrder(oid)
	if err != nil || s == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "发货失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipment": s, "message": "已发货"})
}

// TrackOrder: GET /orders/:id/track
func (h *Handler) TrackOrder(c *gin.Context) {
	oid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的订单ID"})
		return
	}
	s, err := h.Shipment.GetByOrder(oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipment": s})
}

// TrackByNo: GET /shipments/track?no=XXX
func (h *Handler) TrackByNo(c *gin.Context) {
	no := c.Query("no")
	if no == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供运单号"})
		return
	}
	s, err := h.Shipment.TrackByNo(no)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "运单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipment": s})
}

// AdvanceShipment: POST /orders/:id/ship/advance
func (h *Handler) AdvanceShipment(c *gin.Context) {
	oid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的订单ID"})
		return
	}
	s, err := h.Shipment.AdvanceStatus(oid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"shipment": s})
}
