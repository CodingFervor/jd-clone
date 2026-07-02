import axios from 'axios'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api/v1',
  timeout: 12000,
})

// Attach the stored JWT to every request.
http.interceptors.request.use((config) => {
  const token = localStorage.getItem('jd_token')
  if (token) config.headers.Authorization = 'Bearer ' + token
  return config
})

// Centralized error toast on 401.
http.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response && err.response.status === 401) {
      localStorage.removeItem('jd_token')
    }
    return Promise.reject(err)
  }
)

function errMsg(e) {
  return e.response?.data?.error || e.message || '网络错误'
}

// ---- Auth ----
export const login = (username, password) => http.post('/auth/login', { username, password }).then((r) => r.data)
export const register = (payload) => http.post('/auth/register', payload).then((r) => r.data)
export const getProfile = () => http.get('/auth/profile').then((r) => r.data)

// ---- Catalog ----
export const getCategories = () => http.get('/categories').then((r) => r.data.data)
export const getProducts = (params) => http.get('/products', { params }).then((r) => r.data)
export const getProduct = (id) => http.get(`/products/${id}`).then((r) => r.data)
export const getSeckill = () => http.get('/products/seckill').then((r) => r.data.data)

// ---- Cart ----
export const getCart = () => http.get('/cart').then((r) => r.data)
export const addToCart = (product_id, quantity = 1) => http.post('/cart', { product_id, quantity }).then((r) => r.data)
export const updateCart = (id, quantity, selected) => http.put(`/cart/${id}`, { quantity, selected }).then((r) => r.data)
export const deleteCart = (id) => http.delete(`/cart/${id}`).then((r) => r.data)

// ---- Orders ----
export const getOrders = () => http.get('/orders').then((r) => r.data.data)
export const createOrder = (payload) => http.post('/orders', payload).then((r) => r.data.data)
export const payOrder = (id) => http.post(`/orders/${id}/pay`).then((r) => r.data)

// ---- Reviews ----
export const createReview = (payload) => http.post('/reviews', payload).then((r) => r.data.data)

// ---- Admin ----
export const adminCreateProduct = (payload) => http.post('/admin/products', payload).then((r) => r.data)
export const adminUpdateProduct = (id, payload) => http.put(`/admin/products/${id}`, payload).then((r) => r.data)
export const adminDeleteProduct = (id) => http.delete(`/admin/products/${id}`).then((r) => r.data)

// ---- SKU ----
export const getSKUs = (productId) => http.get(`/products/${productId}/skus`).then((r) => r.data.data)
export const createSKU = (productId, payload) => http.post(`/admin/products/${productId}/skus`, payload).then((r) => r.data)

// ---- Image upload (multipart) ----
export const uploadImage = (file) => {
  const fd = new FormData()
  fd.append('file', file)
  return http.post('/upload', fd, { headers: { 'Content-Type': 'multipart/form-data' }, timeout: 30000 }).then((r) => r.data)
}

// ---- Refunds ----
export const createRefund = (orderId, reason, type) => http.post('/refunds', { order_id: orderId, reason, type }).then((r) => r.data.data)
export const getRefunds = () => http.get('/refunds').then((r) => r.data.data)

// ---- Coupons ----
export const getCoupons = () => http.get('/coupons').then((r) => r.data.data)
export const claimCoupon = (id) => http.post(`/coupons/${id}/claim`).then((r) => r.data)
export const getMyCoupons = () => http.get('/coupons/mine').then((r) => r.data.data)

// ---- FTS Search ----
export const ftsSearch = (q) => http.get('/search', { params: { q } }).then((r) => r.data)
export const ftsSuggest = (q) => http.get('/search/suggest', { params: { q } }).then((r) => r.data.data)

// ---- Payment ----
export const createPayment = (orderId, method) => http.post('/payments', { order_id: orderId, method }).then((r) => r.data)
export const confirmPayment = (id) => http.post(`/payments/${id}/confirm`).then((r) => r.data)
export const getPayment = (orderId) => http.get(`/payments/order/${orderId}`).then((r) => r.data.payment)

// ---- Shipment ----
export const shipOrder = (orderId) => http.post(`/orders/${orderId}/ship`).then((r) => r.data)
export const trackOrder = (orderId) => http.get(`/orders/${orderId}/track`).then((r) => r.data.shipment)
export const advanceShipment = (orderId) => http.post(`/orders/${orderId}/ship/advance`).then((r) => r.data.shipment)
export const trackByNo = (no) => http.get(`/shipments/track`, { params: { no } }).then((r) => r.data.shipment)

// ---- Favorites (wishlist) ----
export const listFavorites = () => http.get('/favorites').then((r) => r.data.data)
export const toggleFavorite = (productId) => http.post(`/favorites/${productId}`).then((r) => r.data)
export const checkFavorite = (productId) => http.get(`/favorites/${productId}/check`).then((r) => r.data.favorited)

// ---- Addresses ----
export const getAddresses = () => http.get('/addresses').then((r) => r.data.data)
export const createAddress = (payload) => http.post('/addresses', payload).then((r) => r.data.data)
export const updateAddress = (id, payload) => http.put(`/addresses/${id}`, payload).then((r) => r.data)
export const deleteAddress = (id) => http.delete(`/addresses/${id}`).then((r) => r.data)

// ---- Order lifecycle ----
export const confirmOrder = (orderId) => http.post(`/orders/${orderId}/confirm`).then((r) => r.data)

// ---- Profile ----
export const updateProfile = (payload) => http.put('/auth/profile', payload).then((r) => r.data.data)

// ---- Browse history ----
export const getHistory = (limit) => http.get('/history', { params: { limit } }).then((r) => r.data.data)
export const clearHistory = () => http.delete('/history').then((r) => r.data)

// ---- Daily check-in ----
export const doCheckIn = () => http.post('/checkin').then((r) => r.data)
export const getCheckInStatus = () => http.get('/checkin/status').then((r) => r.data)

// ---- Points mall ----
export const getPointShop = () => http.get('/points/shop').then((r) => r.data)
export const redeemPoints = (id) => http.post(`/points/shop/${id}/redeem`).then((r) => r.data)
export const getRedemptions = () => http.get('/points/redemptions').then((r) => r.data.data)

// ---- Review replies ----
export const replyReview = (reviewId, content) => http.post('/reviews/reply', { review_id: reviewId, content }).then((r) => r.data.data)

// ---- Seckill deals (秒杀) ----
export const getSeckillDeals = () => http.get('/seckill').then((r) => r.data.data)
export const grabSeckill = (id) => http.post(`/seckill/${id}/grab`).then((r) => r.data)

export { errMsg }
