# JD Clone | 京东商城仿制

## English | [中文](#中文)

A JD.com (京东) clone — a full-stack mobile e-commerce app with a Go + Gin backend and a Vue 3 + Vant frontend, backed by SQLite. The server auto-seeds mock products/categories/reviews on first run, so the app is usable immediately.

### Features
- **Home feed** — search bar, banner, category grid, flash-sale (秒杀) floor, product waterfall
- **Category browsing** — JD-style left-sidebar category tree with product list
- **Product detail** — gallery, price, shop, specs, reviews with rating
- **Shopping cart** — select/deselect, change quantity, totals
- **Orders** — place order (cart or buy-now), pay, order history
- **Auth** — register/login with JWT, profile with order entries
- **Search** — keyword search with history & hot terms
- **Admin panel** — product CRUD (create/edit/delete) with swipe cells

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
| POST | /api/v1/auth/login | Login |
| POST | /api/v1/auth/register | Register |
| GET | /api/v1/auth/profile | Current user |
| GET | /api/v1/categories | All categories |
| GET | /api/v1/products | List products (?page=&page_size=&category_id=&q=) |
| GET | /api/v1/products/:id | Product detail + reviews |
| GET | /api/v1/products/seckill | Flash-sale items |
| GET/POST/PUT/DELETE | /api/v1/cart | Cart (auth) |
| GET/POST | /api/v1/orders | Orders (auth); POST /orders/:id/pay to pay |
| POST | /api/v1/reviews | Add review (auth) |
| POST/PUT/DELETE | /api/v1/admin/products | Product CRUD |

### License
MIT — see [LICENSE](LICENSE).

---

<a id="中文"></a>
# 京东商城仿制

京东商城（JD.COM）仿制 —— 全栈移动电商 App，Go + Gin 后端 + Vue 3 + Vant 前端，SQLite 存储。后端首次启动自动填充 mock 商品/分类/评价，开箱即用。

### 功能特性
- **首页** — 搜索栏、Banner、分类宫格、京东秒杀楼层、商品瀑布流
- **分类** — 京东风格左侧分类树 + 右侧商品列表
- **商品详情** — 图册、价格、店铺、规格、带评分评价
- **购物车** — 勾选/取消、改数量、合计
- **订单** — 下单（购物车/立即购买）、付款、订单列表
- **登录注册** — JWT 鉴权、个人中心含订单入口
- **搜索** — 关键词搜索 + 历史记录 + 热门词
- **管理后台** — 商品增删改查（滑动操作）

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
