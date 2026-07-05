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
	q := "SELECT id,name,subtitle,price,original_price,image,images,category,category_id,shop,stock,sales,description,tags,is_seckill,video_url,vip_price,created_at FROM products " +
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
		`SELECT id,name,subtitle,price,original_price,image,images,category,category_id,shop,stock,sales,description,tags,is_seckill,video_url,vip_price,created_at
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
		`SELECT id,name,subtitle,price,original_price,image,images,category,category_id,shop,stock,sales,description,tags,is_seckill,video_url,vip_price,created_at
		 FROM products WHERE id=?`, id)
	if err := scanProductRow(row, p); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

// Related returns products in the same category (or sharing tags) as the given
// product, excluding itself. Used for "看了又看" recommendations.
func (r *ProductRepo) Related(id int64, limit int) ([]model.Product, error) {
	if limit <= 0 {
		limit = 6
	}
	// First get the source product to know its category/tags.
	src, err := r.Get(id)
	if err != nil || src == nil {
		return nil, nil
	}
	rows, err := r.db.Query(
		`SELECT id,name,subtitle,price,original_price,image,images,category,category_id,shop,stock,sales,description,tags,is_seckill,video_url,vip_price,created_at
		 FROM products WHERE id != ? AND category_id=? ORDER BY sales DESC LIMIT ?`, id, src.CategoryID, limit)
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
		&p.Category, &p.CategoryID, &p.Shop, &p.Stock, &p.Sales, &p.Description, &p.Tags, &p.IsSeckill, &p.VideoURL, &p.VipPrice, &p.CreatedAt)
}

func scanProductRow(row *sql.Row, p *model.Product) error {
	return row.Scan(&p.ID, &p.Name, &p.Subtitle, &p.Price, &p.OriginalPrice, &p.Image, &p.Images,
		&p.Category, &p.CategoryID, &p.Shop, &p.Stock, &p.Sales, &p.Description, &p.Tags, &p.IsSeckill, &p.VideoURL, &p.VipPrice, &p.CreatedAt)
}

// ===================== Cart =====================

type CartRepo struct{ db *sql.DB }

func NewCartRepo(db *sql.DB) *CartRepo { return &CartRepo{db: db} }

func (r *CartRepo) List(userID int64) ([]model.CartItem, error) {
	rows, err := r.db.Query(
		`SELECT c.id, c.user_id, c.product_id, c.quantity, c.selected, c.created_at,
		        p.name, p.image, p.price, p.original_price, p.stock
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
			&c.ProductName, &c.ProductImg, &c.Price, &c.OriginalPrice, &c.Stock); err == nil {
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
		`INSERT INTO orders (user_id, order_no, total, status, items_json, address, remark) VALUES (?,?,?,?,?,?,?)`,
		o.UserID, o.OrderNo, o.Total, defaultStr(o.Status, "pending"), o.ItemsJSON, o.Address, o.Remark)
	if err != nil {
		return err
	}
	o.ID, _ = res.LastInsertId()
	return nil
}

// Get loads a single order. When userID > 0, it also enforces ownership.
func (r *OrderRepo) Get(id, userID int64) (*model.Order, error) {
	o := &model.Order{}
	q := `SELECT id, user_id, order_no, total, status, items_json, address, remark, created_at FROM orders WHERE id=?`
	row := r.db.QueryRow(q, id)
	if err := row.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Total, &o.Status, &o.ItemsJSON, &o.Address, &o.Remark, &o.CreatedAt); err != nil {
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
		`SELECT id, user_id, order_no, total, status, items_json, address, remark, created_at
		 FROM orders WHERE user_id=? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Order{}
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Total, &o.Status, &o.ItemsJSON, &o.Address, &o.Remark, &o.CreatedAt); err == nil {
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

// CancelOrder cancels a pending order (only 'pending' orders can be cancelled).
func (r *OrderRepo) CancelOrder(id, userID int64) error {
	res, err := r.db.Exec(
		`UPDATE orders SET status='cancelled' WHERE id=? AND user_id=? AND status='pending'`,
		id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("当前订单状态不允许取消")
	}
	return nil
}

// ===================== Review =====================

type ReviewRepo struct{ db *sql.DB }

func NewReviewRepo(db *sql.DB) *ReviewRepo { return &ReviewRepo{db: db} }

func (r *ReviewRepo) ListByProduct(productID int64) ([]model.Review, error) {
	rows, err := r.db.Query(
		`SELECT id, product_id, user_id, username, rating, content, images, useful, created_at
		 FROM reviews WHERE product_id=? ORDER BY id DESC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Review{}
	for rows.Next() {
		var rv model.Review
		if err := rows.Scan(&rv.ID, &rv.ProductID, &rv.UserID, &rv.Username, &rv.Rating, &rv.Content, &rv.Images, &rv.Useful, &rv.CreatedAt); err == nil {
			rv.Reply, _ = r.getReply(rv.ID)
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
		`INSERT INTO reviews (product_id, user_id, username, rating, content, images) VALUES (?,?,?,?,?,?)`,
		rv.ProductID, rv.UserID, rv.Username, rv.Rating, rv.Content, rv.Images)
	if err != nil {
		return err
	}
	rv.ID, _ = res.LastInsertId()
	return nil
}

// getReply loads the (single) reply for a review, if any.
func (r *ReviewRepo) getReply(reviewID int64) (*model.ReviewReply, error) {
	rep := &model.ReviewReply{}
	err := r.db.QueryRow(
		`SELECT id, review_id, user_id, username, content, created_at FROM review_replies WHERE review_id=? LIMIT 1`, reviewID,
	).Scan(&rep.ID, &rep.ReviewID, &rep.UserID, &rep.Username, &rep.Content, &rep.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rep, err
}

// AddReply posts a reply to a review (merchant/owner response).
func (r *ReviewRepo) AddReply(rep *model.ReviewReply) error {
	res, err := r.db.Exec(
		`INSERT INTO review_replies (review_id, user_id, username, content) VALUES (?,?,?,?)`,
		rep.ReviewID, rep.UserID, rep.Username, rep.Content)
	if err != nil {
		return err
	}
	rep.ID, _ = res.LastInsertId()
	return nil
}

// MarkUseful increments the helpful-vote counter for a review (评价有用).
func (r *ReviewRepo) MarkUseful(id int64) error {
	_, err := r.db.Exec(`UPDATE reviews SET useful = useful + 1 WHERE id=?`, id)
	return err
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
	// Subtract already-redeemed points so totalPoints reflects the *available* balance.
	var spent int
	_ = r.db.QueryRow(`SELECT COALESCE(SUM(points_cost), 0) FROM redemptions WHERE user_id=?`, userID).Scan(&spent)
	totalPoints -= spent
	if totalPoints < 0 {
		totalPoints = 0
	}
	return last, totalPoints, nil
}

// ===================== Points mall (积分商城) =====================

type PointShopRepo struct{ db *sql.DB }

func NewPointShopRepo(db *sql.DB) *PointShopRepo { return &PointShopRepo{db: db} }

// List returns all redeemable point products (in stock, by sort order).
func (r *PointShopRepo) List() ([]model.PointProduct, error) {
	rows, err := r.db.Query(
		`SELECT id, name, image, points, stock, sort_order FROM point_products WHERE stock > 0 ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PointProduct{}
	for rows.Next() {
		var p model.PointProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Image, &p.Points, &p.Stock, &p.SortOrder); err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

// Get loads a single point product.
func (r *PointShopRepo) Get(id int64) (*model.PointProduct, error) {
	p := &model.PointProduct{}
	err := r.db.QueryRow(
		`SELECT id, name, image, points, stock, sort_order FROM point_products WHERE id=?`, id,
	).Scan(&p.ID, &p.Name, &p.Image, &p.Points, &p.Stock, &p.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// AvailablePoints reports a user's redeemable balance (earned minus spent).
func (r *PointShopRepo) AvailablePoints(userID int64) int {
	var earned, spent int
	_ = r.db.QueryRow(`SELECT COALESCE(SUM(points), 0) FROM check_ins WHERE user_id=?`, userID).Scan(&earned)
	_ = r.db.QueryRow(`SELECT COALESCE(SUM(points_cost), 0) FROM redemptions WHERE user_id=?`, userID).Scan(&spent)
	bal := earned - spent
	if bal < 0 {
		bal = 0
	}
	return bal
}

// Redeem exchanges points for a product. It is transactional and enforces both
// the point balance and the stock count.
func (r *PointShopRepo) Redeem(userID, productID int64) (*model.Redemption, error) {
	p, err := r.Get(productID)
	if err != nil || p == nil {
		return nil, fmt.Errorf("商品不存在")
	}
	if p.Stock <= 0 {
		return nil, fmt.Errorf("库存不足")
	}
	if r.AvailablePoints(userID) < p.Points {
		return nil, fmt.Errorf("积分不足")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(
		`INSERT INTO redemptions (user_id, product_id, points_cost) VALUES (?,?,?)`,
		userID, productID, p.Points)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE point_products SET stock = stock - 1 WHERE id=?`, productID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	rd := &model.Redemption{UserID: userID, ProductID: productID, PointsCost: p.Points, Status: "done", ProductName: p.Name}
	rd.ID, _ = res.LastInsertId()
	return rd, nil
}

// ListRedemptions returns a user's redemption history.
func (r *PointShopRepo) ListRedemptions(userID int64) ([]model.Redemption, error) {
	rows, err := r.db.Query(
		`SELECT r.id, r.user_id, r.product_id, r.points_cost, r.status, p.name, r.created_at
		 FROM redemptions r JOIN point_products p ON p.id = r.product_id
		 WHERE r.user_id=? ORDER BY r.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Redemption{}
	for rows.Next() {
		var rd model.Redemption
		if err := rows.Scan(&rd.ID, &rd.UserID, &rd.ProductID, &rd.PointsCost, &rd.Status, &rd.ProductName, &rd.CreatedAt); err == nil {
			out = append(out, rd)
		}
	}
	return out, nil
}

// SeedPointShop populates the points mall with demo rewards if empty.
func (r *PointShopRepo) SeedPointShop() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM point_products`).Scan(&n)
	if n > 0 {
		return
	}
	items := []struct {
		name   string
		image  string
		points int
		stock  int
	}{
		{"京东10元无门槛券", "https://api.dicebear.com/7.x/shapes/svg?seed=jd1", 100, 50},
		{"京东50元满减券", "https://api.dicebear.com/7.x/shapes/svg?seed=jd2", 500, 20},
		{"PLUS会员月卡", "https://api.dicebear.com/7.x/shapes/svg?seed=plus", 800, 10},
		{"蓝牙耳机", "https://api.dicebear.com/7.x/shapes/svg?seed=ear", 2000, 5},
		{"100M流量包", "https://api.dicebear.com/7.x/shapes/svg?seed=flow", 300, 100},
	}
	for i, it := range items {
		_, _ = r.db.Exec(
			`INSERT INTO point_products (name, image, points, stock, sort_order) VALUES (?,?,?,?,?)`,
			it.name, it.image, it.points, it.stock, i)
	}
}

// ===================== Seckill deals (秒杀活动) =====================

type SeckillRepo struct{ db *sql.DB }

func NewSeckillRepo(db *sql.DB) *SeckillRepo { return &SeckillRepo{db: db} }

// ListActive returns currently-active flash-sale deals, joined with product info.
func (r *SeckillRepo) ListActive(limit int) ([]model.SeckillDeal, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(
		`SELECT s.id, s.product_id, s.seckill_price, s.stock, s.sold, s.start_time, s.end_time, s.status,
		        p.name, p.image, p.original_price
		 FROM seckill_deals s JOIN products p ON p.id = s.product_id
		 WHERE s.status='active' AND s.end_time > CURRENT_TIMESTAMP AND s.stock > s.sold
		 ORDER BY s.end_time ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SeckillDeal{}
	for rows.Next() {
		var d model.SeckillDeal
		if err := rows.Scan(&d.ID, &d.ProductID, &d.SeckillPrice, &d.Stock, &d.Sold, &d.StartTime, &d.EndTime, &d.Status,
			&d.ProductName, &d.ProductImage, &d.OriginalPrice); err == nil {
			out = append(out, d)
		}
	}
	return out, nil
}

// Get returns a single seckill deal by id.
func (r *SeckillRepo) Get(id int64) (*model.SeckillDeal, error) {
	d := &model.SeckillDeal{}
	err := r.db.QueryRow(
		`SELECT s.id, s.product_id, s.seckill_price, s.stock, s.sold, s.start_time, s.end_time, s.status,
		        p.name, p.image, p.original_price
		 FROM seckill_deals s JOIN products p ON p.id = s.product_id WHERE s.id=?`, id,
	).Scan(&d.ID, &d.ProductID, &d.SeckillPrice, &d.Stock, &d.Sold, &d.StartTime, &d.EndTime, &d.Status,
		&d.ProductName, &d.ProductImage, &d.OriginalPrice)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// Grab is the atomic purchase action for a flash sale: it enforces the stock
// pool and bumps the sold counter transactionally. Returns the updated deal.
func (r *SeckillRepo) Grab(id, userID int64) (*model.SeckillDeal, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Lock the row and check availability.
	var stock, sold int
	var endTime string
	err = tx.QueryRow(`SELECT stock, sold, end_time FROM seckill_deals WHERE id=? AND status='active'`, id).
		Scan(&stock, &sold, &endTime)
	if err != nil {
		return nil, fmt.Errorf("活动不存在")
	}
	if stock <= sold {
		return nil, fmt.Errorf("已抢光")
	}
	if _, err := tx.Exec(`UPDATE seckill_deals SET sold = sold + 1 WHERE id=?`, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(id)
}

// SeedSeckill populates time-boxed demo flash sales if none exist yet.
func (r *SeckillRepo) SeedSeckill() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM seckill_deals`).Scan(&n)
	if n > 0 {
		return
	}
	// Pick the first few seeded products as flash-sale items.
	rows, err := r.db.Query(`SELECT id, price FROM products ORDER BY id LIMIT 6`)
	if err != nil {
		return
	}
	defer rows.Close()
	type pp struct {
		id    int64
		price float64
	}
	items := []pp{}
	for rows.Next() {
		var p pp
		if rows.Scan(&p.id, &p.price) == nil {
			items = append(items, p)
		}
	}
	now := time.Now()
	for i, p := range items {
		// Each deal runs for ~2 hours from now (staggered starts).
		start := now.Add(time.Duration(i) * time.Minute)
		end := start.Add(2 * time.Hour)
		_, _ = r.db.Exec(
			`INSERT INTO seckill_deals (product_id, seckill_price, stock, sold, start_time, end_time, status) VALUES (?,?,?,?,?,?,?)`,
			p.id, p.price*0.5, 50, 0, start, end, "active")
	}
}

// ===================== Group buys (拼团) =====================

type GroupBuyRepo struct{ db *sql.DB }

func NewGroupBuyRepo(db *sql.DB) *GroupBuyRepo { return &GroupBuyRepo{db: db} }

// ListActive returns open group-buy deals, joined with product info.
func (r *GroupBuyRepo) ListActive(limit int) ([]model.GroupBuy, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(
		`SELECT g.id, g.product_id, g.group_price, g.required, g.joined, g.status, g.end_time, g.created_at,
		        p.name, p.image, p.original_price
		 FROM group_buys g JOIN products p ON p.id = g.product_id
		 WHERE g.status='active' AND g.end_time > CURRENT_TIMESTAMP AND g.joined < g.required
		 ORDER BY g.end_time ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.GroupBuy{}
	for rows.Next() {
		var g model.GroupBuy
		if err := rows.Scan(&g.ID, &g.ProductID, &g.GroupPrice, &g.Required, &g.Joined, &g.Status, &g.EndTime, &g.CreatedAt,
			&g.ProductName, &g.ProductImage, &g.OriginalPrice); err == nil {
			out = append(out, g)
		}
	}
	return out, nil
}

// Get returns a single group-buy deal by id.
func (r *GroupBuyRepo) Get(id int64) (*model.GroupBuy, error) {
	g := &model.GroupBuy{}
	err := r.db.QueryRow(
		`SELECT g.id, g.product_id, g.group_price, g.required, g.joined, g.status, g.end_time, g.created_at,
		        p.name, p.image, p.original_price
		 FROM group_buys g JOIN products p ON p.id = g.product_id WHERE g.id=?`, id,
	).Scan(&g.ID, &g.ProductID, &g.GroupPrice, &g.Required, &g.Joined, &g.Status, &g.EndTime, &g.CreatedAt,
		&g.ProductName, &g.ProductImage, &g.OriginalPrice)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return g, err
}

// Join adds a user to a group buy. When the required count is reached, the deal
// flips to "success". Duplicate joins are rejected by the UNIQUE constraint.
func (r *GroupBuyRepo) Join(id, userID int64) (*model.GroupBuy, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var joined, required int
	err = tx.QueryRow(`SELECT joined, required FROM group_buys WHERE id=? AND status='active'`, id).Scan(&joined, &required)
	if err != nil {
		return nil, fmt.Errorf("拼团不存在或已结束")
	}
	if joined >= required {
		return nil, fmt.Errorf("拼团已满")
	}
	if _, err := tx.Exec(`INSERT INTO group_buy_members (group_buy_id, user_id) VALUES (?,?)`, id, userID); err != nil {
		return nil, fmt.Errorf("您已参与该拼团")
	}
	newJoined := joined + 1
	if newJoined >= required {
		if _, err := tx.Exec(`UPDATE group_buys SET joined=?, status='success' WHERE id=?`, newJoined, id); err != nil {
			return nil, err
		}
	} else {
		if _, err := tx.Exec(`UPDATE group_buys SET joined=? WHERE id=?`, newJoined, id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(id)
}

// SeedGroupBuys populates demo team-purchase deals if none exist yet.
func (r *GroupBuyRepo) SeedGroupBuys() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM group_buys`).Scan(&n)
	if n > 0 {
		return
	}
	rows, err := r.db.Query(`SELECT id, price FROM products ORDER BY id LIMIT 5`)
	if err != nil {
		return
	}
	defer rows.Close()
	type pp struct {
		id    int64
		price float64
	}
	items := []pp{}
	for rows.Next() {
		var p pp
		if rows.Scan(&p.id, &p.price) == nil {
			items = append(items, p)
		}
	}
	end := time.Now().Add(24 * time.Hour)
	for i, p := range items {
		required := 2 + (i % 3) // 2~4 buyers
		_, _ = r.db.Exec(
			`INSERT INTO group_buys (product_id, group_price, required, joined, status, end_time) VALUES (?,?,?,?,?,?)`,
			p.id, p.price*0.7, required, 1, "active", end)
	}
}

// ===================== Presales (预售定金) =====================

type PresaleRepo struct{ db *sql.DB }

func NewPresaleRepo(db *sql.DB) *PresaleRepo { return &PresaleRepo{db: db} }

// ListActive returns open presale deals, joined with product info.
func (r *PresaleRepo) ListActive(limit int) ([]model.Presale, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(
		`SELECT s.id, s.product_id, s.deposit, s.balance, s.final_price, s.stock, s.sold, s.deposit_end, s.balance_start, s.status,
		        p.name, p.image, p.original_price
		 FROM presales s JOIN products p ON p.id = s.product_id
		 WHERE s.status='active' AND s.deposit_end > CURRENT_TIMESTAMP AND s.stock > s.sold
		 ORDER BY s.deposit_end ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Presale{}
	for rows.Next() {
		var s model.Presale
		if err := rows.Scan(&s.ID, &s.ProductID, &s.Deposit, &s.Balance, &s.FinalPrice, &s.Stock, &s.Sold, &s.DepositEnd, &s.BalanceStart, &s.Status,
			&s.ProductName, &s.ProductImage, &s.OriginalPrice); err == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

// Get returns a single presale by id.
func (r *PresaleRepo) Get(id int64) (*model.Presale, error) {
	s := &model.Presale{}
	err := r.db.QueryRow(
		`SELECT s.id, s.product_id, s.deposit, s.balance, s.final_price, s.stock, s.sold, s.deposit_end, s.balance_start, s.status,
		        p.name, p.image, p.original_price
		 FROM presales s JOIN products p ON p.id = s.product_id WHERE s.id=?`, id,
	).Scan(&s.ID, &s.ProductID, &s.Deposit, &s.Balance, &s.FinalPrice, &s.Stock, &s.Sold, &s.DepositEnd, &s.BalanceStart, &s.Status,
		&s.ProductName, &s.ProductImage, &s.OriginalPrice)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

// PayDeposit records a user paying the deposit for a presale (transactional:
// enforces stock + dedups via UNIQUE).
func (r *PresaleRepo) PayDeposit(id, userID int64) (*model.PresaleOrder, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var stock, sold int
	var deposit float64
	err = tx.QueryRow(`SELECT stock, sold, deposit FROM presales WHERE id=? AND status='active'`, id).Scan(&stock, &sold, &deposit)
	if err != nil {
		return nil, fmt.Errorf("预售不存在或已结束")
	}
	if stock <= sold {
		return nil, fmt.Errorf("预售名额已满")
	}
	res, err := tx.Exec(`INSERT INTO presale_orders (presale_id, user_id, deposit_paid) VALUES (?,?,?)`, id, userID, deposit)
	if err != nil {
		return nil, fmt.Errorf("您已参与该预售")
	}
	if _, err := tx.Exec(`UPDATE presales SET sold = sold + 1 WHERE id=?`, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	oid, _ := res.LastInsertId()
	return &model.PresaleOrder{ID: oid, PresaleID: id, UserID: userID, DepositPaid: deposit, Status: "deposit"}, nil
}

// SeedPresales populates demo deposit deals if none exist yet.
func (r *PresaleRepo) SeedPresales() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM presales`).Scan(&n)
	if n > 0 {
		return
	}
	rows, err := r.db.Query(`SELECT id, price FROM products ORDER BY id LIMIT 5`)
	if err != nil {
		return
	}
	defer rows.Close()
	type pp struct {
		id    int64
		price float64
	}
	items := []pp{}
	for rows.Next() {
		var p pp
		if rows.Scan(&p.id, &p.price) == nil {
			items = append(items, p)
		}
	}
	depositEnd := time.Now().Add(48 * time.Hour)
	balanceStart := time.Now().Add(72 * time.Hour)
	for _, p := range items {
		final := p.price * 0.85 // 85折尾款价
		deposit := p.price * 0.1
		balance := final - deposit
		_, _ = r.db.Exec(
			`INSERT INTO presales (product_id, deposit, balance, final_price, stock, sold, deposit_end, balance_start, status) VALUES (?,?,?,?,?,?,?,?,?)`,
			p.id, deposit, balance, final, 30, 0, depositEnd, balanceStart, "active")
	}
}

// ===================== Price history (比价历史) =====================

type PriceHistoryRepo struct{ db *sql.DB }

func NewPriceHistoryRepo(db *sql.DB) *PriceHistoryRepo { return &PriceHistoryRepo{db: db} }

// ListByProduct returns a product's price snapshots, oldest-first.
func (r *PriceHistoryRepo) ListByProduct(productID int64, limit int) ([]model.PriceHistory, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := r.db.Query(
		`SELECT id, product_id, price, recorded_at FROM price_history WHERE product_id=? ORDER BY recorded_at ASC LIMIT ?`, productID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PriceHistory{}
	for rows.Next() {
		var h model.PriceHistory
		if err := rows.Scan(&h.ID, &h.ProductID, &h.Price, &h.RecordedAt); err == nil {
			out = append(out, h)
		}
	}
	return out, nil
}

// Record snapshots a product's current price (called on seed + product update).
func (r *PriceHistoryRepo) Record(productID int64, price float64) error {
	_, err := r.db.Exec(`INSERT INTO price_history (product_id, price) VALUES (?,?)`, productID, price)
	return err
}

// SeedPriceHistory back-fills a few synthetic historical points for each
// product so the chart has data on a fresh database.
func (r *PriceHistoryRepo) SeedPriceHistory() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM price_history`).Scan(&n)
	if n > 0 {
		return
	}
	rows, err := r.db.Query(`SELECT id, price FROM products`)
	if err != nil {
		return
	}
	defer rows.Close()
	type pp struct {
		id    int64
		price float64
	}
	items := []pp{}
	for rows.Next() {
		var p pp
		if rows.Scan(&p.id, &p.price) == nil {
			items = append(items, p)
		}
	}
	now := time.Now()
	for _, p := range items {
		// Generate ~5 points across the last 30 days with small fluctuations.
		for i := 4; i >= 0; i-- {
			day := now.AddDate(0, 0, -i*6)
			factor := 1.0 + float64(i)*0.05 // older points slightly pricier
			hp := p.price * factor
			_, _ = r.db.Exec(`INSERT INTO price_history (product_id, price, recorded_at) VALUES (?,?,?)`, p.id, hp, day)
		}
		// And the current price point.
		_, _ = r.db.Exec(`INSERT INTO price_history (product_id, price) VALUES (?,?)`, p.id, p.price)
	}
}

// ===================== Shop ratings (店铺评分) =====================

type ShopRatingRepo struct{ db *sql.DB }

func NewShopRatingRepo(db *sql.DB) *ShopRatingRepo { return &ShopRatingRepo{db: db} }

// Create records a buyer's multi-dimension shop rating.
func (r *ShopRatingRepo) Create(s *model.ShopRating) error {
	res, err := r.db.Exec(
		`INSERT INTO shop_ratings (shop, user_id, description_score, logistics_score, service_score, comment) VALUES (?,?,?,?,?,?)`,
		s.Shop, s.UserID, clampScore(s.DescriptionScore), clampScore(s.LogisticsScore), clampScore(s.ServiceScore), s.Comment)
	if err != nil {
		return err
	}
	s.ID, _ = res.LastInsertId()
	return nil
}

// Summary returns the aggregate average scores for a shop.
func (r *ShopRatingRepo) Summary(shop string) (*model.ShopRatingSummary, error) {
	s := &model.ShopRatingSummary{Shop: shop}
	err := r.db.QueryRow(
		`SELECT COUNT(*),
		        COALESCE(AVG(description_score),5), COALESCE(AVG(logistics_score),5), COALESCE(AVG(service_score),5)
		 FROM shop_ratings WHERE shop=?`, shop,
	).Scan(&s.Count, &s.DescriptionAvg, &s.LogisticsAvg, &s.ServiceAvg)
	if err != nil {
		// no rows → defaults
		s.DescriptionAvg, s.LogisticsAvg, s.ServiceAvg = 5, 5, 5
	}
	s.Overall = (s.DescriptionAvg + s.LogisticsAvg + s.ServiceAvg) / 3
	return s, nil
}

// ListByShop returns recent ratings for a shop.
func (r *ShopRatingRepo) ListByShop(shop string, limit int) ([]model.ShopRating, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := r.db.Query(
		`SELECT sr.id, sr.shop, sr.user_id, sr.description_score, sr.logistics_score, sr.service_score, sr.comment, sr.created_at,
		        u.nickname
		 FROM shop_ratings sr LEFT JOIN users u ON u.id = sr.user_id
		 WHERE sr.shop=? ORDER BY sr.id DESC LIMIT ?`, shop, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ShopRating{}
	for rows.Next() {
		var s model.ShopRating
		if err := rows.Scan(&s.ID, &s.Shop, &s.UserID, &s.DescriptionScore, &s.LogisticsScore, &s.ServiceScore, &s.Comment, &s.CreatedAt, &s.Username); err == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

func clampScore(n int) int {
	if n < 1 {
		return 1
	}
	if n > 5 {
		return 5
	}
	return n
}

// ===================== Bundles (组合套餐) =====================

type BundleRepo struct{ db *sql.DB }

func NewBundleRepo(db *sql.DB) *BundleRepo { return &BundleRepo{db: db} }

// List returns all bundles with their member products joined.
func (r *BundleRepo) List() ([]model.Bundle, error) {
	rows, err := r.db.Query(`SELECT id, title, bundle_price, original_total, product_ids FROM bundles ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Bundle{}
	for rows.Next() {
		var b model.Bundle
		if err := rows.Scan(&b.ID, &b.Title, &b.BundlePrice, &b.OriginalTotal, &b.ProductIDs); err == nil {
			b.Products = r.loadProducts(b.ProductIDs)
			out = append(out, b)
		}
	}
	return out, nil
}

func (r *BundleRepo) loadProducts(idsCSV string) []model.BundleItem {
	out := []model.BundleItem{}
	for _, idStr := range splitCSV(idsCSV) {
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id == 0 {
			continue
		}
		var item model.BundleItem
		if err := r.db.QueryRow(`SELECT id, name, image, price FROM products WHERE id=?`, id).
			Scan(&item.ID, &item.Name, &item.Image, &item.Price); err == nil {
			out = append(out, item)
		}
	}
	return out
}

func splitCSV(s string) []string {
	parts := []string{}
	cur := ""
	for _, c := range s {
		if c == ',' {
			if cur != "" {
				parts = append(parts, cur)
			}
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

// SeedBundles populates demo bundle deals if none exist.
func (r *BundleRepo) SeedBundles() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM bundles`).Scan(&n)
	if n > 0 {
		return
	}
	rows, err := r.db.Query(`SELECT id, price FROM products ORDER BY id LIMIT 10`)
	if err != nil {
		return
	}
	defer rows.Close()
	type pp struct {
		id    int64
		price float64
	}
	items := []pp{}
	for rows.Next() {
		var p pp
		if rows.Scan(&p.id, &p.price) == nil {
			items = append(items, p)
		}
	}
	// Group into bundles of 3.
	for i := 0; i+3 <= len(items); i += 3 {
		grp := items[i : i+3]
		total := 0.0
		ids := ""
		for j, p := range grp {
			total += p.price
			if j > 0 {
				ids += ","
			}
			ids += fmt.Sprintf("%d", p.id)
		}
		_, _ = r.db.Exec(
			`INSERT INTO bundles (title, bundle_price, original_total, product_ids, sort_order) VALUES (?,?,?,?,?)`,
			fmt.Sprintf("超值%d件套", len(grp)), total*0.85, total, ids, i/3)
	}
}

// ===================== Restock alerts (到货通知) =====================

type RestockRepo struct{ db *sql.DB }

func NewRestockRepo(db *sql.DB) *RestockRepo { return &RestockRepo{db: db} }

// Subscribe records a user's restock subscription (idempotent via UNIQUE).
func (r *RestockRepo) Subscribe(userID, productID int64) error {
	_, err := r.db.Exec(`INSERT OR IGNORE INTO restock_alerts (user_id, product_id) VALUES (?,?)`, userID, productID)
	return err
}

// Unsubscribe removes a subscription.
func (r *RestockRepo) Unsubscribe(userID, productID int64) error {
	_, err := r.db.Exec(`DELETE FROM restock_alerts WHERE user_id=? AND product_id=?`, userID, productID)
	return err
}

// IsSubscribed reports whether a user is subscribed.
func (r *RestockRepo) IsSubscribed(userID, productID int64) bool {
	var n int
	_ = r.db.QueryRow(`SELECT 1 FROM restock_alerts WHERE user_id=? AND product_id=?`, userID, productID).Scan(&n)
	return n == 1
}

// ===================== Product Q&A (商品问答) =====================

type QARepo struct{ db *sql.DB }

func NewQARepo(db *sql.DB) *QARepo { return &QARepo{db: db} }

// ListByProduct returns Q&A entries for a product, newest first.
func (r *QARepo) ListByProduct(productID int64) ([]model.ProductQA, error) {
	rows, err := r.db.Query(
		`SELECT id, product_id, user_id, username, question, answer, answerer, created_at
		 FROM product_qa WHERE product_id=? ORDER BY id DESC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ProductQA{}
	for rows.Next() {
		var qa model.ProductQA
		if err := rows.Scan(&qa.ID, &qa.ProductID, &qa.UserID, &qa.Username, &qa.Question, &qa.Answer, &qa.Answerer, &qa.CreatedAt); err == nil {
			out = append(out, qa)
		}
	}
	return out, nil
}

// Ask records a buyer's question.
func (r *QARepo) Ask(qa *model.ProductQA) error {
	res, err := r.db.Exec(
		`INSERT INTO product_qa (product_id, user_id, username, question) VALUES (?,?,?,?)`,
		qa.ProductID, qa.UserID, qa.Username, qa.Question)
	if err != nil {
		return err
	}
	qa.ID, _ = res.LastInsertId()
	return nil
}

// Answer records a seller's answer to a question.
func (r *QARepo) Answer(id int64, answer, answerer string) error {
	_, err := r.db.Exec(`UPDATE product_qa SET answer=?, answerer=? WHERE id=?`, answer, answerer, id)
	return err
}

// ===================== Order invoices (电子发票) =====================

type InvoiceRepo struct{ db *sql.DB }

func NewInvoiceRepo(db *sql.DB) *InvoiceRepo { return &InvoiceRepo{db: db} }

func (r *InvoiceRepo) Create(inv *model.OrderInvoice) error {
	res, err := r.db.Exec(
		`INSERT INTO order_invoices (order_id, user_id, invoice_type, title, tax_no, email) VALUES (?,?,?,?,?,?)`,
		inv.OrderID, inv.UserID, defaultStr(inv.InvoiceType, "personal"), inv.Title, inv.TaxNo, inv.Email)
	if err != nil {
		return err
	}
	inv.ID, _ = res.LastInsertId()
	return nil
}

func (r *InvoiceRepo) GetByOrder(orderID, userID int64) (*model.OrderInvoice, error) {
	inv := &model.OrderInvoice{}
	err := r.db.QueryRow(
		`SELECT id, order_id, user_id, invoice_type, title, tax_no, email, status FROM order_invoices WHERE order_id=? AND user_id=?`, orderID, userID,
	).Scan(&inv.ID, &inv.OrderID, &inv.UserID, &inv.InvoiceType, &inv.Title, &inv.TaxNo, &inv.Email, &inv.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return inv, err
}

// ===================== Tiered discounts (阶梯满减) =====================

type TieredDiscountRepo struct{ db *sql.DB }

func NewTieredDiscountRepo(db *sql.DB) *TieredDiscountRepo { return &TieredDiscountRepo{db: db} }

// ListActive returns active discount tiers sorted by threshold ascending.
func (r *TieredDiscountRepo) ListActive() ([]model.TieredDiscount, error) {
	rows, err := r.db.Query(
		`SELECT id, threshold, discount, status, sort_order
		 FROM tiered_discounts WHERE status='active'
		 ORDER BY threshold ASC, sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.TieredDiscount{}
	for rows.Next() {
		var t model.TieredDiscount
		if err := rows.Scan(&t.ID, &t.Threshold, &t.Discount, &t.Status, &t.SortOrder); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

// SeedTiers populates demo spend-X-get-Y-off tiers if none exist yet.
func (r *TieredDiscountRepo) SeedTiers() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM tiered_discounts`).Scan(&n)
	if n > 0 {
		return
	}
	tiers := []struct {
		threshold float64
		discount  float64
	}{
		{99, 5},
		{199, 15},
		{299, 30},
		{499, 60},
	}
	for i, t := range tiers {
		_, _ = r.db.Exec(
			`INSERT INTO tiered_discounts (threshold, discount, status, sort_order) VALUES (?,?,?,?)`,
			t.threshold, t.discount, "active", i)
	}
}

// SeedVIPPrices sets a VIP price (95% of regular) for the first few products if none set.
func SeedVIPPrices(db *sql.DB) {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM products WHERE vip_price > 0`).Scan(&n)
	if n > 0 {
		return
	}
	_, _ = db.Exec(`UPDATE products SET vip_price = ROUND(price * 0.95, 2) WHERE vip_price = 0`)
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
