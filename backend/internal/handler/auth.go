package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
	"github.com/CodingFervor/jd-clone/backend/internal/repository"
)

// Handler bundles all repositories and shares the JWT secret.
type Handler struct {
	User     *repository.UserRepo
	Cat      *repository.CategoryRepo
	Product  *repository.ProductRepo
	Cart     *repository.CartRepo
	Order    *repository.OrderRepo
	Review   *repository.ReviewRepo
	Address  *repository.AddressRepo
	SKU      *repository.SKURepo
	Payment  *repository.PaymentRepo
	Shipment *repository.ShipmentRepo
	Refund   *repository.RefundRepo
	Coupon   *repository.CouponRepo
	Favorite *repository.FavoriteRepo
	History  *repository.HistoryRepo
	CheckIn  *repository.CheckInRepo
	jwtKey   []byte
}

func New(jwtSecret string, u *repository.UserRepo, c *repository.CategoryRepo, p *repository.ProductRepo,
	ca *repository.CartRepo, o *repository.OrderRepo, r *repository.ReviewRepo, a *repository.AddressRepo) *Handler {
	return &Handler{User: u, Cat: c, Product: p, Cart: ca, Order: o, Review: r, Address: a, jwtKey: []byte(jwtSecret)}
}

// SetUserExtra attaches the favorite repo (wishlist support).
func (h *Handler) SetUserExtra(fav *repository.FavoriteRepo) {
	h.Favorite = fav
}

// SetHistory attaches the browse-history + check-in repos.
func (h *Handler) SetHistory(hist *repository.HistoryRepo, ci *repository.CheckInRepo) {
	h.History = hist
	h.CheckIn = ci
}

// ---- JWT (HS256, hand-rolled, no external dep) ----

func (h *Handler) signToken(userID int64, username string) (string, error) {
	header := `{"alg":"HS256","typ":"JWT"}`
	payload := `{"user_id":` + strconv.FormatInt(userID, 10) + `,"username":"` + username + `","exp":` + strconv.FormatInt(time.Now().Add(72*time.Hour).Unix(), 10) + `}`
	return encodeSegment(header) + "." + encodeSegment(payload) + "." + h.signature(header, payload), nil
}

func (h *Handler) parseToken(token string) (int64, string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, "", false
	}
	if h.signature(decodeSegment(parts[0]), decodeSegment(parts[1])) != parts[2] {
		return 0, "", false
	}
	payload := decodeSegment(parts[1])
	// crude extraction of user_id and username
	uid := extractInt(payload, "user_id")
	uname := extractStr(payload, "username")
	exp := extractInt(payload, "exp")
	if exp > 0 && time.Now().Unix() > exp {
		return 0, "", false
	}
	return uid, uname, true
}

func (h *Handler) signature(header, payload string) string {
	sum := sha256.Sum256([]byte(header + "." + payload + "." + string(h.jwtKey)))
	return hex.EncodeToString(sum[:])
}

// currentUserID reads the auth header and returns the user id; on failure it
// aborts with 401 and returns false.
// currentUserID returns the user id from the bearer token. When optional is
// true, a missing/invalid token returns (0, true) instead of 401 — used by
// endpoints that serve both anonymous and authenticated users (e.g. coupons).
func (h *Handler) currentUserID(c *gin.Context, optional ...bool) (int64, bool) {
	auth := c.GetHeader("Authorization")
	tok := strings.TrimPrefix(auth, "Bearer ")
	uid, _, ok := h.parseToken(tok)
	if !ok {
		if len(optional) > 0 && optional[0] {
			return 0, true
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return 0, false
	}
	return uid, true
}

func encodeSegment(s string) string {
	return base64URL(s)
}

func decodeSegment(s string) string {
	return base64URLDecode(s)
}

func base64URL(s string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	b := []byte(s)
	out := make([]byte, 0, len(b)*4/3+4)
	for i := 0; i < len(b); i += 3 {
		var n uint32
		var cnt int
		for j := 0; j < 3; j++ {
			if i+j < len(b) {
				n = n<<8 | uint32(b[i+j])
				cnt++
			} else {
				n <<= 8
			}
		}
		n <<= uint(8 * (3 - cnt))
		out = append(out, alphabet[(n>>18)&63])
		out = append(out, alphabet[(n>>12)&63])
		if cnt > 1 {
			out = append(out, alphabet[(n>>6)&63])
		}
		if cnt > 2 {
			out = append(out, alphabet[n&63])
		}
	}
	return string(out)
}

func base64URLDecode(s string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	idx := make([]byte, 256)
	for i := 0; i < 256; i++ {
		idx[i] = 255
	}
	for i := 0; i < 64; i++ {
		idx[alphabet[i]] = byte(i)
	}
	out := make([]byte, 0, len(s)*3/4)
	var buf uint32
	var bits int
	for i := 0; i < len(s); i++ {
		v := idx[s[i]]
		if v == 255 {
			continue
		}
		buf = buf<<6 | uint32(v)
		bits += 6
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(buf>>uint(bits)&0xFF))
		}
	}
	return string(out)
}

func extractInt(json, key string) int64 {
	needle := `"` + key + `":`
	i := strings.Index(json, needle)
	if i < 0 {
		return 0
	}
	rest := json[i+len(needle):]
	end := 0
	for end < len(rest) {
		ch := rest[end]
		if (ch < '0' || ch > '9') && ch != '-' {
			break
		}
		end++
	}
	n, _ := strconv.ParseInt(rest[:end], 10, 64)
	return n
}

func extractStr(json, key string) string {
	needle := `"` + key + `":"`
	i := strings.Index(json, needle)
	if i < 0 {
		return ""
	}
	rest := json[i+len(needle):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// hashPassword is a simple SHA-256 hash (demo only; use bcrypt in production).
func hashPassword(plain string) string {
	sum := sha256.Sum256([]byte(plain + "jd-salt"))
	return hex.EncodeToString(sum[:])
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- Auth endpoints ----

func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码必填"})
		return
	}
	u, err := h.User.FindByUsername(req.Username)
	if err != nil || u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	// Support both seeded plaintext and hashed passwords.
	if u.Password != req.Password && u.Password != hashPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	tok, _ := h.signToken(u.ID, u.Username)
	c.JSON(http.StatusOK, gin.H{"token": tok, "user": u})
}

func (h *Handler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	if h.User.Exists(req.Username) {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	u := &model.User{Username: req.Username, Password: hashPassword(req.Password), Nickname: req.Nickname}
	if u.Nickname == "" {
		u.Nickname = req.Username
	}
	u.Avatar = "https://api.dicebear.com/7.x/avataaars/svg?seed=" + req.Username
	if err := h.User.Create(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "注册失败"})
		return
	}
	tok, _ := h.signToken(u.ID, u.Username)
	c.JSON(http.StatusOK, gin.H{"token": tok, "user": u})
}

func (h *Handler) Profile(c *gin.Context) {
	uid, ok := h.currentUserID(c)
	if !ok {
		return
	}
	u, err := h.User.Get(uid)
	if err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	count, _ := h.Cart.Count(uid)
	c.JSON(http.StatusOK, gin.H{"user": u, "cart_count": count})
}
