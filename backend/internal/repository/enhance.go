package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
)

// ===================== SKU =====================

type SKURepo struct{ db *sql.DB }

func NewSKURepo(db *sql.DB) *SKURepo { return &SKURepo{db: db} }

// ListByProduct returns all SKUs of a product.
func (r *SKURepo) ListByProduct(productID int64) ([]model.SKU, error) {
	rows, err := r.db.Query(
		`SELECT id, product_id, spec, spec_text, price, stock, sku_code FROM skus WHERE product_id=? ORDER BY id`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SKU{}
	for rows.Next() {
		var s model.SKU
		if err := rows.Scan(&s.ID, &s.ProductID, &s.Spec, &s.SpecText, &s.Price, &s.Stock, &s.SKUCode); err == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *SKURepo) Get(id int64) (*model.SKU, error) {
	s := &model.SKU{}
	err := r.db.QueryRow(
		`SELECT id, product_id, spec, spec_text, price, stock, sku_code FROM skus WHERE id=?`, id,
	).Scan(&s.ID, &s.ProductID, &s.Spec, &s.SpecText, &s.Price, &s.Stock, &s.SKUCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

// Create adds a SKU variant.
func (r *SKURepo) Create(s *model.SKU) error {
	res, err := r.db.Exec(
		`INSERT INTO skus (product_id, spec, spec_text, price, stock, sku_code) VALUES (?,?,?,?,?,?)`,
		s.ProductID, s.Spec, s.SpecText, s.Price, s.Stock, s.SKUCode)
	if err != nil {
		return err
	}
	s.ID, _ = res.LastInsertId()
	return nil
}

// CountByProduct reports how many SKUs a product has.
func (r *SKURepo) CountByProduct(productID int64) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM skus WHERE product_id=?`, productID).Scan(&n)
	return n, err
}

// Recommend returns the best-value in-stock SKU for a product: the one with the
// lowest unit price that still has stock. Returns nil if none are in stock.
func (r *SKURepo) Recommend(productID int64) (*model.SKU, error) {
	s := &model.SKU{}
	err := r.db.QueryRow(
		`SELECT id, product_id, spec, spec_text, price, stock, sku_code FROM skus
		 WHERE product_id=? AND stock > 0 ORDER BY price ASC, id ASC LIMIT 1`, productID,
	).Scan(&s.ID, &s.ProductID, &s.Spec, &s.SpecText, &s.Price, &s.Stock, &s.SKUCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

// ===================== Payment =====================

type PaymentRepo struct{ db *sql.DB }

func NewPaymentRepo(db *sql.DB) *PaymentRepo { return &PaymentRepo{db: db} }

// GetDB exposes the underlying handle (used by EnhanceHandler to load orders).
func (r *PaymentRepo) GetDB() *sql.DB { return r.db }

// Create records a pending payment attempt for an order and returns it.
func (r *PaymentRepo) Create(orderID, userID int64, amount float64, method string) (*model.Payment, error) {
	if method == "" {
		method = "wechat"
	}
	p := &model.Payment{OrderID: orderID, UserID: userID, Amount: amount, Method: method, Status: "pending"}
	p.TransactionNo = fmt.Sprintf("PAY%d%d", time.Now().Unix(), userID)
	res, err := r.db.Exec(
		`INSERT INTO payments (order_id, user_id, amount, method, transaction_no, status) VALUES (?,?,?,?,?,?)`,
		p.OrderID, p.UserID, p.Amount, p.Method, p.TransactionNo, p.Status)
	if err != nil {
		return nil, err
	}
	p.ID, _ = res.LastInsertId()
	return p, nil
}

// MarkSuccess finalizes a payment: marks the payment row success AND flips the
// order status to paid. This is the "payment success callback" in the sandbox.
func (r *PaymentRepo) MarkSuccess(paymentID int64) (*model.Payment, error) {
	// load row
	p := &model.Payment{}
	if err := r.db.QueryRow(
		`SELECT id, order_id, user_id, amount, method, transaction_no, status FROM payments WHERE id=?`, paymentID,
	).Scan(&p.ID, &p.OrderID, &p.UserID, &p.Amount, &p.Method, &p.TransactionNo, &p.Status); err != nil {
		return nil, err
	}
	if p.Status == "success" {
		return p, nil // idempotent
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE payments SET status='success' WHERE id=?`, paymentID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE orders SET status='paid' WHERE id=?`, p.OrderID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	p.Status = "success"
	return p, nil
}

// GetByOrder returns the (latest) payment for an order.
func (r *PaymentRepo) GetByOrder(orderID int64) (*model.Payment, error) {
	p := &model.Payment{}
	err := r.db.QueryRow(
		`SELECT id, order_id, user_id, amount, method, transaction_no, status FROM payments WHERE order_id=? ORDER BY id DESC LIMIT 1`, orderID,
	).Scan(&p.ID, &p.OrderID, &p.UserID, &p.Amount, &p.Method, &p.TransactionNo, &p.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// ===================== Shipment =====================

type ShipmentRepo struct{ db *sql.DB }

func NewShipmentRepo(db *sql.DB) *ShipmentRepo { return &ShipmentRepo{db: db} }

// CreateForOrder generates a shipment (tracking number + initial tracks) for an
// order, simulating the seller shipping it. The order is moved to "shipped".
func (r *ShipmentRepo) CreateForOrder(orderID int64) (*model.Shipment, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	trackingNo := fmt.Sprintf("JD%s%06d", time.Now().Format("20060102"), orderID%1000000)
	res, err := tx.Exec(
		`INSERT INTO shipments (order_id, tracking_no, carrier, status) VALUES (?,?,?,?)`,
		orderID, trackingNo, "京东快递", "shipped")
	if err != nil {
		return nil, err
	}
	shipID, _ := res.LastInsertId()

	// Seed a realistic multi-step trajectory (shipped → transit → out for delivery).
	seedTracks := []struct{ desc, loc string }{
		{"【京东物流】您的订单已出库，京东快递员已取件", "京东上海亚洲一号仓"},
		{"【京东物流】快件已到达【上海中转场】", "上海中转场"},
		{"【京东物流】快件已从上海发往北京分拨中心", "上海→北京"},
		{"【京东物流】快件已到达北京分拨中心，正在分拣", "北京分拨中心"},
	}
	now := time.Now().Add(-24 * time.Hour) // backdate the first event a day ago
	for i, t := range seedTracks {
		occurred := now.Add(time.Duration(i) * 6 * time.Hour)
		_, _ = tx.Exec(
			`INSERT INTO shipment_tracks (shipment_id, description, location, occurred_at) VALUES (?,?,?,?)`,
			shipID, t.desc, t.loc, occurred)
	}
	if _, err := tx.Exec(`UPDATE orders SET status='shipped' WHERE id=?`, orderID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByOrder(orderID)
}

// GetByOrder returns the shipment + its track list for an order.
func (r *ShipmentRepo) GetByOrder(orderID int64) (*model.Shipment, error) {
	s := &model.Shipment{}
	err := r.db.QueryRow(
		`SELECT id, order_id, tracking_no, carrier, status, created_at FROM shipments WHERE order_id=?`, orderID,
	).Scan(&s.ID, &s.OrderID, &s.TrackingNo, &s.Carrier, &s.Status, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tracks, _ := r.listTracks(s.ID)
	s.Tracks = tracks
	return s, nil
}

// TrackByNo returns a shipment by tracking number.
func (r *ShipmentRepo) TrackByNo(no string) (*model.Shipment, error) {
	s := &model.Shipment{}
	err := r.db.QueryRow(
		`SELECT id, order_id, tracking_no, carrier, status, created_at FROM shipments WHERE tracking_no=?`, no,
	).Scan(&s.ID, &s.OrderID, &s.TrackingNo, &s.Carrier, &s.Status, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tracks, _ := r.listTracks(s.ID)
	s.Tracks = tracks
	return s, nil
}

func (r *ShipmentRepo) listTracks(shipmentID int64) ([]model.Track, error) {
	rows, err := r.db.Query(
		`SELECT id, shipment_id, description, location, occurred_at FROM shipment_tracks WHERE shipment_id=? ORDER BY occurred_at DESC`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Track{}
	for rows.Next() {
		var t model.Track
		if err := rows.Scan(&t.ID, &t.ShipmentID, &t.Description, &t.Location, &t.OccurredAt); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

// AdvanceStatus moves a shipment forward in its lifecycle and appends a track
// event, simulating real logistics updates.
func (r *ShipmentRepo) AdvanceStatus(orderID int64) (*model.Shipment, error) {
	s, err := r.GetByOrder(orderID)
	if err != nil || s == nil {
		return nil, fmt.Errorf("shipment not found")
	}
	next := map[string]string{"shipped": "in_transit", "in_transit": "delivered"}
	nextDesc := map[string]string{
		"in_transit": "【京东物流】快件已派送，京东小哥正在为您送货",
		"delivered":  "【京东物流】您的订单已送达，感谢使用京东，欢迎再次光临",
	}
	ns, ok := next[s.Status]
	if !ok {
		return s, nil // already delivered, nothing to advance
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE shipments SET status=? WHERE id=?`, ns, s.ID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(
		`INSERT INTO shipment_tracks (shipment_id, description, location, occurred_at) VALUES (?,?,?,?)`,
		s.ID, nextDesc[ns], "配送中", time.Now()); err != nil {
		return nil, err
	}
	if ns == "delivered" {
		if _, err := tx.Exec(`UPDATE orders SET status='completed' WHERE id=?`, orderID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByOrder(orderID)
}
