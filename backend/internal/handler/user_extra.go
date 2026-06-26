package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
)

// ===================== Favorites (wishlist) =====================

// ToggleFavorite adds or removes a favorite for the current user.
func (h *Handler) ToggleFavorite(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}
	if p, _ := h.Product.Get(pid); p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品不存在"})
		return
	}
	faved, err := h.Favorite.Toggle(uid, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorited": faved})
}

// ListFavorites returns the current user's favorited products.
func (h *Handler) ListFavorites(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	list, err := h.Favorite.ListByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// CheckFavorite reports whether a product is favorited by the current user.
func (h *Handler) CheckFavorite(c *gin.Context) {
	uid, ok := h.currentUserID(c, true)
	if !ok {
		return
	}
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorited": h.Favorite.IsFavorited(uid, pid)})
}

// ===================== Address CRUD =====================

func (h *Handler) CreateAddress(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req model.AddressInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	a := &model.Address{UserID: uid, Name: req.Name, Phone: req.Phone, Detail: req.Detail, IsDefault: req.IsDefault}
	if err := h.Address.Create(a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "添加失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": a})
}

func (h *Handler) UpdateAddress(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req model.AddressInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	a := &model.Address{ID: id, UserID: uid, Name: req.Name, Phone: req.Phone, Detail: req.Detail, IsDefault: req.IsDefault}
	if err := h.Address.Update(a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已更新"})
}

func (h *Handler) DeleteAddress(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.Address.Delete(id, uid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ===================== Confirm receipt (order lifecycle) =====================

// ConfirmOrder marks a delivered/shipped order as completed (确认收货).
func (h *Handler) ConfirmOrder(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的订单ID"})
		return
	}
	if err := h.Order.ConfirmReceipt(id, uid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Mirror the delivery status on the shipment record if present.
	if s, _ := h.Shipment.GetByOrder(id); s != nil {
		_, _ = h.Shipment.AdvanceStatus(id)
	}
	c.JSON(http.StatusOK, gin.H{"message": "已确认收货"})
}

// ===================== Edit profile =====================

func (h *Handler) UpdateProfile(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req model.ProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	// Carry forward existing fields so partial edits don't blank them.
	u, err := h.User.Get(uid)
	if err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if req.Nickname != "" {
		u.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		u.Avatar = req.Avatar
	}
	if req.Phone != "" {
		u.Phone = req.Phone
	}
	if err := h.User.UpdateProfile(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": u})
}

// ===================== Browse history =====================

func (h *Handler) ListHistory(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	list, err := h.History.ListByUser(uid, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) ClearHistory(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if err := h.History.Clear(uid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "清除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已清除"})
}

// ===================== Daily check-in =====================

func (h *Handler) DoCheckIn(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	ci, isNew, err := h.CheckIn.CheckIn(uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "签到失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ci, "is_new": isNew})
}

func (h *Handler) CheckInStatus(c *gin.Context) {
	uid, ok := h.currentUserID(c, true)
	if !ok {
		return
	}
	last, total, err := h.CheckIn.Status(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"last": last, "total_points": total})
}
