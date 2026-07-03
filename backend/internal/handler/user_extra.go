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

// ===================== Points mall (积分商城) =====================

func (h *Handler) ListPointShop(c *gin.Context) {
	uid, _ := h.currentUserID(c, true)
	list, err := h.Shop.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	points := 0
	if uid > 0 {
		points = h.Shop.AvailablePoints(uid)
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "points": points})
}

func (h *Handler) RedeemPoints(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	rd, err := h.Shop.Redeem(uid, pid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rd, "points": h.Shop.AvailablePoints(uid)})
}

func (h *Handler) ListRedemptions(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	list, err := h.Shop.ListRedemptions(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ===================== Review replies =====================

func (h *Handler) ReplyReview(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		ReviewID int64  `json:"review_id" binding:"required"`
		Content  string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	u, _ := h.User.Get(uid)
	name := ""
	if u != nil {
		name = u.Nickname
		if name == "" {
			name = u.Username
		}
	}
	rep := &model.ReviewReply{ReviewID: req.ReviewID, UserID: uid, Username: name, Content: req.Content}
	if err := h.Review.AddReply(rep); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "回复失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rep})
}

// ===================== Seckill deals (秒杀活动) =====================

// ListSeckillDeals: GET /seckill (public)
func (h *Handler) ListSeckillDeals(c *gin.Context) {
	deals, err := h.Seckill.ListActive(10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": deals})
}

// GrabSeckill: POST /seckill/:id/grab — atomic flash-sale purchase (requires auth).
func (h *Handler) GrabSeckill(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	deal, err := h.Seckill.Grab(id, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": deal, "message": "抢购成功"})
}

// ===================== Group buys (拼团) =====================

// ListGroupBuys: GET /group-buys (public)
func (h *Handler) ListGroupBuys(c *gin.Context) {
	deals, err := h.GroupBuy.ListActive(10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": deals})
}

// JoinGroupBuy: POST /group-buys/:id/join — join a team purchase (requires auth).
func (h *Handler) JoinGroupBuy(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	g, err := h.GroupBuy.Join(id, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	msg := "参团成功"
	if g.Status == "success" {
		msg = "拼团成功！"
	}
	c.JSON(http.StatusOK, gin.H{"data": g, "message": msg})
}

// ===================== Presales (预售定金) =====================

// ListPresales: GET /presales (public)
func (h *Handler) ListPresales(c *gin.Context) {
	list, err := h.Presale.ListActive(10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// PayPresaleDeposit: POST /presales/:id/deposit — pay the deposit (requires auth).
func (h *Handler) PayPresaleDeposit(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	po, err := h.Presale.PayDeposit(id, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": po, "message": "定金支付成功"})
}

// ===================== Price history (比价历史) =====================

// ListPriceHistory: GET /products/:id/price-history (public)
func (h *Handler) ListPriceHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}
	list, err := h.PriceHistory.ListByProduct(id, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	// Compute trend stats: lowest, highest, current.
	stats := gin.H{"lowest": 0.0, "highest": 0.0, "current": 0.0}
	if len(list) > 0 {
		low, high := list[0].Price, list[0].Price
		for _, p := range list {
			if p.Price < low {
				low = p.Price
			}
			if p.Price > high {
				high = p.Price
			}
		}
		stats = gin.H{"lowest": low, "highest": high, "current": list[len(list)-1].Price}
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "stats": stats})
}
