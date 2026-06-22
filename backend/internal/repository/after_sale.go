package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
)

// ===================== Refund =====================

type RefundRepo struct{ db *sql.DB }

func NewRefundRepo(db *sql.DB) *RefundRepo { return &RefundRepo{db: db} }

func (r *RefundRepo) Create(rf *model.Refund) error {
	res, err := r.db.Exec(
		`INSERT INTO refunds (order_id, user_id, reason, type, amount) VALUES (?,?,?,?,?)`,
		rf.OrderID, rf.UserID, rf.Reason, defaultStr(rf.Type, "refund_only"), rf.Amount)
	if err != nil {
		return err
	}
	rf.ID, _ = res.LastInsertId()
	return nil
}

func (r *RefundRepo) ListByUser(userID int64) ([]model.Refund, error) {
	rows, err := r.db.Query(
		`SELECT id, order_id, user_id, reason, type, amount, status, admin_note, created_at
		 FROM refunds WHERE user_id=? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Refund{}
	for rows.Next() {
		var rf model.Refund
		if err := rows.Scan(&rf.ID, &rf.OrderID, &rf.UserID, &rf.Reason, &rf.Type, &rf.Amount, &rf.Status, &rf.AdminNote, &rf.CreatedAt); err == nil {
			out = append(out, rf)
		}
	}
	return out, nil
}

func (r *RefundRepo) Get(id, userID int64) (*model.Refund, error) {
	rf := &model.Refund{}
	q := `SELECT id, order_id, user_id, reason, type, amount, status, admin_note, created_at FROM refunds WHERE id=?`
	args := []any{id}
	if userID > 0 {
		q += " AND user_id=?"
		args = append(args, userID)
	}
	err := r.db.QueryRow(q, args...).Scan(&rf.ID, &rf.OrderID, &rf.UserID, &rf.Reason, &rf.Type, &rf.Amount, &rf.Status, &rf.AdminNote, &rf.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rf, err
}

// UpdateStatus moves a refund through its lifecycle (admin action).
func (r *RefundRepo) UpdateStatus(id int64, status, note string) error {
	_, err := r.db.Exec(`UPDATE refunds SET status=?, admin_note=? WHERE id=?`, status, note, id)
	return err
}

// ===================== Coupon =====================

type CouponRepo struct{ db *sql.DB }

func NewCouponRepo(db *sql.DB) *CouponRepo { return &CouponRepo{db: db} }

// ListAvailable returns all active coupons, annotated with whether the given
// user has already claimed each.
func (r *CouponRepo) ListAvailable(userID int64) ([]model.Coupon, error) {
	rows, err := r.db.Query(
		`SELECT id, title, coupon_type, threshold, value, total_count, claimed_count, start_date, end_date, status FROM coupons WHERE status='active' ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Coupon{}
	for rows.Next() {
		var c model.Coupon
		if err := rows.Scan(&c.ID, &c.Title, &c.CouponType, &c.Threshold, &c.Value, &c.TotalCount, &c.ClaimedCount, &c.StartDate, &c.EndDate, &c.Status); err == nil {
			if userID > 0 {
				var claimed int
				_ = r.db.QueryRow(`SELECT 1 FROM user_coupons WHERE user_id=? AND coupon_id=?`, userID, c.ID).Scan(&claimed)
				c.IsClaimed = claimed == 1
			}
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *CouponRepo) Get(id int64) (*model.Coupon, error) {
	c := &model.Coupon{}
	err := r.db.QueryRow(
		`SELECT id, title, coupon_type, threshold, value, total_count, claimed_count, start_date, end_date, status FROM coupons WHERE id=?`, id,
	).Scan(&c.ID, &c.Title, &c.CouponType, &c.Threshold, &c.Value, &c.TotalCount, &c.ClaimedCount, &c.StartDate, &c.EndDate, &c.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// Claim lets a user claim a coupon (idempotent; fails if already claimed or sold out).
func (r *CouponRepo) Claim(userID, couponID int64) error {
	c, err := r.Get(couponID)
	if err != nil || c == nil {
		return fmt.Errorf("优惠券不存在")
	}
	if c.TotalCount > 0 && c.ClaimedCount >= c.TotalCount {
		return fmt.Errorf("优惠券已领完")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO user_coupons (user_id, coupon_id) VALUES (?,?)`, userID, couponID)
	if err != nil {
		return fmt.Errorf("您已领取过该优惠券")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("领取失败")
	}
	if _, err := tx.Exec(`UPDATE coupons SET claimed_count = claimed_count + 1 WHERE id=?`, couponID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListUserCoupons returns the coupons a user has claimed (unused first).
func (r *CouponRepo) ListUserCoupons(userID int64) ([]model.UserCoupon, error) {
	rows, err := r.db.Query(
		`SELECT uc.id, uc.user_id, uc.coupon_id, uc.is_used, uc.created_at,
		        c.title, c.coupon_type, c.threshold, c.value, c.end_date
		 FROM user_coupons uc JOIN coupons c ON c.id = uc.coupon_id
		 WHERE uc.user_id=? ORDER BY uc.is_used ASC, uc.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.UserCoupon{}
	for rows.Next() {
		var uc model.UserCoupon
		var c model.Coupon
		if err := rows.Scan(&uc.ID, &uc.UserID, &uc.CouponID, &uc.IsUsed, &uc.CreatedAt,
			&c.Title, &c.CouponType, &c.Threshold, &c.Value, &c.EndDate); err == nil {
			c.ID = uc.CouponID
			uc.Coupon = &c
			out = append(out, uc)
		}
	}
	return out, nil
}

// MarkUsed marks a user-coupon as used.
func (r *CouponRepo) MarkUsed(userCouponID, userID int64) error {
	_, err := r.db.Exec(`UPDATE user_coupons SET is_used=1 WHERE id=? AND user_id=?`, userCouponID, userID)
	return err
}

// Create inserts a new coupon template (admin).
func (r *CouponRepo) Create(c *model.Coupon) error {
	res, err := r.db.Exec(
		`INSERT INTO coupons (title, coupon_type, threshold, value, total_count, start_date, end_date, status) VALUES (?,?,?,?,?,?,?,?)`,
		c.Title, defaultStr(c.CouponType, "deduct"), c.Threshold, c.Value, c.TotalCount, c.StartDate, c.EndDate, defaultStr(c.Status, "active"))
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

// ===================== FTS5 Search =====================

// SearchFTS uses SQLite FTS5 to do full-text search over products. It falls
// back to LIKE if FTS5 is unavailable.
func (r *RefundRepo) SearchProducts(q string, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 20
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	// Build FTS5 MATCH query: prefix-match each token.
	tokens := strings.Fields(q)
	matchParts := []string{}
	for _, t := range tokens {
		matchParts = append(matchParts, t+"*")
	}
	matchExpr := strings.Join(matchParts, " ")
	rows, err := r.db.Query(
		`SELECT rowid FROM products_fts WHERE products_fts MATCH ? ORDER BY rank LIMIT ?`, matchExpr, limit)
	if err != nil {
		// FTS5 not available — fall back to LIKE.
		return r.searchLike(q, limit)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *RefundRepo) searchLike(q string, limit int) ([]int64, error) {
	rows, err := r.db.Query(
		`SELECT id FROM products WHERE name LIKE ? ORDER BY sales DESC LIMIT ?`, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// Suggest returns product name prefixes for search auto-complete.
func (r *RefundRepo) Suggest(prefix string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(
		`SELECT DISTINCT name FROM products WHERE name LIKE ? LIMIT ?`, prefix+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			out = append(out, name)
		}
	}
	return out, nil
}
