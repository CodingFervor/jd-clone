<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showSuccessToast, showToast, showDialog } from 'vant'
import { getProduct, addToCart, createOrder, createReview, uploadImage, checkFavorite, toggleFavorite, replyReview, getPriceHistory, checkRestock, subscribeRestock, unsubscribeRestock, getProductQA, askProductQA, markReviewUseful, subscribePriceAlert, checkPriceAlert } from '../api'

const route = useRoute()
const router = useRouter()
const product = ref(null)
const reviews = ref([])
const skus = ref([])
const selectedSKU = ref(null)
const recommendedSKU = ref(null)
// Estimated delivery (预计送达) — purely frontend, computed once on load.
const deliveryEstimate = ref('')
// Price history (比价历史)
const priceHistory = ref([])
const priceStats = ref(null)
const showPriceHistory = ref(false)
const showPoster = ref(false)
const restockSubscribed = ref(false)
// Price drop alert (降价提醒)
const priceAlert = ref(null)
const showPriceAlert = ref(false)
const alertTargetPrice = ref('')
// Product Q&A
const qaList = ref([])
const showQA = ref(false)
const qaQuestion = ref('')
const relatedProducts = ref([])
const loading = ref(true)
// ---- Feature: 商品视频自动播放 (Product Video Auto-play) ----
// The product intro video auto-plays (muted) once scrolled 50% into view via
// an IntersectionObserver, pauses when scrolled away, and exposes a manual
// play/pause toggle plus a tap-to-unmute overlay. All refs degrade gracefully
// when the product has no video_url (the whole video block is v-if-guarded).
const videoRef = ref(null) // bound to the <video> element
const videoMuted = ref(true) // auto-play must start muted per browser policy
const videoPlaying = ref(false) // mirrors the video's playing/paused state
const videoVisible = ref(false) // whether the video is currently in the viewport
let videoObserver = null

// Toggle between play and pause (manual control button). Preferred over the
// native controls overlay so the toggle state stays in sync with auto-play.
function toggleVideoPlay() {
  const v = videoRef.value
  if (!v) return
  if (v.paused) {
    v.play().catch(() => { /* autoplay can be blocked; ignore */ })
  } else {
    v.pause()
  }
}

// Unmute the video (the auto-play started muted). Triggered by the 🔇 overlay
// tap so we keep the user-gesture requirement browsers enforce for sound.
function unmuteVideo() {
  const v = videoRef.value
  videoMuted.value = false
  if (v) v.muted = false
}

// Bind native media events to our reactive flags so the UI stays accurate
// even when autoplay or the OS media keys change playback state.
function onVideoPlay() { videoPlaying.value = true }
function onVideoPause() { videoPlaying.value = false }
function onVideoEnded() { videoPlaying.value = false }
const showReview = ref(false)
const reviewRating = ref(5)
const reviewContent = ref('')
const reviewImages = ref([])
// Video clip URL attached to a review (视频评价). Optional; stored alongside
// the photos using the "video:" prefix convention in the images field.
const reviewVideo = ref('')
// Toggle the video URL input box inside the review form.
const showVideoInput = ref(false)
const favorited = ref(false)
// Build the gallery list from the product's images field (comma-separated),
// falling back to the single main image.
const gallery = computed(() => {
  if (!product.value) return []
  const imgs = (product.value.images || '').split(',').map((s) => s.trim()).filter(Boolean)
  if (imgs.length) return imgs
  return product.value.image ? [product.value.image] : []
})
// Review summary stats (评价概览统计): computed from the loaded reviews ref.
// Computes total count, average rating (1 decimal), good rate (4-5 star share),
// and the per-star distribution used to render the horizontal bars.
const reviewStats = computed(() => {
  const list = reviews.value
  const total = list.length
  if (!total) return { total: 0, avg: '0.0', goodRate: 0, dist: [0, 0, 0, 0, 0] }
  let sum = 0
  let good = 0
  const dist = [0, 0, 0, 0, 0] // index 0 → 1★, index 4 → 5★
  for (const r of list) {
    const s = Math.max(1, Math.min(5, Math.round(Number(r.rating) || 0)))
    sum += s
    dist[s - 1]++
    if (s >= 4) good++
  }
  return {
    total,
    avg: (sum / total).toFixed(1),
    goodRate: Math.round((good / total) * 100),
    dist,
  }
})
// Star distribution bars ordered 1★ (top) → 5★ (bottom), bar widths scaled
// relative to the busiest star level so the longest bar is full width.
const reviewDistBars = computed(() => {
  const dist = reviewStats.value.dist
  const max = Math.max(1, ...dist)
  return [1, 2, 3, 4, 5].map((star) => ({
    star,
    count: dist[star - 1],
    pct: Math.round((dist[star - 1] / max) * 100),
  }))
})
// Review filter tabs (评价筛选标签): tracks the active filter chip and feeds
// the count badges + the filtered list shown in the v-for below.
const reviewFilter = ref('all')
// Each entry carries a `match` predicate applied against a review record, plus
// a static label; the count is filled in dynamically from the live reviews.
const REVIEW_FILTERS = [
  { key: 'all', label: '全部', match: () => true },
  { key: 'good', label: '好评', match: (r) => Math.round(Number(r.rating) || 0) >= 4 },
  { key: 'mid', label: '中评', match: (r) => Math.round(Number(r.rating) || 0) === 3 },
  { key: 'bad', label: '差评', match: (r) => Math.round(Number(r.rating) || 0) <= 2 },
  { key: 'photo', label: '有图', match: (r) => reviewMedia(r.images).photos.length > 0 },
]
// The chips to render: original config plus a live count per filter.
const reviewFilters = computed(() =>
  REVIEW_FILTERS.map((f) => ({
    ...f,
    count: reviews.value.filter(f.match).length,
  }))
)
// Active filter's predicate (defaults to "all" which matches everything).
const activeFilter = computed(() => {
  const f = REVIEW_FILTERS.find((x) => x.key === reviewFilter.value)
  return f ? f.match : () => true
})
// Filtered review list driven by the active chip; used in the v-for loop.
const filteredReviews = computed(() => reviews.value.filter(activeFilter.value))
// Spec matrix table (商品规格矩阵表): transforms the SKU list into a table
// with one column per spec dimension plus price/stock. Only shown when SKUs
// expose at least 2 parseable dimensions; otherwise the tag selector alone is
// enough. Each SKU.spec is a JSON string like {"颜色":"黑色","版本":"256GB"}.
const specMatrix = computed(() => {
  const parsed = skus.value.map((s) => {
    let dims = {}
    try {
      const obj = JSON.parse(s.spec || '{}')
      if (obj && typeof obj === 'object' && !Array.isArray(obj)) dims = obj
    } catch {
      dims = {}
    }
    return { sku: s, dims }
  })
  // Collect dimension names in first-seen order.
  const dimNames = []
  const seen = new Set()
  for (const p of parsed) {
    for (const k of Object.keys(p.dims)) {
      if (!seen.has(k)) {
        seen.add(k)
        dimNames.push(k)
      }
    }
  }
  return { dimNames, rows: parsed }
})
// Eco Score (环保评分): deterministic 0-100 score derived from a hash of
// the product id so the same product always shows the same score. Used to
// render a color-coded environmental impact badge in the spec cell group.
function hashId(str) {
  let h = 0
  for (let i = 0; i < str.length; i++) {
    h = (h << 5) - h + str.charCodeAt(i)
    h |= 0
  }
  return Math.abs(h)
}
const ecoScore = computed(() => {
  const id = product.value ? product.value.id : 0
  return hashId(String(id)) % 101
})
const ecoLevel = computed(() => {
  const s = ecoScore.value
  if (s >= 90) return '优秀'
  if (s >= 70) return '良好'
  if (s >= 60) return '合格'
  return '一般'
})
const ecoColor = computed(() => {
  const s = ecoScore.value
  if (s >= 90) return '#07c160' // green
  if (s >= 70) return '#95d475' // light-green
  if (s >= 60) return '#ff976a' // orange
  return '#999' // gray
})
// Brand Story (品牌故事): a collapsible section above the description. The
// story text is templated from the shop name; the three "品牌理念" tags are
// deterministically picked from a pool based on the product id hash.
const brandStoryCollapsed = ref(true)
const brandStory = computed(() => {
  const shop = product.value && product.value.shop ? product.value.shop : '本品牌'
  return `${shop}致力于为消费者提供优质的产品和服务。自创立以来，始终坚持品质第一的理念，用心打磨每一个细节。`
})
const BRAND_TAG_POOL = ['品质保证', '正品行货', '售后无忧', '极速配送', '用户至上', '匠心工艺']
const brandTags = computed(() => {
  const id = product.value ? product.value.id : 0
  const seed = hashId(String(id))
  const picked = []
  const pool = BRAND_TAG_POOL.slice()
  for (let i = 0; i < 3 && pool.length; i++) {
    const idx = (seed + i * 7) % pool.length
    picked.push(pool.splice(idx, 1)[0])
  }
  return picked
})
// Product Origin (产地溯源): a deterministic fake origin city chosen from a
// fixed pool by product.id % 10. Each city carries its province prefix so the
// cell can render "浙江·杭州". Clicking the cell opens a popup showing a
// stylized route from the origin to the user.
const ORIGIN_CITIES = [
  { city: '北京', province: '北京' },
  { city: '上海', province: '上海' },
  { city: '广州', province: '广东' },
  { city: '深圳', province: '广东' },
  { city: '杭州', province: '浙江' },
  { city: '成都', province: '四川' },
  { city: '武汉', province: '湖北' },
  { city: '西安', province: '陕西' },
  { city: '南京', province: '江苏' },
  { city: '重庆', province: '重庆' },
]
const origin = computed(() => {
  const id = product.value ? Number(product.value.id) || 0 : 0
  const entry = ORIGIN_CITIES[id % ORIGIN_CITIES.length]
  return entry
})
const originLabel = computed(() => {
  const o = origin.value
  return o ? `${o.province}·${o.city}` : ''
})
const showOrigin = ref(false)
// Purchase Note Templates (购买须知): three templated notes shown in a
// collapsible section above the brand story. The start index is rotated by the
// product id so the same product always shows the notes in the same order,
// while different products present them differently.
const PURCHASE_NOTES = [
  '本品支持7天无理由退货',
  '颜色可能因屏幕显示略有差异',
  '请确认规格后下单，拆封后不支持退换',
]
const purchaseNotesCollapsed = ref(true)
const purchaseNotes = computed(() => {
  const id = product.value ? Number(product.value.id) || 0 : 0
  const start = id % PURCHASE_NOTES.length
  const out = []
  for (let i = 0; i < PURCHASE_NOTES.length; i++) {
    out.push(PURCHASE_NOTES[(start + i) % PURCHASE_NOTES.length])
  }
  return out
})
async function onUploadReviewImage(item) {
  try {
    const res = await uploadImage(item.file)
    reviewImages.value.push(res.url)
  } catch (e) {
    showToast('图片上传失败')
  }
}
function removeReviewImage(idx) {
  reviewImages.value.splice(idx, 1)
}

onMounted(async () => {
  try {
    const res = await getProduct(route.params.id)
    product.value = res.data
    reviews.value = res.reviews || []
    skus.value = res.skus || []
    // ---- 比价悬浮球: remember the last-viewed product so the Home page can
    // surface a floating price-tracking widget. We snapshot the id, name,
    // thumbnail and the price the user saw at this visit. On the next Home
    // visit the widget compares this stored price with the live one.
    try {
      const p = res.data
      if (p && p.id) {
        localStorage.setItem(
          'jd_last_viewed',
          JSON.stringify({
            id: p.id,
            name: p.name,
            image: p.image,
            price: Number(p.price) || 0,
            original_price: Number(p.original_price) || 0,
            visited_at: Date.now(),
          })
        )
      }
    } catch (_) {
      // localStorage may be unavailable (private mode); ignore.
    }
    // Compute the estimated delivery label once on load.
    deliveryEstimate.value = computeDelivery()
    // Auto-select the recommended (best-value) SKU.
    if (res.recommended_sku) {
      selectedSKU.value = res.recommended_sku
      recommendedSKU.value = res.recommended_sku
    }
    relatedProducts.value = res.related || []
    if (localStorage.getItem('jd_token')) {
      favorited.value = await checkFavorite(route.params.id)
    }
    // Load price history (best-effort).
    getPriceHistory(route.params.id).then((d) => { priceHistory.value = d.data || []; priceStats.value = d.stats }).catch(() => {})
    // Check restock subscription.
    if (localStorage.getItem('jd_token')) {
      checkRestock(route.params.id).then((s) => { restockSubscribed.value = s }).catch(() => {})
    }
    // Check price-drop alert subscription (降价提醒).
    if (localStorage.getItem('jd_token')) {
      checkPriceAlert(route.params.id).then((d) => { priceAlert.value = d.data }).catch(() => {})
    }
    // Load Q&A.
    getProductQA(route.params.id).then((d) => { qaList.value = d || [] }).catch(() => {})
  } catch (e) {
    showToast('商品不存在')
  } finally {
    loading.value = false
    // Wire up the sticky tab IntersectionObserver once the DOM has rendered.
    nextTick(() => {
      setupSectionObserver()
      setupVideoObserver()
    })
  }
})

async function doFavorite() {
  if (!checkLogin()) return
  try {
    const res = await toggleFavorite(product.value.id)
    favorited.value = res.favorited
    showSuccessToast(res.favorited ? '已收藏' : '已取消收藏')
  } catch (e) {
    showToast(e.response?.data?.error || '操作失败')
  }
}

function selectSKU(sku) {
  selectedSKU.value = sku
}
// Effective price: the chosen SKU's price, else the product's.
function currentPrice() {
  return selectedSKU.value ? selectedSKU.value.price : (product.value ? product.value.price : 0)
}
// Effective stock: the chosen SKU's stock, else the product's.
function currentStock() {
  if (selectedSKU.value && typeof selectedSKU.value.stock === 'number') return selectedSKU.value.stock
  return product.value && typeof product.value.stock === 'number' ? product.value.stock : 0
}
// Stock pressure meter (库存紧张指示): maps the effective stock to a label,
// color, progress-bar width and whether it should blink. The "flame" emoji
// shows up for low-stock items to add urgency.
const stockMeter = computed(() => {
  const s = currentStock()
  if (s > 100) return { label: '库存充足', color: '#07c160', bar: 0, blink: false, flame: false }
  if (s >= 20) return { label: '有货', color: '#07c160', bar: 50, blink: false, flame: false }
  if (s >= 5) return { label: '库存紧张', color: '#ff976a', bar: 80, blink: false, flame: true }
  if (s >= 1) return { label: '仅剩' + s + '件', color: '#e1251b', bar: 95, blink: true, flame: true }
  return { label: '暂时缺货', color: '#999', bar: 0, blink: false, flame: false }
})

async function doAddCart() {
  if (!checkLogin()) return
  try {
    await addToCart(product.value.id, 1)
    showSuccessToast('已加入购物车')
  } catch (e) {
    showToast(e.response?.data?.error || '失败')
  }
}
async function buyNow() {
  if (!checkLogin()) return
  try {
    const order = await createOrder({ items: [{ product_id: product.value.id, quantity: 1 }], address: '' })
    showSuccessToast('下单成功')
    router.push('/orders')
  } catch (e) {
    showToast(e.response?.data?.error || '下单失败')
  }
}
function checkLogin() {
  if (!localStorage.getItem('jd_token')) {
    showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login'))
    return false
  }
  return true
}
async function submitReview() {
  if (!reviewContent.value.trim()) {
    showToast('请输入评价内容')
    return
  }
  // Combine photos and the optional video clip into one images field.
  // Video URLs are prefixed with "video:" so the display layer can tell them
  // apart from photos (e.g. "a.jpg,b.jpg,video:https://...").
  const parts = [...reviewImages.value]
  const vid = reviewVideo.value.trim()
  if (vid) parts.push('video:' + vid)
  try {
    const rv = await createReview({ product_id: product.value.id, rating: reviewRating.value, content: reviewContent.value, images: parts.join(',') })
    reviews.value.unshift(rv)
    showReview.value = false
    reviewContent.value = ''
    reviewImages.value = []
    reviewVideo.value = ''
    showVideoInput.value = false
    showSuccessToast('评价成功')
  } catch (e) {
    showToast('请先登录')
  }
}
const replyingTo = ref(null)
const replyText = ref('')
function toggleReply(r) {
  replyingTo.value = replyingTo.value === r.id ? null : r.id
  replyText.value = ''
}
async function submitReply(r) {
  if (!replyText.value.trim()) { showToast('请输入回复内容'); return }
  try {
    const rep = await replyReview(r.id, replyText.value)
    r.reply = rep
    replyingTo.value = null
    replyText.value = ''
    showSuccessToast('回复成功')
  } catch (e) {
    showToast('请先登录')
  }
}
async function doUseful(r) {
  if (!checkLogin()) return
  try {
    await markReviewUseful(r.id)
    r.useful = (r.useful || 0) + 1
    showSuccessToast('已标记有用')
  } catch (e) {
    showToast('请先登录')
  }
}
function fmt(n) {
  return Number(n).toFixed(2)
}
// Parse a review's images field into photos and video entries.
// Photos have no prefix; video URLs are prefixed with "video:".
// Returns { photos: [], videos: [] } with plain URL strings.
function reviewMedia(images) {
  const photos = []
  const videos = []
  if (!images) return { photos, videos }
  for (const part of String(images).split(',')) {
    const s = part.trim()
    if (!s) continue
    if (s.startsWith('video:')) {
      videos.push(s.slice('video:'.length))
    } else {
      photos.push(s)
    }
  }
  return { photos, videos }
}
function goProduct(id) {
  router.replace('/product/' + id)
  // Force reload by re-running onMounted logic.
  setTimeout(() => window.location.reload(), 50)
}
// Estimate delivery based on the current time of day (预计送达).
// Before 11am → today, before 5pm → tomorrow, otherwise the day after tomorrow.
function computeDelivery() {
  const h = new Date().getHours()
  if (h < 11) return '今日达'
  if (h < 17) return '明日达'
  return '后天达'
}
// Deterministic pseudo-QR pattern for the share poster (purely decorative).
function qrPattern(n) {
  const row = Math.floor((n - 1) / 8)
  const col = (n - 1) % 8
  // Corner finder squares.
  const corner = (row < 2 || row > 5) && (col < 2 || col > 5)
  return corner || ((row * 7 + col * 3 + n) % 3 === 0)
}
async function copyShareLink() {
  const link = window.location.href
  try {
    await navigator.clipboard.writeText(link)
    showSuccessToast('链接已复制')
  } catch (e) {
    showToast('复制失败，请手动复制')
  }
}

// ---- Share long image (分享长图) ----
// Builds a CSS "long card" preview inside the share poster popup and offers
// a "保存图片" action that, since we can't easily do real canvas capture
// here, copies a formatted text summary of the product instead.
const showLongImage = ref(false)
// Key specs surfaced on the long card: shop, sales, tags, delivery, origin.
const longCardSpecs = computed(() => {
  if (!product.value) return []
  const specs = []
  specs.push({ label: '店铺', value: product.value.shop || '京东自营' })
  specs.push({ label: '销量', value: (product.value.sales || 0) + '人付款' })
  specs.push({ label: '标签', value: product.value.tags || '京东自营' })
  specs.push({ label: '送达', value: deliveryEstimate.value || '极速达' })
  if (originLabel.value) specs.push({ label: '产地', value: originLabel.value })
  return specs
})

// Copy the formatted product summary: "🔥 商品名 | 现价¥X 原价¥Y | 京东自营".
async function copyProductInfo() {
  const name = product.value ? product.value.name : ''
  const cur = fmt(currentPrice())
  const orig = product.value ? fmt(product.value.original_price) : '0.00'
  const tag = product.value && product.value.tags ? product.value.tags : '京东自营'
  const text = `🔥 ${name} | 现价¥${cur} 原价¥${orig} | ${tag}`
  try {
    await navigator.clipboard.writeText(text)
    showSuccessToast('已复制商品信息')
  } catch (e) {
    // Fallback for non-secure contexts / older browsers.
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      showSuccessToast('已复制商品信息')
    } catch (_) {
      showToast('复制失败，请手动复制')
    }
    document.body.removeChild(ta)
  }
}
async function toggleRestock() {
  if (!checkLogin()) return
  try {
    if (restockSubscribed.value) {
      await unsubscribeRestock(product.value.id)
      restockSubscribed.value = false
      showSuccessToast('已取消到货通知')
    } else {
      await subscribeRestock(product.value.id)
      restockSubscribed.value = true
      showSuccessToast('到货后将通知您')
    }
  } catch (e) {
    showToast('操作失败')
  }
}
// Open the price-alert sheet. The target price is prefilled with 90% of the
// current price (the same default the backend uses) unless already subscribed.
function openPriceAlert() {
  if (!checkLogin()) return
  alertTargetPrice.value = (currentPrice() * 0.9).toFixed(2)
  showPriceAlert.value = true
}
async function submitPriceAlert() {
  const target = Number(alertTargetPrice.value)
  if (!target || target <= 0) {
    showToast('请输入有效的目标价')
    return
  }
  try {
    const res = await subscribePriceAlert(product.value.id, target)
    priceAlert.value = res.data
    showPriceAlert.value = false
    showSuccessToast('降价提醒已开启')
  } catch (e) {
    showToast(e.response?.data?.error || '订阅失败')
  }
}
async function submitQA() {
  if (!qaQuestion.value.trim()) { showToast('请输入问题'); return }
  if (!checkLogin()) return
  try {
    const qa = await askProductQA(product.value.id, qaQuestion.value)
    qaList.value.unshift(qa)
    showQA.value = false
    qaQuestion.value = ''
    showSuccessToast('提问成功')
  } catch (e) { showToast('请先登录') }
}
// Map price-history points to bar heights (0-100%) relative to the range.
function priceBars() {
  if (!priceHistory.value.length) return []
  const prices = priceHistory.value.map((p) => p.price)
  const min = Math.min(...prices)
  const max = Math.max(...prices)
  const range = max - min || 1
  return priceHistory.value.map((p) => ({
    price: p.price,
    height: 30 + Math.round(((p.price - min) / range) * 70), // 30-100%
    date: String(p.recorded_at).slice(5, 10), // MM-DD
  }))
}
function priceTrend() {
  if (!priceHistory.value.length || priceHistory.value.length < 2) return 'flat'
  const first = priceHistory.value[0].price
  const last = priceHistory.value[priceHistory.value.length - 1].price
  if (last < first) return 'down'
  if (last > first) return 'up'
  return 'flat'
}

// ---- Sticky detail tab navigation (商品详情Tab导航) ----
// Tabs: 商品 | 评价 | 问答 | 推荐. Clicking a tab smooth-scrolls to its
// section; an IntersectionObserver keeps the active tab in sync as the user
// scrolls. We intentionally guard everything so that if a section is missing
// the tab still renders but degrades gracefully.
const DETAIL_TABS = [
  { key: 'product', label: '商品', sectionId: 'sec-product' },
  { key: 'reviews', label: '评价', sectionId: 'sec-reviews' },
  { key: 'qa', label: '问答', sectionId: 'sec-qa' },
  { key: 'recommend', label: '推荐', sectionId: 'sec-recommend' },
]
const activeTab = ref('product')
let sectionObserver = null
// Track which sections are currently intersecting so scroll-driven highlight
// picks the topmost visible one.
const visibleSections = ref(new Set())

function scrollToTab(tab) {
  const el = document.getElementById(tab.sectionId)
  if (!el) return
  // Offset for the fixed nav-bar (~46px) + sticky tabs (~40px).
  const offset = 90
  const top = el.getBoundingClientRect().top + window.scrollY - offset
  // Temporarily ignore observer updates during programmatic scroll so the
  // clicked tab stays active until scrolling settles.
  pauseObserver = true
  activeTab.value = tab.key
  window.scrollTo({ top, behavior: 'smooth' })
  setTimeout(() => { pauseObserver = false }, 600)
}

let pauseObserver = false

function setupSectionObserver() {
  if (typeof IntersectionObserver === 'undefined') return
  sectionObserver = new IntersectionObserver(
    (entries) => {
      if (pauseObserver) return
      const next = new Set(visibleSections.value)
      let changed = false
      for (const e of entries) {
        const id = e.target.id
        if (e.isIntersecting) {
          if (!next.has(id)) { next.add(id); changed = true }
        } else {
          if (next.has(id)) { next.delete(id); changed = true }
        }
      }
      if (changed) visibleSections.value = next
      // Highlight the first tab (in DOM order) whose section is visible.
      for (const t of DETAIL_TABS) {
        if (visibleSections.value.has(t.sectionId)) {
          activeTab.value = t.key
          return
        }
      }
    },
    // Trigger when a section's top crosses ~120px down from the viewport top
    // (accounting for nav bar + sticky tabs) to the bottom edge.
    { rootMargin: '-120px 0px -70% 0px', threshold: 0 }
  )
  for (const t of DETAIL_TABS) {
    const el = document.getElementById(t.sectionId)
    if (el) sectionObserver.observe(el)
  }
}

// ---- Feature: 商品视频自动播放 — IntersectionObserver wiring ----
// Observe the product video element. When at least 50% of it is in view we
// auto-play it (muted, because browsers block autoplay-with-sound); when it
// scrolls away we pause it. No-op if IntersectionObserver is unavailable or
// the product has no video.
function setupVideoObserver() {
  if (typeof IntersectionObserver === 'undefined') return
  const v = videoRef.value
  if (!v) return
  videoObserver = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        videoVisible.value = e.isIntersecting
        const el = e.target
        if (e.isIntersecting) {
          // Autoplay muted so the browser allows it; ignore rejection.
          el.muted = videoMuted.value
          el.play().catch(() => {})
        } else {
          el.pause()
        }
      }
    },
    { threshold: 0.5 } // 50% visible triggers play
  )
  videoObserver.observe(v)
}

onUnmounted(() => {
  if (sectionObserver) {
    sectionObserver.disconnect()
    sectionObserver = null
  }
  if (videoObserver) {
    videoObserver.disconnect()
    videoObserver = null
  }
})
</script>

<template>
  <div v-if="loading" class="loading"><van-loading /></div>
  <div v-else-if="product" class="detail">
    <van-nav-bar title="商品详情" left-arrow @click-left="router.back()" fixed placeholder />
    <!-- Product intro video (商品视频介绍) -->
    <!-- Feature: 商品视频自动播放 — auto-plays (muted) when scrolled 50% into
         view, pauses on scroll-away, with a tap-to-unmute overlay and a
         play/pause toggle. The whole block is skipped when there's no video. -->
    <div v-if="product.video_url" class="product-video">
      <video
        ref="videoRef"
        :src="product.video_url"
        preload="metadata"
        class="pv-player"
        playsinline
        webkit-playsinline
        :muted="videoMuted"
        poster=""
        @play="onVideoPlay"
        @pause="onVideoPause"
        @ended="onVideoEnded"
      ></video>
      <!-- Tap-to-unmute overlay: shown while the video is muted. -->
      <div v-if="videoMuted" class="pv-unmute" @click.stop="unmuteVideo">
        🔇 点击取消静音
      </div>
      <!-- Manual play/pause toggle button -->
      <div class="pv-toggle" @click.stop="toggleVideoPlay">
        <van-icon :name="videoPlaying ? 'pause' : 'play'" />
      </div>
    </div>
    <!-- Image gallery carousel: uses the multi-image field, falls back to the main image -->
    <van-swipe class="gallery" :autoplay="3000" indicator-color="#e1251b" v-if="gallery.length > 1">
      <van-swipe-item v-for="(img, i) in gallery" :key="i">
        <van-image width="100%" height="375" :src="img" fit="cover" />
      </van-swipe-item>
    </van-swipe>
    <van-image v-else width="100%" height="375" :src="product.image" fit="cover" />
    <div class="price-block">
      <span class="big-price">¥{{ fmt(currentPrice()) }}</span>
      <span class="origin">¥{{ fmt(product.original_price) }}</span>
      <span v-if="product.vip_price > 0 && product.vip_price < currentPrice()" class="vip-price">
        <van-icon name="diamond-o" /> PLUS会员 ¥{{ fmt(product.vip_price) }}
      </span>
    </div>
    <!-- SKU spec selector -->
    <div v-if="skus.length" class="sku-block">
      <div class="sku-title">
        已选：<b>{{ selectedSKU ? selectedSKU.spec_text : '请选择规格' }}</b>
        <van-tag v-if="recommendedSKU && selectedSKU && selectedSKU.id === recommendedSKU.id" type="danger" round size="mini" style="margin-left:6px">AI推荐·性价比</van-tag>
      </div>
      <div class="sku-tags">
        <span
          v-for="s in skus"
          :key="s.id"
          class="sku-tag"
          :class="{ active: selectedSKU && selectedSKU.id === s.id }"
          @click="selectSKU(s)"
        >{{ s.spec_text }} <small>¥{{ fmt(s.price) }}</small></span>
      </div>
      <!-- Spec matrix table: shown only when SKUs span 2+ spec dimensions -->
      <table v-if="specMatrix.dimNames.length >= 2" class="spec-matrix">
        <thead>
          <tr>
            <th v-for="dim in specMatrix.dimNames" :key="dim">{{ dim }}</th>
            <th>价格</th>
            <th>库存</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in specMatrix.rows"
            :key="row.sku.id"
            :class="{ 'row-active': selectedSKU && selectedSKU.id === row.sku.id }"
            @click="selectSKU(row.sku)"
          >
            <td v-for="dim in specMatrix.dimNames" :key="dim">{{ row.dims[dim] || '-' }}</td>
            <td class="sm-price">¥{{ fmt(row.sku.price) }}</td>
            <td :class="{ 'sm-out': row.sku.stock <= 0 }">{{ row.sku.stock > 0 ? row.sku.stock + '件' : '缺货' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <!-- 库存紧张指示: visual stock pressure meter driven by the selected SKU's
         stock (or the product's stock when no SKU is selected) -->
    <div class="stock-meter" :class="{ blink: stockMeter.blink }">
      <span class="stock-flame" v-if="stockMeter.flame">🔥</span>
      <span class="stock-label" :style="{ color: stockMeter.color }">
        {{ stockMeter.label }}
      </span>
      <div v-if="stockMeter.bar" class="stock-bar-track">
        <div
          class="stock-bar-fill"
          :style="{ width: stockMeter.bar + '%', background: stockMeter.color }"
        ></div>
      </div>
    </div>
    <div class="title-block">
      <h2 class="p-title">{{ product.name }}</h2>
      <p class="p-sub">{{ product.subtitle }}</p>
    </div>
    <van-cell-group inset>
      <van-cell title="店铺" :value="product.shop" is-link @click="router.push('/shop/' + encodeURIComponent(product.shop))" />
      <van-cell title="销量" :value="product.sales + '人付款'" />
      <van-cell title="标签" :value="product.tags || '京东自营'" />
      <van-cell title="预计送达">
        <template #value>
          <span class="delivery-value">🚚 {{ deliveryEstimate }}</span>
        </template>
      </van-cell>
      <van-cell>
        <template #title>
          <span class="eco-title"><van-icon name="passed" /> 环保评分</span>
        </template>
        <template #value>
          <span class="eco-value" :style="{ color: ecoColor }">
            🌱 {{ ecoScore }} · {{ ecoLevel }}
          </span>
        </template>
      </van-cell>
      <van-cell :title="restockSubscribed ? '到货通知已开启' : '到货通知'" is-link @click="toggleRestock">
        <template #right-icon>
          <van-switch :model-value="restockSubscribed" size="20" @click.stop="toggleRestock" active-color="#e1251b" />
        </template>
      </van-cell>
      <van-cell title="📍 产地溯源" is-link @click="showOrigin = true">
        <template #value>
          <span class="origin-value">📍 {{ originLabel }}</span>
        </template>
      </van-cell>
    </van-cell-group>
    <!-- Sticky detail tab navigation (商品详情Tab导航) -->
    <div class="detail-tabs-sticky">
      <div class="detail-tabs">
        <span
          v-for="t in DETAIL_TABS"
          :key="t.key"
          class="detail-tab"
          :class="{ active: activeTab === t.key }"
          @click="scrollToTab(t)"
        >{{ t.label }}</span>
      </div>
    </div>

    <!-- Price history (比价历史) -->
    <div v-if="priceHistory.length" class="price-history">
      <div class="ph-head">
        <span>📈 比价历史</span>
        <span class="ph-head-right">
          <span v-if="priceStats" class="ph-stats">
            最低 <b class="green">¥{{ fmt(priceStats.lowest) }}</b>
            <span v-if="priceTrend() === 'down'" class="trend down">↓降价</span>
            <span v-else-if="priceTrend() === 'up'" class="trend up">↑涨价</span>
            <span v-else class="trend flat">→平稳</span>
          </span>
          <van-tag v-if="priceAlert" type="danger" round size="medium" @click="openPriceAlert">已订阅 ¥{{ fmt(priceAlert.target_price) }}</van-tag>
          <van-button v-else size="mini" type="danger" plain round icon="bell" @click="openPriceAlert">降价提醒</van-button>
        </span>
      </div>
      <div class="ph-chart">
        <div v-for="(b, i) in priceBars()" :key="i" class="ph-bar-col">
          <div class="ph-bar" :style="{ height: b.height + '%' }"></div>
          <span class="ph-date">{{ b.date }}</span>
        </div>
      </div>
    </div>
    <!-- Purchase Note Templates (购买须知) -->
    <div class="purchase-notes">
      <div class="pn-head" @click="purchaseNotesCollapsed = !purchaseNotesCollapsed">
        <span>📋 购买须知</span>
        <span class="pn-toggle">{{ purchaseNotesCollapsed ? '展开' : '收起' }} <van-icon :name="purchaseNotesCollapsed ? 'arrow-down' : 'arrow-up'" /></span>
      </div>
      <div v-show="!purchaseNotesCollapsed" class="pn-body">
        <div v-for="(note, i) in purchaseNotes" :key="i" class="pn-item">
          <span class="pn-check">✓</span>
          <span class="pn-text">{{ note }}</span>
        </div>
      </div>
    </div>
    <!-- Brand Story (品牌故事) -->
    <div id="sec-product" class="brand-story">
      <div class="bs-head" @click="brandStoryCollapsed = !brandStoryCollapsed">
        <span>品牌故事</span>
        <span class="bs-toggle">{{ brandStoryCollapsed ? '展开' : '收起' }} <van-icon :name="brandStoryCollapsed ? 'arrow-down' : 'arrow-up'" /></span>
      </div>
      <div v-show="!brandStoryCollapsed" class="bs-body">
        <p class="bs-text">{{ brandStory }}</p>
        <div class="bs-tags">
          <span class="bs-tag-label">品牌理念</span>
          <van-tag v-for="t in brandTags" :key="t" plain round color="#e1251b">{{ t }}</van-tag>
        </div>
      </div>
    </div>
    <div v-if="product.description" class="desc">
      <h3>商品详情</h3>
      <p>{{ product.description }}</p>
    </div>
    <!-- Product Q&A (商品问答) -->
    <div id="sec-qa" class="qa-section">
      <div class="rev-head">
        <span>商品问答 ({{ qaList.length }})</span>
        <van-button size="mini" type="danger" plain @click="showQA = true">提问</van-button>
      </div>
      <div v-if="!qaList.length" class="qa-empty">暂无问答</div>
      <div v-for="qa in qaList.slice(0, 5)" :key="qa.id" class="qa-item">
        <div class="qa-q"><span class="qa-tag-q">问</span> {{ qa.question }}</div>
        <div v-if="qa.answer" class="qa-a"><span class="qa-tag-a">答</span> {{ qa.answer }} <small class="qa-answerer">{{ qa.answerer }}</small></div>
      </div>
    </div>

    <!-- Q&A popup -->
    <van-popup v-model:show="showQA" position="bottom" round closeable>
      <div class="qa-form">
        <h3>我要提问</h3>
        <van-field v-model="qaQuestion" type="textarea" placeholder="说说你想了解的问题" rows="3" />
        <van-button type="danger" block @click="submitQA" style="margin-top:12px">提交问题</van-button>
      </div>
    </van-popup>

    <!-- Price drop alert popup (降价提醒) -->
    <van-popup v-model:show="showPriceAlert" position="bottom" round closeable>
      <div class="alert-form">
        <h3>降价提醒</h3>
        <p class="alert-hint">设置目标价格，当商品价格低于该价格时通知您</p>
        <van-field
          v-model="alertTargetPrice"
          type="number"
          label="目标价格"
          placeholder="请输入目标价格"
        >
          <template #left-icon><span style="color:#e1251b;font-weight:bold">¥</span></template>
        </van-field>
        <p class="alert-current">当前价格：¥{{ fmt(currentPrice()) }}</p>
        <van-button type="danger" block round @click="submitPriceAlert" style="margin-top:12px">
          {{ priceAlert ? '更新提醒' : '开启提醒' }}
        </van-button>
      </div>
    </van-popup>

    <!-- Product origin route popup (产地溯源) -->
    <van-popup v-model:show="showOrigin" round closeable position="bottom" :style="{ width: '88%' }">
      <div class="origin-box">
        <h3 class="origin-title">📍 产地溯源</h3>
        <div class="origin-route">
          <div class="or-point or-start">
            <div class="or-icon">📍</div>
            <div class="or-name">{{ originLabel }}</div>
            <div class="or-sub">产地直发</div>
          </div>
          <div class="or-track">
            <span class="or-truck">🚚</span>
            <div class="or-line"></div>
            <div class="or-dots"><span></span><span></span><span></span></div>
          </div>
          <div class="or-point or-end">
            <div class="or-icon">🏠</div>
            <div class="or-name">您</div>
            <div class="or-sub">送货上门</div>
          </div>
        </div>
        <p class="origin-desc">本商品由 <b>{{ originLabel }}</b> 产地直发，全程冷链运输，新鲜直达您手中。</p>
        <van-button block type="danger" round @click="showOrigin = false" style="margin-top: 12px">知道了</van-button>
      </div>
    </van-popup>

    <div id="sec-reviews" class="reviews">
      <div class="rev-head">
        <span>商品评价 ({{ reviews.length }})</span>
        <van-button size="mini" type="danger" plain @click="showReview = true">写评价</van-button>
      </div>
      <!-- 评价概览统计: average score + good rate on the left, star distribution bars on the right -->
      <div v-if="reviewStats.total" class="rev-summary">
        <div class="rs-left">
          <div class="rs-avg">{{ reviewStats.avg }}</div>
          <van-rate :model-value="Number(reviewStats.avg)" allow-half readonly size="13" color="#e1251b" void-color="#eee" />
          <div class="rs-goodrate">好评率 {{ reviewStats.goodRate }}%</div>
        </div>
        <div class="rs-right">
          <div v-for="b in reviewDistBars" :key="b.star" class="rs-bar-row">
            <span class="rs-bar-star">{{ b.star }}★</span>
            <div class="rs-bar-track"><div class="rs-bar-fill" :style="{ width: b.pct + '%' }"></div></div>
            <span class="rs-bar-count">{{ b.count }}</span>
          </div>
        </div>
      </div>
      <!-- 评价筛选标签: horizontal pill chips to filter reviews by rating/photos -->
      <div v-if="reviewStats.total" class="rev-filters">
        <span
          v-for="f in reviewFilters"
          :key="f.key"
          class="rev-filter-chip"
          :class="{ active: reviewFilter === f.key }"
          @click="reviewFilter = f.key"
        >{{ f.label }}({{ f.count }})</span>
      </div>
      <div v-for="r in filteredReviews" :key="r.id" class="rev-item">
        <div class="rev-user">
          <span>{{ r.username }}</span>
          <van-rate v-model="r.rating" readonly size="12" />
          <span class="rev-reply-btn" @click="toggleReply(r)">回复</span>
        </div>
        <div class="rev-content">{{ r.content }}</div>
        <div v-if="r.images" class="rev-photos">
          <van-image v-for="(img, i) in reviewMedia(r.images).photos" :key="'p' + i" width="72" height="72" radius="6" :src="img" fit="cover" />
          <video
            v-for="(vid, i) in reviewMedia(r.images).videos"
            :key="'v' + i"
            :src="vid"
            controls
            preload="metadata"
            class="rev-video"
          ></video>
        </div>
        <div class="rev-actions">
          <span class="rev-useful-btn" @click="doUseful(r)"><van-icon name="good-job-o" /> 有用 ({{ r.useful || 0 }})</span>
        </div>
        <div v-if="r.reply" class="rev-reply">
          <span class="rev-reply-name">{{ r.reply.username }}：</span>{{ r.reply.content }}
        </div>
        <div v-if="replyingTo === r.id" class="rev-reply-box">
          <van-field v-model="replyText" placeholder="写下你的回复..." />
          <van-button size="small" type="danger" @click="submitReply(r)">发送</van-button>
        </div>
      </div>
      <van-empty v-if="!reviews.length" description="暂无评价" />
      <van-empty v-else-if="!filteredReviews.length" description="该筛选下暂无评价" />
    </div>

    <!-- Related products (看了又看) -->
    <div v-if="relatedProducts.length" id="sec-recommend" class="related-section">
      <div class="rs-head">看了又看</div>
      <div class="rs-scroll">
        <div v-for="rp in relatedProducts" :key="rp.id" class="rs-card" @click="goProduct(rp.id)">
          <van-image width="100" height="100" radius="6" :src="rp.image" fit="cover" />
          <div class="rs-name van-multi-ellipsis--l2">{{ rp.name }}</div>
          <div class="rs-price">¥{{ fmt(rp.price) }}</div>
        </div>
      </div>
    </div>

    <!-- Bottom action bar -->
    <van-action-bar>
      <van-action-bar-icon icon="chat-o" text="客服" @click="showToast('客服功能为演示')" />
      <van-action-bar-icon icon="share-o" text="分享" @click="showPoster = true" />
      <van-action-bar-icon :icon="favorited ? 'star' : 'star-o'" :text="favorited ? '已收藏' : '收藏'" :color="favorited ? '#e1251b' : '#323233'" @click="doFavorite" />
      <van-action-bar-icon icon="cart-o" text="购物车" @click="router.push('/cart')" />
      <van-action-bar-button color="#ff976a" type="warning" text="加入购物车" @click="doAddCart" />
      <van-action-bar-button color="#e1251b" type="danger" text="立即购买" @click="buyNow" />
    </van-action-bar>

    <!-- Share poster popup -->
    <van-popup v-model:show="showPoster" round closeable position="bottom" :style="{ width: '85%' }">
      <div class="poster">
        <div class="poster-head">分享给好友</div>
        <div class="poster-card">
          <van-image width="100%" height="200" radius="8" :src="product.image" fit="cover" />
          <div class="pc-name van-multi-ellipsis--l2">{{ product.name }}</div>
          <div class="pc-price">¥{{ fmt(currentPrice()) }}</div>
          <div class="pc-qr">
            <div class="qr-box">
              <div class="qr-grid">
                <div v-for="n in 64" :key="n" class="qr-cell" :class="{ on: qrPattern(n) }"></div>
              </div>
            </div>
            <div class="qr-text">扫码查看商品</div>
          </div>
          <div class="pc-brand">京东 JD.COM</div>
        </div>
        <van-button block plain type="danger" round icon="description" style="margin-top: 12px" @click="showLongImage = true">📋生成长图</van-button>
        <van-button block type="danger" round style="margin-top: 8px" @click="copyShareLink">复制分享链接</van-button>

        <!-- Share long image (分享长图) -->
        <div v-if="showLongImage" class="long-img-wrap">
          <div class="li-head">
            <span>长图预览</span>
            <span class="li-close" @click="showLongImage = false">收起</span>
          </div>
          <div class="long-img-scroll">
            <div class="long-card">
              <div class="lc-top-bar">
                <span class="lc-logo">JD</span>
                <span class="lc-top-name">京东</span>
                <span class="lc-tag">自营</span>
              </div>
              <van-image width="100%" height="220" radius="8" :src="product.image" fit="cover" />
              <div class="lc-name van-multi-ellipsis--l2">{{ product.name }}</div>
              <p v-if="product.subtitle" class="lc-sub">{{ product.subtitle }}</p>
              <div class="lc-price-row">
                <span class="lc-price">¥{{ fmt(currentPrice()) }}</span>
                <span class="lc-origin">¥{{ fmt(product.original_price) }}</span>
              </div>
              <div v-if="product.vip_price > 0 && product.vip_price < currentPrice()" class="lc-vip">
                <span class="lc-vip-badge">PLUS</span>
                <span class="lc-vip-price">会员价 ¥{{ fmt(product.vip_price) }}</span>
              </div>
              <div class="lc-specs">
                <div v-for="(sp, i) in longCardSpecs" :key="i" class="lc-spec-row">
                  <span class="lc-spec-label">{{ sp.label }}</span>
                  <span class="lc-spec-value">{{ sp.value }}</span>
                </div>
              </div>
              <div class="lc-footer">
                <div class="lc-qr">
                  <div class="qr-grid">
                    <div v-for="n in 64" :key="n" class="qr-cell" :class="{ on: qrPattern(n) }"></div>
                  </div>
                  <div class="lc-qr-text">长按识别</div>
                </div>
                <div class="lc-watermark">JD 京东</div>
              </div>
            </div>
          </div>
          <van-button block type="danger" round icon="records" style="margin-top: 10px" @click="copyProductInfo">保存图片</van-button>
        </div>
      </div>
    </van-popup>

    <!-- Review popup -->
    <van-popup v-model:show="showReview" position="bottom" round closeable>
      <div class="rev-form">
        <h3>写评价</h3>
        <van-rate v-model="reviewRating" />
        <van-field v-model="reviewContent" type="textarea" placeholder="说说你的使用感受" rows="3" />
        <div class="rev-upload">
          <div class="rev-upload-row">
            <van-uploader :after-read="onUploadReviewImage" accept="image/*" multiple :preview-image="false">
              <van-button icon="photo-o" size="small" plain round>添加晒图</van-button>
            </van-uploader>
            <van-button icon="video-o" size="small" plain round @click="showVideoInput = !showVideoInput">{{ showVideoInput ? '收起视频' : '添加视频' }}</van-button>
          </div>
          <div v-if="showVideoInput" class="rev-video-input">
            <van-field v-model="reviewVideo" placeholder="粘贴视频链接 https://..." clearable />
            <small class="rev-video-hint">演示功能：直接粘贴视频地址即可</small>
          </div>
          <div v-if="reviewImages.length" class="rev-imgs">
            <div v-for="(img, i) in reviewImages" :key="i" class="rev-img-wrap">
              <van-image width="60" height="60" radius="6" :src="img" fit="cover" />
              <van-icon name="cross" class="rev-img-del" @click="removeReviewImage(i)" />
            </div>
          </div>
        </div>
        <van-button type="danger" block @click="submitReview">提交评价</van-button>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.detail { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.product-video { background: #000; width: 100%; position: relative; }
.pv-player { width: 100%; max-height: 280px; object-fit: contain; display: block; }
/* Feature: 商品视频自动播放 — unmute overlay + play/pause toggle */
.pv-unmute {
  position: absolute;
  top: 10px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  font-size: 12px;
  padding: 4px 12px;
  border-radius: 14px;
  cursor: pointer;
  white-space: nowrap;
  pointer-events: auto;
  z-index: 2;
  transition: background 0.2s ease;
}
.pv-unmute:active { background: rgba(225, 37, 27, 0.8); }
.pv-toggle {
  position: absolute;
  bottom: 10px;
  right: 10px;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  cursor: pointer;
  z-index: 2;
}
.pv-toggle:active { background: rgba(225, 37, 27, 0.8); }
.price-block { padding: 12px 16px; background: #fff; }
.vip-price { margin-left: 12px; color: #333; font-size: 13px; background: linear-gradient(90deg, #ffd700, #ffaa00); padding: 2px 10px; border-radius: 12px; }
.qa-section { background: #fff; margin-top: 8px; padding: 12px 16px; }
.qa-empty { color: #999; font-size: 13px; padding: 12px 0; }
.qa-item { padding: 10px 0; border-top: 1px solid #f5f5f5; }
.qa-q { font-size: 14px; line-height: 20px; }
.qa-tag-q { background: #e1251b; color: #fff; font-size: 11px; padding: 1px 6px; border-radius: 4px; margin-right: 6px; }
.qa-a { font-size: 13px; color: #666; margin-top: 6px; line-height: 18px; }
.qa-tag-a { background: #07c160; color: #fff; font-size: 11px; padding: 1px 6px; border-radius: 4px; margin-right: 6px; }
.qa-answerer { color: #999; margin-left: 6px; }
.qa-form { padding: 20px; }
.qa-form h3 { text-align: center; margin-bottom: 16px; }
.qa-form .van-field { border: 1px solid #eee; }
.big-price { color: #e1251b; font-size: 28px; font-weight: bold; }
.origin { color: #999; text-decoration: line-through; margin-left: 10px; font-size: 14px; }
.sku-block { padding: 12px 16px; background: #fff; border-top: 1px solid #f5f5f5; }
.sku-title { font-size: 13px; color: #666; margin-bottom: 8px; }
.sku-title b { color: #333; }
.sku-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.sku-tag { padding: 6px 12px; background: #f7f7f7; border: 1px solid #eee; border-radius: 16px; font-size: 13px; color: #333; }
.sku-tag.active { background: #fff5f5; border-color: #e1251b; color: #e1251b; }
.sku-tag small { color: #e1251b; margin-left: 4px; }
/* Spec matrix table (商品规格矩阵表) */
.spec-matrix { width: 100%; border-collapse: collapse; margin-top: 12px; font-size: 12px; }
.spec-matrix th, .spec-matrix td { border: 1px solid #eee; padding: 8px 6px; text-align: center; white-space: nowrap; }
.spec-matrix th { background: #fafafa; color: #666; font-weight: 500; }
.spec-matrix tbody tr { cursor: pointer; }
.spec-matrix tbody tr:active { background: #fafafa; }
.spec-matrix .row-active { background: #fff5f5; }
.spec-matrix .row-active td { color: #e1251b; font-weight: 600; }
.spec-matrix .sm-price { color: #e1251b; }
.spec-matrix .sm-out { color: #999; }
.title-block { padding: 0 16px 12px; background: #fff; }
.p-title { font-size: 17px; line-height: 24px; }
.p-sub { color: #999; font-size: 13px; margin-top: 4px; }
.delivery-value { color: #e1251b; font-weight: 500; }
/* Eco score badge (环保评分) */
.eco-title { display: inline-flex; align-items: center; gap: 4px; }
.eco-value { font-weight: bold; }
/* Product origin cell (产地溯源) */
.origin-value { color: #e1251b; font-weight: 500; }
.origin-box { padding: 20px; }
.origin-title { text-align: center; font-size: 16px; margin-bottom: 18px; }
.origin-route { display: flex; align-items: center; justify-content: space-between; gap: 4px; padding: 8px 4px 12px; }
.or-point { display: flex; flex-direction: column; align-items: center; gap: 4px; min-width: 64px; }
.or-icon { font-size: 30px; line-height: 1; }
.or-name { font-size: 14px; font-weight: bold; color: #333; }
.or-sub { font-size: 11px; color: #999; }
.or-track { position: relative; flex: 1; display: flex; align-items: center; justify-content: center; height: 30px; }
.or-truck { position: absolute; font-size: 18px; z-index: 1; }
.or-line { position: absolute; left: 0; right: 0; top: 50%; height: 3px; background: repeating-linear-gradient(90deg, #e1251b 0, #e1251b 6px, transparent 6px, transparent 12px); transform: translateY(-50%); }
.or-dots { position: absolute; right: -2px; display: flex; gap: 2px; }
.or-dots span { width: 4px; height: 4px; border-radius: 50%; background: #e1251b; opacity: 0.6; }
.origin-desc { font-size: 13px; color: #666; line-height: 20px; text-align: center; margin-top: 10px; }
.origin-desc b { color: #e1251b; }
/* Purchase Note Templates (购买须知) */
.purchase-notes { background: #fff; margin-top: 8px; padding: 12px 16px; }
.pn-head { display: flex; justify-content: space-between; align-items: center; font-size: 15px; font-weight: bold; }
.pn-toggle { font-size: 13px; font-weight: normal; color: #e1251b; display: inline-flex; align-items: center; gap: 2px; }
.pn-body { padding-top: 10px; }
.pn-item { display: flex; align-items: flex-start; gap: 8px; padding: 6px 0; }
.pn-check { color: #07c160; font-weight: bold; flex-shrink: 0; line-height: 20px; }
.pn-text { font-size: 13px; color: #666; line-height: 20px; }
.desc, .reviews { background: #fff; margin-top: 8px; padding: 12px 16px; }
/* Brand Story (品牌故事) */
.brand-story { background: #fff; margin-top: 8px; padding: 12px 16px; }
/* Sticky detail tab navigation (商品详情Tab导航) */
.detail-tabs-sticky {
  position: sticky;
  top: 46px; /* sit just below the fixed nav-bar */
  z-index: 10;
  background: #fff;
  margin-top: 8px;
}
.detail-tabs {
  display: flex;
  border-bottom: 1px solid #f5f5f5;
}
.detail-tab {
  flex: 1;
  text-align: center;
  padding: 12px 0;
  font-size: 14px;
  color: #666;
  cursor: pointer;
  position: relative;
  transition: color 0.2s ease;
}
.detail-tab.active {
  color: #e1251b;
  font-weight: 600;
}
.detail-tab.active::after {
  content: '';
  position: absolute;
  left: 50%;
  bottom: 0;
  transform: translateX(-50%);
  width: 24px;
  height: 3px;
  border-radius: 2px;
  background: #e1251b;
}
.bs-head { display: flex; justify-content: space-between; align-items: center; font-size: 15px; font-weight: bold; }
.bs-toggle { font-size: 13px; font-weight: normal; color: #e1251b; display: inline-flex; align-items: center; gap: 2px; }
.bs-body { padding-top: 10px; }
.bs-text { font-size: 13px; color: #666; line-height: 20px; }
.bs-tags { display: flex; align-items: center; gap: 8px; margin-top: 10px; flex-wrap: wrap; }
.bs-tag-label { font-size: 12px; color: #999; }
.price-history { background: #fff; margin-top: 8px; padding: 12px 16px; }
.ph-head { display: flex; justify-content: space-between; align-items: center; font-size: 14px; font-weight: bold; margin-bottom: 10px; }
.ph-stats { font-size: 12px; color: #666; font-weight: normal; }
.ph-stats b.green { color: #07c160; }
.trend { margin-left: 6px; font-size: 11px; }
.trend.down { color: #07c160; }
.trend.up { color: #e1251b; }
.trend.flat { color: #999; }
.ph-chart { display: flex; align-items: flex-end; gap: 6px; height: 80px; }
.ph-head-right { display: flex; align-items: center; gap: 8px; }
.alert-form { padding: 20px; }
.alert-form h3 { text-align: center; margin-bottom: 8px; }
.alert-hint { color: #999; font-size: 12px; text-align: center; margin-bottom: 16px; }
.alert-form .van-field { border: 1px solid #eee; border-radius: 8px; }
.alert-current { color: #666; font-size: 13px; margin-top: 10px; text-align: center; }
.ph-bar-col { flex: 1; display: flex; flex-direction: column; align-items: center; height: 100%; justify-content: flex-end; }
.ph-bar { width: 60%; background: linear-gradient(180deg, #ff9800, #e1251b); border-radius: 3px 3px 0 0; min-height: 8px; }
.ph-date { font-size: 9px; color: #999; margin-top: 4px; }
.desc h3, .rev-head { font-size: 15px; margin-bottom: 8px; display: flex; justify-content: space-between; align-items: center; }
/* 评价概览统计 card: average/good-rate on left, star distribution bars on right */
.rev-summary { display: flex; align-items: center; gap: 16px; padding: 12px 0; border-top: 1px solid #f5f5f5; }
.rs-left { display: flex; flex-direction: column; align-items: center; gap: 4px; min-width: 96px; }
.rs-avg { font-size: 32px; font-weight: bold; color: #e1251b; line-height: 1; }
.rs-goodrate { font-size: 12px; color: #666; margin-top: 2px; }
.rs-right { flex: 1; display: flex; flex-direction: column; gap: 4px; }
.rs-bar-row { display: flex; align-items: center; gap: 6px; font-size: 11px; color: #999; }
.rs-bar-star { width: 24px; text-align: right; }
.rs-bar-track { flex: 1; height: 8px; background: #f0f0f0; border-radius: 4px; overflow: hidden; }
.rs-bar-fill { height: 100%; background: linear-gradient(90deg, #ff9800, #e1251b); border-radius: 4px; }
.rs-bar-count { width: 20px; text-align: right; }
/* Review filter tabs (评价筛选标签) — horizontal scrollable pills */
.rev-filters { display: flex; gap: 8px; overflow-x: auto; padding: 8px 0 4px; border-top: 1px solid #f5f5f5; -webkit-overflow-scrolling: touch; }
.rev-filters::-webkit-scrollbar { display: none; }
.rev-filter-chip { flex-shrink: 0; padding: 5px 14px; background: #f7f7f7; border: 1px solid #eee; border-radius: 14px; font-size: 13px; color: #333; cursor: pointer; white-space: nowrap; }
.rev-filter-chip.active { background: #e1251b; border-color: #e1251b; color: #fff; font-weight: 500; }
/* Stock pressure meter (库存紧张指示) */
.stock-meter { display: flex; align-items: center; gap: 8px; background: #fff; padding: 10px 16px; border-top: 1px solid #f5f5f5; }
.stock-flame { font-size: 14px; line-height: 1; }
.stock-label { font-size: 13px; font-weight: 500; }
.stock-bar-track { flex: 1; height: 8px; background: #f0f0f0; border-radius: 4px; overflow: hidden; max-width: 200px; }
.stock-bar-fill { height: 100%; border-radius: 4px; transition: width .3s; }
.stock-meter.blink .stock-label { animation: stock-blink 1s steps(2, start) infinite; }
.stock-meter.blink .stock-bar-fill { animation: stock-blink 1s steps(2, start) infinite; }
@keyframes stock-blink { 50% { opacity: .35; } }
.rev-item { padding: 10px 0; border-top: 1px solid #f5f5f5; }
.rev-user { display: flex; gap: 8px; align-items: center; font-size: 13px; color: #666; }
.rev-content { font-size: 13px; margin-top: 4px; line-height: 18px; }
.rev-photos { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 6px; }
.rev-video { width: 200px; height: auto; max-height: 200px; border-radius: 6px; background: #000; object-fit: contain; }
.rev-actions { margin-top: 6px; }
.rev-useful-btn { color: #999; font-size: 12px; cursor: pointer; }
.rev-useful-btn:active { color: #e1251b; }
.rev-reply-btn { margin-left: auto; color: #e1251b; font-size: 12px; }
.rev-reply { background: #f7f7f7; border-radius: 6px; padding: 6px 10px; margin-top: 6px; font-size: 12px; color: #666; line-height: 18px; }
.rev-reply-name { color: #e1251b; }
.rev-reply-box { display: flex; gap: 8px; align-items: center; margin-top: 8px; }
.rev-reply-box .van-field { flex: 1; border: 1px solid #eee; border-radius: 6px; }
.rev-form { padding: 20px; }
.rev-form h3 { text-align: center; margin-bottom: 16px; }
.rev-form .van-field { margin: 12px 0; border: 1px solid #eee; }
.rev-upload { margin: 8px 0; }
.rev-upload-row { display: flex; gap: 8px; flex-wrap: wrap; }
.rev-video-input { margin-top: 8px; }
.rev-video-input .van-field { border: 1px solid #eee; border-radius: 6px; }
.rev-video-hint { color: #999; font-size: 11px; display: block; margin-top: 4px; }
.rev-imgs { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.rev-img-wrap { position: relative; }
.rev-img-del { position: absolute; top: -6px; right: -6px; background: #e1251b; color: #fff; border-radius: 50%; padding: 2px; font-size: 12px; }
.poster { padding: 20px; }
.poster-head { text-align: center; font-size: 16px; font-weight: bold; margin-bottom: 16px; }
.poster-card { background: #fff; border: 1px solid #eee; border-radius: 12px; padding: 16px; text-align: center; }
.pc-name { font-size: 15px; line-height: 22px; margin: 12px 0 6px; text-align: left; }
.pc-price { color: #e1251b; font-size: 24px; font-weight: bold; text-align: left; }
.pc-qr { display: flex; flex-direction: column; align-items: center; margin-top: 16px; }
.qr-box { padding: 8px; border: 1px solid #eee; border-radius: 8px; }
.qr-grid { display: grid; grid-template-columns: repeat(8, 1fr); gap: 1px; width: 120px; height: 120px; }
.qr-cell { background: #fff; }
.qr-cell.on { background: #333; }
.qr-text { font-size: 11px; color: #999; margin-top: 6px; }
.pc-brand { color: #e1251b; font-size: 13px; font-weight: bold; margin-top: 12px; }

/* Share long image (分享长图) */
.long-img-wrap { margin-top: 16px; }
.li-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  font-weight: bold;
  margin-bottom: 10px;
}
.li-close { font-size: 12px; font-weight: normal; color: #e1251b; cursor: pointer; }
.long-img-scroll {
  max-height: 60vh;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  border: 1px solid #eee;
  border-radius: 12px;
  padding: 4px;
}
.long-card {
  background: linear-gradient(180deg, #fff5f5 0%, #fff 30%);
  border-radius: 10px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.lc-top-bar {
  display: flex;
  align-items: center;
  gap: 6px;
}
.lc-logo {
  color: #fff;
  background: #e1251b;
  font-weight: bold;
  font-size: 13px;
  padding: 1px 6px;
  border-radius: 4px;
}
.lc-top-name { font-size: 15px; font-weight: bold; color: #333; }
.lc-tag {
  font-size: 10px;
  color: #e1251b;
  border: 1px solid #e1251b;
  border-radius: 3px;
  padding: 0 4px;
}
.lc-name { font-size: 16px; line-height: 22px; color: #333; font-weight: 600; }
.lc-sub { font-size: 12px; color: #999; line-height: 18px; margin-top: -4px; }
.lc-price-row { display: flex; align-items: baseline; gap: 10px; }
.lc-price { color: #e1251b; font-size: 28px; font-weight: bold; }
.lc-origin { color: #999; text-decoration: line-through; font-size: 14px; }
.lc-vip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: linear-gradient(90deg, #ffd700, #ffaa00);
  align-self: flex-start;
  padding: 3px 10px;
  border-radius: 14px;
  font-size: 12px;
  color: #6b4a00;
}
.lc-vip-badge { font-weight: bold; color: #5a3d00; }
.lc-specs {
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: #fafafa;
  border-radius: 8px;
  padding: 10px 12px;
}
.lc-spec-row { display: flex; font-size: 13px; line-height: 20px; }
.lc-spec-label { width: 48px; color: #999; flex-shrink: 0; }
.lc-spec-value { color: #333; flex: 1; }
.lc-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
  margin-top: 4px;
}
.lc-qr { display: flex; flex-direction: column; align-items: center; }
.lc-qr-text { font-size: 10px; color: #999; margin-top: 4px; }
.lc-watermark {
  font-size: 18px;
  font-weight: bold;
  letter-spacing: 1px;
  color: #e1251b;
  opacity: 0.85;
}

.related-section { background: #fff; margin-top: 8px; padding: 12px 16px; }
.rs-head { font-size: 15px; font-weight: bold; margin-bottom: 10px; }
.rs-scroll { display: flex; gap: 10px; overflow-x: auto; }
.rs-card { flex-shrink: 0; width: 110px; }
.rs-name { font-size: 12px; color: #333; line-height: 16px; margin-top: 4px; height: 32px; }
.rs-price { color: #e1251b; font-size: 14px; font-weight: bold; }
</style>
