package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
)

// ---- Categories ----

func (h *Handler) ListCategories(c *gin.Context) {
	cats, err := h.Cat.All()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cats})
}

// ---- Products ----

func (h *Handler) ListProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	categoryID, _ := strconv.ParseInt(c.Query("category_id"), 10, 64)
	keyword := c.Query("q")
	items, total, err := h.Product.List(page, pageSize, categoryID, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *Handler) GetProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}
	p, err := h.Product.Get(id)
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品不存在"})
		return
	}
	reviews, _ := h.Review.ListByProduct(id)
	skus := h.SKUsForProduct(id)
	// Record the view in the user's browse history (best-effort; anonymous
	// visitors are skipped by the repo).
	if h.History != nil {
		if uid, ok := h.currentUserID(c, true); ok {
			_ = h.History.RecordView(uid, id)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": p, "reviews": reviews, "skus": skus})
}

func (h *Handler) ListSeckill(c *gin.Context) {
	items, err := h.Product.Seckill(6)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// ---- Cart (requires auth) ----

func (h *Handler) ListCart(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	items, err := h.Cart.List(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	total := 0.0
	for _, it := range items {
		if it.Selected == 1 {
			total += it.Price * float64(it.Quantity)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "total": len(items), "selected_total": total})
}

func (h *Handler) AddCart(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req model.AddCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}
	if err := h.Cart.Add(uid, req.ProductID, req.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "加入购物车失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已加入购物车"})
}

func (h *Handler) UpdateCart(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req model.UpdateCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	if req.Selected != 0 && req.Selected != 1 {
		req.Selected = 1
	}
	if err := h.Cart.Update(id, uid, req.Quantity, req.Selected); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已更新"})
}

func (h *Handler) DeleteCart(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.Cart.Delete(id, uid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ---- Orders (requires auth) ----

func (h *Handler) CreateOrder(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单不能为空"})
		return
	}
	type line struct {
		ProductID int64   `json:"product_id"`
		Name      string  `json:"name"`
		Image     string  `json:"image"`
		Price     float64 `json:"price"`
		Quantity  int     `json:"quantity"`
	}
	var lines []line
	total := 0.0
	for _, it := range req.Items {
		p, _ := h.Product.Get(it.ProductID)
		if p == nil {
			continue
		}
		lines = append(lines, line{ProductID: p.ID, Name: p.Name, Image: p.Image, Price: p.Price, Quantity: it.Quantity})
		total += p.Price * float64(it.Quantity)
	}
	itemsJSON, _ := json.Marshal(lines)
	o := &model.Order{UserID: uid, Total: total, Status: "pending", ItemsJSON: string(itemsJSON), Address: req.Address}
	if err := h.Order.Create(o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "下单失败"})
		return
	}
	_ = h.Cart.Clear(uid)
	c.JSON(http.StatusOK, gin.H{"data": o})
}

func (h *Handler) ListOrders(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	orders, err := h.Order.ListByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

func (h *Handler) PayOrder(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.Order.UpdateStatus(id, uid, "paid"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "支付成功"})
}

// ---- Reviews ----

func (h *Handler) ListReviews(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	reviews, _ := h.Review.ListByProduct(pid)
	c.JSON(http.StatusOK, gin.H{"data": reviews})
}

func (h *Handler) CreateReview(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req model.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	u, _ := h.User.Get(uid)
	uname := ""
	if u != nil {
		uname = u.Nickname
	}
	rv := &model.Review{ProductID: req.ProductID, UserID: uid, Username: uname, Rating: req.Rating, Content: req.Content, Images: req.Images}
	if err := h.Review.Create(rv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评价失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rv})
}

// ---- Addresses ----

func (h *Handler) ListAddresses(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	addrs, err := h.Address.List(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": addrs})
}

// ---- Admin (product CRUD; demo: no role check, same endpoints) ----

func (h *Handler) AdminCreateProduct(c *gin.Context) {
	var p model.ProductInput
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	id, err := h.Product.Create(&p)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "message": "已创建"})
}

func (h *Handler) AdminUpdateProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var p model.ProductInput
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	if err := h.Product.Update(id, &p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已更新"})
}

func (h *Handler) AdminDeleteProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.Product.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
