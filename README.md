# JD Clone | 京东商城仿制

## English | [中文](#中文)

A JD.com (京东) clone — a full-stack mobile e-commerce app with a Go + Gin backend and a Vue 3 + Vant frontend, backed by SQLite. The server auto-seeds mock products/categories/reviews on first run, so the app is usable immediately.

### Features
- **Home feed** — search bar, banner, category grid, flash-sale (秒杀) floor, product waterfall
- **Category browsing** — JD-style left-sidebar category tree with product list
- **Product detail** — gallery, price, shop, **SKU spec selector** (color/storage/size), reviews with rating + photo upload
- **Shopping cart** — select/deselect, change quantity, totals
- **Sandbox payment** — cashier page with WeChat/Alipay/UnionPay, payment confirmation callback, order status machine (pending→paid)
- **Shipment tracking** — auto-generated tracking number + 4-step logistics trajectory (出库→中转→发往→分拣), advance through in_transit→delivered, logistics timeline page
- **After-sale refunds** — apply for refund from orders, status tracking (pending→approved/rejected→completed), admin approval
- **Coupon marketing** — coupon center with 满666 (满减) + 折扣 (discount) types, claim/used tracking, "my coupons" view; 5 seeded coupons
- **FTS5 full-text search** — SQLite FTS5-powered product search with auto-complete suggestions (LIKE fallback if FTS unavailable)
- **Image upload** — real multipart upload for product images (admin) and review photos (晒图)
- **Auth** — register/login with JWT, profile with order entries
- **Admin panel** — product CRUD + SKU management + image upload

### Tech Stack
- **Backend**: Go 1.22 + Gin + SQLite (`modernc.org/sqlite`, pure-Go, CGO-free static build)
- **Frontend**: Vue 3 + Vite + Vant 4 (mobile UI) + Vue Router + Axios
- **Deploy**: Docker Compose (backend + nginx frontend) with a SQLite volume

### Project Structure
```
jd-clone/
├── backend/
│   ├── cmd/server/main.go        # entry: init DB → seed → routes → graceful shutdown
│   └── internal/
│       ├── config/               # env config
│       ├── db/                   # SQLite connection + schema
│       ├── seed/                 # idempotent mock data
│       ├── model/                # entities + DTOs
│       ├── repository/           # SQL data access
│       ├── handler/              # HTTP handlers (auth + shop)
│       └── server/               # gin routes + CORS
├── frontend/
│   └── src/
│       ├── views/                # Home/Category/Cart/Detail/Orders/...
│       ├── api/                  # axios client
│       └── router/
├── docker-compose.yml
└── README.md
```

### Quick Start

#### Docker Compose (full stack)
```bash
docker-compose up -d --build
# Frontend: http://localhost
# API:      http://localhost:8080
```

#### Run separately (development)
```bash
# Terminal 1 — backend
cd backend
go run ./cmd/server          # :8080, auto-seeds DB

# Terminal 2 — frontend (dev server proxies /api → :8080)
cd frontend
npm install
npm run dev                  # :5173
```

### Demo Account
`admin` / `admin123` (auto-seeded)

### API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/auth/login \| /register | Auth |
| GET | /api/v1/auth/profile | Current user (auth) |
| GET | /api/v1/categories | All categories |
| GET | /api/v1/products | List (?page=&page_size=&category_id=&q=) |
| GET | /api/v1/products/:id | Detail + reviews + SKUs |
| GET | /api/v1/products/seckill | Flash-sale items |
| GET | /api/v1/products/:id/skus | Product SKU specs |
| GET | /api/v1/search | FTS5 full-text search (?q=) |
| GET | /api/v1/search/suggest | Search auto-complete (?q=) |
| GET/POST/PUT/DELETE | /api/v1/cart | Cart (auth) |
| GET/POST | /api/v1/orders | Orders (auth) |
| POST | /api/v1/payments | Create payment (auth) |
| POST | /api/v1/payments/:id/confirm | Confirm payment (sandbox callback) |
| POST | /api/v1/orders/:id/ship | Ship order → tracking no + trajectory |
| GET | /api/v1/orders/:id/track | Shipment tracking timeline |
| POST | /api/v1/orders/:id/ship/advance | Advance logistics status |
| POST | /api/v1/refunds | Apply for refund (auth) |
| GET | /api/v1/refunds | List my refunds (auth) |
| GET | /api/v1/coupons | Available coupons |
| POST | /api/v1/coupons/:id/claim | Claim coupon (auth) |
| GET | /api/v1/coupons/mine | My coupons (auth) |
| POST | /api/v1/reviews | Add review with photos (auth) |
| POST | /api/v1/upload | Image upload — multipart (auth) |
| POST/PUT/DELETE | /api/v1/admin/products | Product CRUD |
| POST | /api/v1/admin/products/:id/skus | Add SKU variant |

### License
MIT — see [LICENSE](LICENSE).

---

<a id="中文"></a>
# 京东商城仿制

京东商城（JD.COM）仿制 —— 全栈移动电商 App，Go + Gin 后端 + Vue 3 + Vant 前端，SQLite 存储。后端首次启动自动填充 mock 商品/分类/评价，开箱即用。

### 功能特性
- **首页** — 搜索栏、Banner、分类宫格、京东秒杀楼层、商品瀑布流
- **分类** — 京东风格左侧分类树 + 右侧商品列表
- **商品详情** — 图册、价格、店铺、**SKU 规格选择**（颜色/存储/尺码）、带评分评价 + 晒图上传
- **购物车** — 勾选/取消、改数量、合计
- **沙箱支付** — 收银台页面（微信/支付宝/银联）、支付确认回调、订单状态机（待付款→已付款）
- **物流跟踪** — 自动生成运单号 + 4 步京东物流轨迹（出库→中转→发往→分拣）、可推进运输→送达、物流时间线页面
- **售后退货** — 订单页申请退款、状态跟踪（审核中→已通过/已拒绝→已完成）、管理员审核
- **优惠券营销** — 领券中心（满减券 + 折扣券）、领取/已用追踪、"我的优惠券"；预置 5 张优惠券
- **FTS5 全文搜索** — SQLite FTS5 驱动的商品搜索 + 输入联想（FTS 不可用时自动降级 LIKE）
- **图片上传** — 商品图片（后台上传）+ 评价晒图（用户多图上传）真实 multipart 文件上传
- **登录注册** — JWT 鉴权、个人中心含订单/售后/优惠券入口
- **管理后台** — 商品增删改查 + SKU 管理 + 图片上传

### 技术栈
- **后端**：Go 1.22 + Gin + SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO-free 静态构建）
- **前端**：Vue 3 + Vite + Vant 4（移动端 UI）+ Vue Router + Axios
- **部署**：Docker Compose（后端 + nginx 前端）+ SQLite 数据卷

### 快速开始

#### Docker Compose（一键全栈）
```bash
docker-compose up -d --build
# 前端：http://localhost
# API：http://localhost:8080
```

#### 分别运行（开发模式）
```bash
# 终端 1 — 后端
cd backend
go run ./cmd/server          # :8080，自动建表+填充数据

# 终端 2 — 前端（dev 自动代理 /api → :8080）
cd frontend
npm install
npm run dev                  # :5173
```

### 演示账号
`admin` / `admin123`（启动自动创建）

API 端点详见上方英文版表格。

### 开源许可
MIT — 详见 [LICENSE](LICENSE)。
