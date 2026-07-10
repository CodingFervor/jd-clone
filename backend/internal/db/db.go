package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// Init opens the SQLite database (creating the file if needed) and creates all
// tables. It must be called once at startup, before any repository is used.
func Init(path string) error {
	// Ensure the data directory exists.
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create data dir: %w", err)
		}
	}

	var err error
	DB, err = sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(1) // SQLite serializes writes; avoid "database is locked".
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	if err = createTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func createTables() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			nickname TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			icon TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			subtitle TEXT NOT NULL DEFAULT '',
			price REAL NOT NULL,
			original_price REAL NOT NULL DEFAULT 0,
			image TEXT NOT NULL DEFAULT '',
			images TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			category_id INTEGER NOT NULL DEFAULT 0,
			shop TEXT NOT NULL DEFAULT '京东自营',
			stock INTEGER NOT NULL DEFAULT 999,
			sales INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			video_url TEXT NOT NULL DEFAULT '',
			vip_price REAL NOT NULL DEFAULT 0,
			is_seckill INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_products_seckill ON products(is_seckill)`,
		`CREATE TABLE IF NOT EXISTS cart_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL DEFAULT 1,
			selected INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cart_user ON cart_items(user_id)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			order_no TEXT NOT NULL UNIQUE,
			total REAL NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			items_json TEXT NOT NULL,
			address TEXT NOT NULL DEFAULT '',
			remark TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id)`,
		`CREATE TABLE IF NOT EXISTS reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			rating INTEGER NOT NULL DEFAULT 5,
			content TEXT NOT NULL DEFAULT '',
			images TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_product ON reviews(product_id)`,
		`CREATE TABLE IF NOT EXISTS addresses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			phone TEXT NOT NULL,
			detail TEXT NOT NULL,
			is_default INTEGER NOT NULL DEFAULT 0
		)`,
		// SKU: a specific spec combination (color/size/version) of a product.
		`CREATE TABLE IF NOT EXISTS skus (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			spec TEXT NOT NULL,             -- JSON, e.g. {"颜色":"黑色","版本":"256GB"}
			spec_text TEXT NOT NULL DEFAULT '', -- display string "黑色 256GB"
			price REAL NOT NULL,
			stock INTEGER NOT NULL DEFAULT 0,
			sku_code TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_skus_product ON skus(product_id)`,
		// Shipments: a shipped order's logistics envelope.
		`CREATE TABLE IF NOT EXISTS shipments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			tracking_no TEXT NOT NULL,
			carrier TEXT NOT NULL DEFAULT '京东快递',
			status TEXT NOT NULL DEFAULT 'shipped', -- pending, shipped, in_transit, delivered
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shipments_order ON shipments(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_shipments_tracking ON shipments(tracking_no)`,
		// Shipment tracks: chronological logistics events.
		`CREATE TABLE IF NOT EXISTS shipment_tracks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shipment_id INTEGER NOT NULL,
			description TEXT NOT NULL,
			location TEXT NOT NULL DEFAULT '',
			occurred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_shipment ON shipment_tracks(shipment_id)`,
		// Payments: a record of a payment attempt per order.
		`CREATE TABLE IF NOT EXISTS payments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			amount REAL NOT NULL,
			method TEXT NOT NULL DEFAULT 'wechat', -- wechat, alipay, unionpay
			transaction_no TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending', -- pending, success, failed
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_order ON payments(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_user ON payments(user_id)`,
		// Refunds: after-sale return/refund requests.
		`CREATE TABLE IF NOT EXISTS refunds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'refund', -- refund_only (仅退款), return_refund (退货退款)
			amount REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending', -- pending, approved, rejected, completed
			admin_note TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refunds_user ON refunds(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refunds_order ON refunds(order_id)`,
		// Coupons: discount coupons claimable by users.
		`CREATE TABLE IF NOT EXISTS coupons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			coupon_type TEXT NOT NULL DEFAULT 'deduct', -- deduct (满减), discount (折扣)
			threshold REAL NOT NULL DEFAULT 0,     -- 满 threshold 元可用
			value REAL NOT NULL DEFAULT 0,         -- deduct: 减金额; discount: 折扣(0.8=8折)
			total_count INTEGER NOT NULL DEFAULT 0,
			claimed_count INTEGER NOT NULL DEFAULT 0,
			start_date TEXT NOT NULL DEFAULT '',
			end_date TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// User-coupon associations (领取记录).
		`CREATE TABLE IF NOT EXISTS user_coupons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			coupon_id INTEGER NOT NULL,
			is_used INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, coupon_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_coupons_user ON user_coupons(user_id)`,
		// Favorites: products a user has favorited (wishlist).
		`CREATE TABLE IF NOT EXISTS favorites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, product_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id)`,
		// Browse history: products a logged-in user has viewed (recent first).
		`CREATE TABLE IF NOT EXISTS browse_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			viewed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, product_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_user ON browse_history(user_id)`,
		// Daily check-ins: one row per user per day (积分/奖励 demo).
		`CREATE TABLE IF NOT EXISTS check_ins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			check_date TEXT NOT NULL,
			streak INTEGER NOT NULL DEFAULT 1,
			points INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, check_date)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_checkins_user ON check_ins(user_id)`,
		// Point products: redeemable rewards in the points mall (积分商城).
		`CREATE TABLE IF NOT EXISTS point_products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			image TEXT NOT NULL DEFAULT '',
			points INTEGER NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		// Redemptions: a user's exchange record (扣减积分).
		`CREATE TABLE IF NOT EXISTS redemptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			points_cost INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'done',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_redemptions_user ON redemptions(user_id)`,
		// Review replies: a merchant/owner response to a buyer review.
		`CREATE TABLE IF NOT EXISTS review_replies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			review_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_replies_review ON review_replies(review_id)`,
		// Seckill deals: time-boxed flash sales with a separate stock pool.
		`CREATE TABLE IF NOT EXISTS seckill_deals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			seckill_price REAL NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			sold INTEGER NOT NULL DEFAULT 0,
			start_time DATETIME NOT NULL,
			end_time DATETIME NOT NULL,
			status TEXT NOT NULL DEFAULT 'active'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_seckill_active ON seckill_deals(status)`,
		// Group buys: team-purchase deals requiring N buyers to unlock the price (拼团).
		`CREATE TABLE IF NOT EXISTS group_buys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			group_price REAL NOT NULL DEFAULT 0,
			required INTEGER NOT NULL DEFAULT 2,
			joined INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			end_time DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_groupbuys_active ON group_buys(status)`,
		// Group buy members: who joined which group buy (拼团成员).
		`CREATE TABLE IF NOT EXISTS group_buy_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_buy_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(group_buy_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gbm_group ON group_buy_members(group_buy_id)`,
		// Presales: deposit-then-balance deals (预售定金).
		`CREATE TABLE IF NOT EXISTS presales (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			deposit REAL NOT NULL DEFAULT 0,
			balance REAL NOT NULL DEFAULT 0,
			final_price REAL NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			sold INTEGER NOT NULL DEFAULT 0,
			deposit_end DATETIME NOT NULL,
			balance_start DATETIME NOT NULL,
			status TEXT NOT NULL DEFAULT 'active'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_presales_active ON presales(status)`,
		// Presale orders: who paid a deposit for which presale (定金记录).
		`CREATE TABLE IF NOT EXISTS presale_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			presale_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			deposit_paid REAL NOT NULL DEFAULT 0,
			balance_paid REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'deposit', -- deposit, paid, cancelled
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(presale_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_presale_orders_user ON presale_orders(user_id)`,
		// Refund tracks: a chronological log of each refund status change (售后进度).
		`CREATE TABLE IF NOT EXISTS refund_tracks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			refund_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refund_tracks_refund ON refund_tracks(refund_id)`,
		// Price history: snapshots of a product's price over time (比价历史).
		`CREATE TABLE IF NOT EXISTS price_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			price REAL NOT NULL,
			recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_price_history_product ON price_history(product_id)`,
		// Shop ratings: aggregate buyer scores for a shop (店铺评分).
		`CREATE TABLE IF NOT EXISTS shop_ratings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shop TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			description_score INTEGER NOT NULL DEFAULT 5,
			logistics_score INTEGER NOT NULL DEFAULT 5,
			service_score INTEGER NOT NULL DEFAULT 5,
			comment TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shop_ratings_shop ON shop_ratings(shop)`,
		// Bundle deals: a set of products sold together at a discount (组合套餐).
		`CREATE TABLE IF NOT EXISTS bundles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			bundle_price REAL NOT NULL DEFAULT 0,
			original_total REAL NOT NULL DEFAULT 0,
			product_ids TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		// Restock alerts: users who want to be notified when a product is back in stock (到货通知).
		`CREATE TABLE IF NOT EXISTS restock_alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			notified INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, product_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_restock_product ON restock_alerts(product_id)`,
		// Product Q&A: buyer questions + seller answers (商品问答).
		`CREATE TABLE IF NOT EXISTS product_qa (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			question TEXT NOT NULL,
			answer TEXT NOT NULL DEFAULT '',
			answerer TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_qa_product ON product_qa(product_id)`,
		// Order invoices: electronic invoice requests per order (电子发票).
		`CREATE TABLE IF NOT EXISTS order_invoices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			invoice_type TEXT NOT NULL DEFAULT 'personal',
			title TEXT NOT NULL DEFAULT '',
			tax_no TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'issued',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_user ON order_invoices(user_id)`,
		// Tiered discounts: store-wide spend-X-get-Y-off tiers (阶梯满减).
		`CREATE TABLE IF NOT EXISTS tiered_discounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			threshold REAL NOT NULL DEFAULT 0,
			discount REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		// Lottery prizes: weighted prizes for the points lottery wheel (积分大转盘).
		`CREATE TABLE IF NOT EXISTS lottery_prizes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			points_cost INTEGER NOT NULL DEFAULT 50,
			prize TEXT NOT NULL DEFAULT '',
			probability INTEGER NOT NULL DEFAULT 10,
			icon TEXT NOT NULL DEFAULT '🎁'
		)`,
		// Gift cards: prepaid cards redeemable for store credit (礼品卡).
		`CREATE TABLE IF NOT EXISTS gift_cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			amount REAL NOT NULL DEFAULT 0,
			user_id INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'unused',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gift_cards_user ON gift_cards(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gift_cards_code ON gift_cards(code)`,
		// Price alerts: a user's subscription to be notified when a product drops below a target price (降价提醒).
		`CREATE TABLE IF NOT EXISTS price_alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			target_price REAL NOT NULL DEFAULT 0,
			notified INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, product_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_price_alerts_product ON price_alerts(product_id)`,
		`CREATE INDEX IF NOT EXISTS idx_price_alerts_user ON price_alerts(user_id)`,
		// FTS5 full-text search virtual table over products.
		`CREATE VIRTUAL TABLE IF NOT EXISTS products_fts USING fts5(name, subtitle, category, tags, description, content='products', content_rowid='id')`,
		// Triggers to keep the FTS index in sync with products.
		`CREATE TRIGGER IF NOT EXISTS products_ai AFTER INSERT ON products BEGIN
			INSERT INTO products_fts(rowid, name, subtitle, category, tags, description)
			VALUES (new.id, new.name, new.subtitle, new.category, new.tags, new.description);
		END`,
		`CREATE TRIGGER IF NOT EXISTS products_ad AFTER DELETE ON products BEGIN
			INSERT INTO products_fts(products_fts, rowid, name, subtitle, category, tags, description)
			VALUES ('delete', old.id, old.name, old.subtitle, old.category, old.tags, old.description);
		END`,
		`CREATE TRIGGER IF NOT EXISTS products_au AFTER UPDATE ON products BEGIN
			INSERT INTO products_fts(products_fts, rowid, name, subtitle, category, tags, description)
			VALUES ('delete', old.id, old.name, old.subtitle, old.category, old.tags, old.description);
			INSERT INTO products_fts(rowid, name, subtitle, category, tags, description)
			VALUES (new.id, new.name, new.subtitle, new.category, new.tags, new.description);
		END`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return fmt.Errorf("exec: %w [stmt=%s]", err, truncate(s, 60))
		}
	}
	return migrate()
}

// migrate applies additive post-create schema changes for databases created
// before a feature shipped. Each step is best-effort and idempotent.
func migrate() error {
	// Add the images column to reviews (added after launch) — multi-photo 评价晒图.
	_, _ = DB.Exec(`ALTER TABLE reviews ADD COLUMN images TEXT NOT NULL DEFAULT ''`)
	// Add the remark column to orders (订单备注).
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN remark TEXT NOT NULL DEFAULT ''`)
	// Add the video_url column to products (商品视频介绍).
	_, _ = DB.Exec(`ALTER TABLE products ADD COLUMN video_url TEXT NOT NULL DEFAULT ''`)
	// Add the vip_price column to products (会员价).
	_, _ = DB.Exec(`ALTER TABLE products ADD COLUMN vip_price REAL NOT NULL DEFAULT 0`)
	// Add the useful column to reviews (评价有用数).
	_, _ = DB.Exec(`ALTER TABLE reviews ADD COLUMN useful INTEGER NOT NULL DEFAULT 0`)
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
