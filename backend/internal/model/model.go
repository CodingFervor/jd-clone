package model

import "time"

// User is a shopper account.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}

// Category groups products.
type Category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
}

// Product is a sellable item.
type Product struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Subtitle      string    `json:"subtitle"`
	Price         float64   `json:"price"`
	OriginalPrice float64   `json:"original_price"`
	Image         string    `json:"image"`
	Images        string    `json:"images"` // comma-separated gallery
	Category      string    `json:"category"`
	CategoryID    int64     `json:"category_id"`
	Shop          string    `json:"shop"`
	Stock         int       `json:"stock"`
	Sales         int       `json:"sales"`
	Description   string    `json:"description"`
	Tags          string    `json:"tags"`
	IsSeckill     int       `json:"is_seckill"`
	CreatedAt     time.Time `json:"created_at"`
}

// CartItem is an entry in a user's shopping cart.
type CartItem struct {
	ID        int64 `json:"id"`
	UserID    int64 `json:"user_id"`
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
	Selected  int   `json:"selected"`
	// Joined product fields (populated for list responses).
	ProductName string    `json:"product_name" db:"p_name"`
	ProductImg  string    `json:"product_image" db:"p_image"`
	Price       float64   `json:"price" db:"p_price"`
	Stock       int       `json:"stock" db:"p_stock"`
	CreatedAt   time.Time `json:"created_at"`
}

// Order is a placed purchase.
type Order struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	OrderNo   string    `json:"order_no"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"` // pending, paid, shipped, completed, cancelled
	ItemsJSON string    `json:"items_json"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

// Review is a buyer's rating of a product.
type Review struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Rating    int       `json:"rating"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Address is a shipping destination.
type Address struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Detail    string `json:"detail"`
	IsDefault int    `json:"is_default"`
}

// SKU is a specific spec combination (color/size/version) of a product.
type SKU struct {
	ID        int64   `json:"id"`
	ProductID int64   `json:"product_id"`
	Spec      string  `json:"spec"`      // JSON, e.g. {"颜色":"黑色","版本":"256GB"}
	SpecText  string  `json:"spec_text"` // display string "黑色 256GB"
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	SKUCode   string  `json:"sku_code"`
}

// Shipment is a shipped order's logistics envelope.
type Shipment struct {
	ID         int64     `json:"id"`
	OrderID    int64     `json:"order_id"`
	TrackingNo string    `json:"tracking_no"`
	Carrier    string    `json:"carrier"`
	Status     string    `json:"status"` // pending, shipped, in_transit, delivered
	Tracks     []Track   `json:"tracks"`
	CreatedAt  time.Time `json:"created_at"`
}

// Track is one logistics event.
type Track struct {
	ID          int64     `json:"id"`
	ShipmentID  int64     `json:"shipment_id"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// Payment records a payment attempt for an order.
type Payment struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	UserID        int64     `json:"user_id"`
	Amount        float64   `json:"amount"`
	Method        string    `json:"method"` // wechat, alipay, unionpay
	TransactionNo string    `json:"transaction_no"`
	Status        string    `json:"status"` // pending, success, failed
	CreatedAt     time.Time `json:"created_at"`
}

// ---- Request DTOs ----

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
}

type AddCartRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity"`
}

type UpdateCartRequest struct {
	Quantity int `json:"quantity"`
	Selected int `json:"selected"`
}

type CreateOrderRequest struct {
	Items   []OrderItemInput `json:"items" binding:"required"`
	Address string           `json:"address"`
}
type OrderItemInput struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity" binding:"required"`
}

type CreateReviewRequest struct {
	ProductID int64  `json:"product_id" binding:"required"`
	Rating    int    `json:"rating"`
	Content   string `json:"content"`
}

type ProductInput struct {
	Name          string  `json:"name" binding:"required"`
	Subtitle      string  `json:"subtitle"`
	Price         float64 `json:"price" binding:"required"`
	OriginalPrice float64 `json:"original_price"`
	Image         string  `json:"image"`
	Images        string  `json:"images"`
	Category      string  `json:"category"`
	CategoryID    int64   `json:"category_id"`
	Shop          string  `json:"shop"`
	Stock         int     `json:"stock"`
	Sales         int     `json:"sales"`
	Description   string  `json:"description"`
	Tags          string  `json:"tags"`
	IsSeckill     int     `json:"is_seckill"`
}
