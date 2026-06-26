package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
)

// ===================== User =====================

type UserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(u *model.User) error {
	hashed := u.Password // seeded/mock users store plaintext; real register hashes
	res, err := r.db.Exec(
		`INSERT INTO users (username, password, nickname, avatar, phone) VALUES (?,?,?,?,?)`,
		u.Username, hashed, defaultStr(u.Nickname, u.Username), u.Avatar, u.Phone)
	if err != nil {
		return err
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password, nickname, avatar, phone, created_at FROM users WHERE username=?`, username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Avatar, &u.Phone, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *UserRepo) Get(id int64) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password, nickname, avatar, phone, created_at FROM users WHERE id=?`, id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Avatar, &u.Phone, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *UserRepo) Exists(username string) bool {
	var n int
	_ = r.db.QueryRow(`SELECT 1 FROM users WHERE username=? LIMIT 1`, username).Scan(&n)
	return n == 1
}

// UpdateProfile edits a user's mutable profile fields (nickname/avatar/phone).
func (r *UserRepo) UpdateProfile(u *model.User) error {
	res, err := r.db.Exec(
		`UPDATE users SET nickname=?, avatar=?, phone=? WHERE id=?`,
		u.Nickname, u.Avatar, u.Phone, u.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ===================== Category =====================

type CategoryRepo struct{ db *sql.DB }

func NewCategoryRepo(db *sql.DB) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) All() ([]model.Category, error) {
	rows, err := r.db.Query(`SELECT id, name, icon, sort_order FROM categories ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Category{}
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder); err == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *CategoryRepo) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&n)
	return n, err
}

// ===================== Product =====================

type ProductRepo struct{ db *sql.DB }

func NewProductRepo(db *sql.DB) *ProductRepo { return &ProductRepo{db: db} }

func (r *ProductRepo) List(page, pageSize int, categoryID int64, keyword string) ([]model.Product, int, error) {
	where := ""
	args := []any{}
	if categoryID > 0 {
		where = "WHERE category_id=?"
		args = append(args, categoryID)
	}
	if keyword != "" {
		if where == "" {
			where = "WHERE name LIKE ?"
		} else {
			where += " AND name LIKE ?"
		}
		args = append(args, "%"+keyword+"%")
	}
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := r.db.QueryRow("SELECT COUNT(*) FROM products "+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := "SELECT id,name,subtitle,price,original_price,image,images,category,category_id,shop,stock,sales,description,tags,is_seckill,created_at FROM products " +
		where + " ORDER BY sales DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := scanProduct(rows, &p); err == nil {
			out = append(out, p)
		}
	}
	return out, total, nil
}

func (r *ProductRepo) Seckill(limit int) ([]model.Product, error) {
	if limit <= 0 {
		limit = 6
	}
	rows, err := r.db.Query(
		`SELECT id,name,subtitle,price,original_price,image,images,category,category_id,shop,stock,sales,description,tags,is_seckill,created_at
		 FROM products WHERE is_seckill=1 ORDER BY sales DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := scanProduct(rows, &p); err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *ProductRepo) Get(id int64) (*model.Product, error) {
	p := &model.Product{}
	row := r.db.QueryRow(
		`SELECT id,name,subtitle,price,original_price,image,images,category,category_id,shop,stock,sales,description,tags,is_seckill,created_at
		 FROM products WHERE id=?`, id)
	if err := scanProductRow(row, p); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *ProductRepo) Create(p *model.ProductInput) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO products (name,subtitle,price,original_price,image,images,category,category_id,shop,stock,sales,description,tags,is_seckill)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.Name, p.Subtitle, p.Price, p.OriginalPrice, p.Image, p.Images, p.Category, p.CategoryID,
		defaultStr(p.Shop, "京东自营"), defaultInt(p.Stock, 999), p.Sales, p.Description, p.Tags, p.IsSeckill)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ProductRepo) Update(id int64, p *model.ProductInput) error {
	_, err := r.db.Exec(
		`UPDATE products SET name=?,subtitle=?,price=?,original_price=?,image=?,images=?,category=?,category_id=?,shop=?,stock=?,sales=?,description=?,tags=?,is_seckill=? WHERE id=?`,
		p.Name, p.Subtitle, p.Price, p.OriginalPrice, p.Image, p.Images, p.Category, p.CategoryID,
		defaultStr(p.Shop, "京东自营"), p.Stock, p.Sales, p.Description, p.Tags, p.IsSeckill, id)
	return err
}

func (r *ProductRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM products WHERE id=?`, id)
	return err
}

func scanProduct(rows *sql.Rows, p *model.Product) error {
	return rows.Scan(&p.ID, &p.Name, &p.Subtitle, &p.Price, &p.OriginalPrice, &p.Image, &p.Images,
		&p.Category, &p.CategoryID, &p.Shop, &p.Stock, &p.Sales, &p.Description, &p.Tags, &p.IsSeckill, &p.CreatedAt)
}

func scanProductRow(row *sql.Row, p *model.Product) error {
	return row.Scan(&p.ID, &p.Name, &p.Subtitle, &p.Price, &p.OriginalPrice, &p.Image, &p.Images,
		&p.Category, &p.CategoryID, &p.Shop, &p.Stock, &p.Sales, &p.Description, &p.Tags, &p.IsSeckill, &p.CreatedAt)
}

// ===================== Cart =====================

type CartRepo struct{ db *sql.DB }

func NewCartRepo(db *sql.DB) *CartRepo { return &CartRepo{db: db} }

func (r *CartRepo) List(userID int64) ([]model.CartItem, error) {
	rows, err := r.db.Query(
		`SELECT c.id, c.user_id, c.product_id, c.quantity, c.selected, c.created_at,
		        p.name, p.image, p.price, p.stock
		 FROM cart_items c JOIN products p ON p.id = c.product_id
		 WHERE c.user_id=? ORDER BY c.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.CartItem{}
	for rows.Next() {
		var c model.CartItem
		if err := rows.Scan(&c.ID, &c.UserID, &c.ProductID, &c.Quantity, &c.Selected, &c.CreatedAt,
			&c.ProductName, &c.ProductImg, &c.Price, &c.Stock); err == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *CartRepo) Add(userID, productID int64, qty int) error {
	if qty < 1 {
		qty = 1
	}
	// Upsert: if the product is already in the cart, increment quantity.
	var existingID int64
	err := r.db.QueryRow(`SELECT id FROM cart_items WHERE user_id=? AND product_id=?`, userID, productID).Scan(&existingID)
	if err == nil {
		_, err = r.db.Exec(`UPDATE cart_items SET quantity = quantity + ? WHERE id=?`, qty, existingID)
		return err
	}
	if err != sql.ErrNoRows {
		return err
	}
	_, err = r.db.Exec(`INSERT INTO cart_items (user_id, product_id, quantity) VALUES (?,?,?)`, userID, productID, qty)
	return err
}

func (r *CartRepo) Update(id, userID int64, qty, selected int) error {
	if qty < 0 {
		qty = 0
	}
	if qty == 0 {
		_, err := r.db.Exec(`DELETE FROM cart_items WHERE id=? AND user_id=?`, id, userID)
		return err
	}
	_, err := r.db.Exec(`UPDATE cart_items SET quantity=?, selected=? WHERE id=? AND user_id=?`, qty, selected, id, userID)
	return err
}

func (r *CartRepo) Delete(id, userID int64) error {
	_, err := r.db.Exec(`DELETE FROM cart_items WHERE id=? AND user_id=?`, id, userID)
	return err
}

func (r *CartRepo) Count(userID int64) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COALESCE(SUM(quantity),0) FROM cart_items WHERE user_id=?`, userID).Scan(&n)
	return n, err
}

func (r *CartRepo) Clear(userID int64) error {
	_, err := r.db.Exec(`DELETE FROM cart_items WHERE user_id=?`, userID)
	return err
}

// ===================== Order =====================

type OrderRepo struct{ db *sql.DB }

func NewOrderRepo(db *sql.DB) *OrderRepo { return &OrderRepo{db: db} }

func (r *OrderRepo) Create(o *model.Order) error {
	o.OrderNo = fmt.Sprintf("JD%d%d", time.Now().Unix(), o.UserID)
	res, err := r.db.Exec(
		`INSERT INTO orders (user_id, order_no, total, status, items_json, address) VALUES (?,?,?,?,?,?)`,
		o.UserID, o.OrderNo, o.Total, defaultStr(o.Status, "pending"), o.ItemsJSON, o.Address)
	if err != nil {
		return err
	}
	o.ID, _ = res.LastInsertId()
	return nil
}

// Get loads a single order. When userID > 0, it also enforces ownership.
func (r *OrderRepo) Get(id, userID int64) (*model.Order, error) {
	o := &model.Order{}
	q := `SELECT id, user_id, order_no, total, status, items_json, address, created_at FROM orders WHERE id=?`
	row := r.db.QueryRow(q, id)
	if err := row.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Total, &o.Status, &o.ItemsJSON, &o.Address, &o.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if userID > 0 && o.UserID != userID {
		return nil, nil
	}
	return o, nil
}

func (r *OrderRepo) ListByUser(userID int64) ([]model.Order, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, order_no, total, status, items_json, address, created_at
		 FROM orders WHERE user_id=? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Order{}
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Total, &o.Status, &o.ItemsJSON, &o.Address, &o.CreatedAt); err == nil {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *OrderRepo) UpdateStatus(id, userID int64, status string) error {
	_, err := r.db.Exec(`UPDATE orders SET status=? WHERE id=? AND user_id=?`, status, id, userID)
	return err
}

// ConfirmReceipt completes a delivered order (确认收货). Only an order in
// 'shipped' or 'in_transit' state may be confirmed.
func (r *OrderRepo) ConfirmReceipt(id, userID int64) error {
	res, err := r.db.Exec(
		`UPDATE orders SET status='completed' WHERE id=? AND user_id=? AND status IN ('shipped','in_transit')`,
		id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("当前订单状态不允许确认收货")
	}
	return nil
}

// ===================== Review =====================

type ReviewRepo struct{ db *sql.DB }

func NewReviewRepo(db *sql.DB) *ReviewRepo { return &ReviewRepo{db: db} }

func (r *ReviewRepo) ListByProduct(productID int64) ([]model.Review, error) {
	rows, err := r.db.Query(
		`SELECT id, product_id, user_id, username, rating, content, created_at
		 FROM reviews WHERE product_id=? ORDER BY id DESC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Review{}
	for rows.Next() {
		var rv model.Review
		if err := rows.Scan(&rv.ID, &rv.ProductID, &rv.UserID, &rv.Username, &rv.Rating, &rv.Content, &rv.CreatedAt); err == nil {
			out = append(out, rv)
		}
	}
	return out, nil
}

func (r *ReviewRepo) Create(rv *model.Review) error {
	if rv.Rating < 1 || rv.Rating > 5 {
		rv.Rating = 5
	}
	res, err := r.db.Exec(
		`INSERT INTO reviews (product_id, user_id, username, rating, content) VALUES (?,?,?,?,?)`,
		rv.ProductID, rv.UserID, rv.Username, rv.Rating, rv.Content)
	if err != nil {
		return err
	}
	rv.ID, _ = res.LastInsertId()
	return nil
}

// ===================== Address =====================

type AddressRepo struct{ db *sql.DB }

func NewAddressRepo(db *sql.DB) *AddressRepo { return &AddressRepo{db: db} }

func (r *AddressRepo) List(userID int64) ([]model.Address, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, phone, detail, is_default FROM addresses WHERE user_id=? ORDER BY is_default DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Address{}
	for rows.Next() {
		var a model.Address
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Phone, &a.Detail, &a.IsDefault); err == nil {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *AddressRepo) Get(id, userID int64) (*model.Address, error) {
	a := &model.Address{}
	err := r.db.QueryRow(
		`SELECT id, user_id, name, phone, detail, is_default FROM addresses WHERE id=? AND user_id=?`, id, userID,
	).Scan(&a.ID, &a.UserID, &a.Name, &a.Phone, &a.Detail, &a.IsDefault)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

// Create adds an address. If is_default, all other addresses for the user are
// cleared so there is only ever one default.
func (r *AddressRepo) Create(a *model.Address) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if a.IsDefault == 1 {
		if _, err := tx.Exec(`UPDATE addresses SET is_default=0 WHERE user_id=?`, a.UserID); err != nil {
			return err
		}
	}
	res, err := tx.Exec(
		`INSERT INTO addresses (user_id, name, phone, detail, is_default) VALUES (?,?,?,?,?)`,
		a.UserID, a.Name, a.Phone, a.Detail, a.IsDefault)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	a.ID, _ = res.LastInsertId()
	return nil
}

func (r *AddressRepo) Update(a *model.Address) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if a.IsDefault == 1 {
		if _, err := tx.Exec(`UPDATE addresses SET is_default=0 WHERE user_id=?`, a.UserID); err != nil {
			return err
		}
	}
	res, err := tx.Exec(
		`UPDATE addresses SET name=?, phone=?, detail=?, is_default=? WHERE id=? AND user_id=?`,
		a.Name, a.Phone, a.Detail, a.IsDefault, a.ID, a.UserID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *AddressRepo) Delete(id, userID int64) error {
	_, err := r.db.Exec(`DELETE FROM addresses WHERE id=? AND user_id=?`, id, userID)
	return err
}

// ===================== Favorite (wishlist) =====================

type FavoriteRepo struct{ db *sql.DB }

func NewFavoriteRepo(db *sql.DB) *FavoriteRepo { return &FavoriteRepo{db: db} }

// Toggle adds or removes a favorite; returns true if now favorited.
func (r *FavoriteRepo) Toggle(userID, productID int64) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Try to delete first.
	res, err := tx.Exec(`DELETE FROM favorites WHERE user_id=? AND product_id=?`, userID, productID)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		// was favorited, now removed.
		return false, tx.Commit()
	}
	if _, err := tx.Exec(`INSERT INTO favorites (user_id, product_id) VALUES (?,?)`, userID, productID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// IsFavorited reports whether a user has favorited a product.
func (r *FavoriteRepo) IsFavorited(userID, productID int64) bool {
	var one int
	_ = r.db.QueryRow(`SELECT 1 FROM favorites WHERE user_id=? AND product_id=?`, userID, productID).Scan(&one)
	return one == 1
}

// ListByUser returns a user's favorited products (newest first).
func (r *FavoriteRepo) ListByUser(userID int64) ([]model.Favorite, error) {
	rows, err := r.db.Query(
		`SELECT f.id, f.product_id, f.created_at, p.name, p.image, p.price
		 FROM favorites f JOIN products p ON p.id = f.product_id
		 WHERE f.user_id=? ORDER BY f.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Favorite{}
	for rows.Next() {
		var f model.Favorite
		if err := rows.Scan(&f.ID, &f.ProductID, &f.CreatedAt, &f.ProductName, &f.ProductImg, &f.Price); err == nil {
			f.UserID = userID
			out = append(out, f)
		}
	}
	return out, nil
}

// CountByUser reports how many products a user has favorited.
func (r *FavoriteRepo) CountByUser(userID int64) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM favorites WHERE user_id=?`, userID).Scan(&n)
	return n, err
}

// ===================== Browse history =====================

type HistoryRepo struct{ db *sql.DB }

func NewHistoryRepo(db *sql.DB) *HistoryRepo { return &HistoryRepo{db: db} }

// RecordView bumps a product to the top of the user's history (insert-or-
// update the viewed_at via upsert). Anonymous users (uid <= 0) are skipped.
func (r *HistoryRepo) RecordView(userID, productID int64) error {
	if userID <= 0 {
		return nil
	}
	_, err := r.db.Exec(
		`INSERT INTO browse_history (user_id, product_id, viewed_at) VALUES (?,?,CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id, product_id) DO UPDATE SET viewed_at=CURRENT_TIMESTAMP`,
		userID, productID)
	return err
}

// ListByUser returns the user's recently viewed products (newest first).
func (r *HistoryRepo) ListByUser(userID int64, limit int) ([]model.History, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := r.db.Query(
		`SELECT h.id, h.product_id, h.viewed_at, p.name, p.image, p.price
		 FROM browse_history h JOIN products p ON p.id = h.product_id
		 WHERE h.user_id=? ORDER BY h.viewed_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.History{}
	for rows.Next() {
		var h model.History
		if err := rows.Scan(&h.ID, &h.ProductID, &h.ViewedAt, &h.ProductName, &h.ProductImg, &h.Price); err == nil {
			h.UserID = userID
			out = append(out, h)
		}
	}
	return out, nil
}

func (r *HistoryRepo) Clear(userID int64) error {
	_, err := r.db.Exec(`DELETE FROM browse_history WHERE user_id=?`, userID)
	return err
}

// ===================== Daily check-in =====================

type CheckInRepo struct{ db *sql.DB }

func NewCheckInRepo(db *sql.DB) *CheckInRepo { return &CheckInRepo{db: db} }

// CheckIn records today's check-in for a user. It computes a streak (consecutive
// days) and awards points. Returns the resulting record and whether it was a new
// check-in (false = already checked in today).
func (r *CheckInRepo) CheckIn(userID int64) (*model.CheckIn, bool, error) {
	today := time.Now().Format("2006-01-02")
	// Already checked in today?
	var existing model.CheckIn
	err := r.db.QueryRow(
		`SELECT id, user_id, check_date, streak, points FROM check_ins WHERE user_id=? AND check_date=?`, userID, today,
	).Scan(&existing.ID, &existing.UserID, &existing.CheckDate, &existing.Streak, &existing.Points)
	if err == nil {
		return &existing, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}
	// Determine streak from yesterday's record.
	streak := 1
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	var prevStreak int
	_ = r.db.QueryRow(`SELECT streak FROM check_ins WHERE user_id=? AND check_date=?`, userID, yesterday).Scan(&prevStreak)
	if prevStreak > 0 {
		streak = prevStreak + 1
	}
	// Points scale gently with streak: day N awards N points (capped at 30).
	points := streak
	if points > 30 {
		points = 30
	}
	ci := &model.CheckIn{UserID: userID, CheckDate: today, Streak: streak, Points: points}
	res, err := r.db.Exec(
		`INSERT INTO check_ins (user_id, check_date, streak, points) VALUES (?,?,?,?)`,
		userID, today, streak, points)
	if err != nil {
		return nil, false, err
	}
	ci.ID, _ = res.LastInsertId()
	return ci, true, nil
}

// Status returns the latest check-in for a user (or nil if none) + total points.
func (r *CheckInRepo) Status(userID int64) (last *model.CheckIn, totalPoints int, err error) {
	last = &model.CheckIn{}
	err = r.db.QueryRow(
		`SELECT id, user_id, check_date, streak, points FROM check_ins WHERE user_id=? ORDER BY check_date DESC LIMIT 1`, userID,
	).Scan(&last.ID, &last.UserID, &last.CheckDate, &last.Streak, &last.Points)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	_ = r.db.QueryRow(`SELECT COALESCE(SUM(points), 0) FROM check_ins WHERE user_id=?`, userID).Scan(&totalPoints)
	return last, totalPoints, nil
}

// ===================== helpers =====================

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func defaultInt(n, d int) int {
	if n == 0 {
		return d
	}
	return n
}
