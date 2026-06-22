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

// ---- Payment ----
export const createPayment = (orderId, method) => http.post('/payments', { order_id: orderId, method }).then((r) => r.data)
export const confirmPayment = (id) => http.post(`/payments/${id}/confirm`).then((r) => r.data)
export const getPayment = (orderId) => http.get(`/payments/order/${orderId}`).then((r) => r.data.payment)

// ---- Shipment ----
export const shipOrder = (orderId) => http.post(`/orders/${orderId}/ship`).then((r) => r.data)
export const trackOrder = (orderId) => http.get(`/orders/${orderId}/track`).then((r) => r.data.shipment)
export const advanceShipment = (orderId) => http.post(`/orders/${orderId}/ship/advance`).then((r) => r.data.shipment)
export const trackByNo = (no) => http.get(`/shipments/track`, { params: { no } }).then((r) => r.data.shipment)

export { errMsg }
