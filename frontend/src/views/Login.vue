<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { login, register, getCoupons, claimCoupon } from '../api'

const route = useRoute()
const router = useRouter()
const mode = ref('login')
const username = ref('admin')
const password = ref('admin123')
const nickname = ref('')

// ---- New user welcome coupon pack (新人专享礼包) ----
// One-time full-screen popup shown after the first successful login/register,
// auto-claiming a handful of coupons. Tracked in localStorage so it only ever
// shows once per browser.
const WELCOME_KEY = 'jd_welcome_claimed'
const showWelcome = ref(false)
const claiming = ref(false)
const confettiPieces = ref([])
let confettiTimer = null

// Three demo coupons displayed in the pack. (满50减5 / 满100减15 / 满200减30)
const demoCoupons = [
  { threshold: 50, value: 5 },
  { threshold: 100, value: 15 },
  { threshold: 200, value: 30 },
]

function spawnConfetti() {
  const emojis = ['🎉', '🎊', '✨', '🎈', '🎁', '💫', '🧧']
  confettiPieces.value = Array.from({ length: 24 }, (_, i) => ({
    id: i,
    emoji: emojis[Math.floor(Math.random() * emojis.length)],
    left: Math.random() * 100, // vw
    delay: Math.random() * 0.8, // s
    duration: 2.2 + Math.random() * 1.6, // s
    size: 16 + Math.random() * 18, // px
  }))
}

function closeWelcome() {
  showWelcome.value = false
  if (confettiTimer) {
    clearInterval(confettiTimer)
    confettiTimer = null
  }
}

// "立即领取": claim the first 3 available coupons from the backend, then
// mark the welcome pack as claimed so it never shows again. If the API
// fails (not logged in / already claimed), we skip silently per spec.
async function claimWelcome() {
  if (claiming.value) return
  claiming.value = true
  try {
    const list = await getCoupons()
    const available = (list || []).slice(0, 3)
    await Promise.allSettled((available || []).map((c) => claimCoupon(c.id)))
    localStorage.setItem(WELCOME_KEY, '1')
    showSuccessToast('礼包领取成功')
  } catch (_) {
    // Silently ignore claim failures (e.g. user not logged in / already claimed).
    localStorage.setItem(WELCOME_KEY, '1')
  } finally {
    claiming.value = false
    closeWelcome()
    const redirect = route.query.redirect || '/mine'
    router.replace(redirect)
  }
}

// Open the welcome pack once, for a freshly-authenticated user who hasn't
// seen it before. Returns true if the popup was shown (so the caller can
// defer the redirect until the user dismisses it).
function maybeShowWelcome() {
  if (localStorage.getItem(WELCOME_KEY)) return false
  showWelcome.value = true
  spawnConfetti()
  // Keep refreshing the confetti emoji set so the animation stays lively.
  confettiTimer = setInterval(spawnConfetti, 2600)
  return true
}

async function submit() {
  if (!username.value || !password.value) {
    showToast('请输入用户名和密码')
    return
  }
  try {
    let res
    if (mode.value === 'login') {
      res = await login(username.value, password.value)
    } else {
      res = await register({ username: username.value, password: password.value, nickname: nickname.value })
    }
    localStorage.setItem('jd_token', res.token)
    localStorage.setItem('jd_user', JSON.stringify(res.user))
    showSuccessToast(mode.value === 'login' ? '登录成功' : '注册成功')
    // Show the new-user welcome pack on first login; defer the redirect.
    if (maybeShowWelcome()) return
    const redirect = route.query.redirect || '/mine'
    router.replace(redirect)
  } catch (e) {
    showToast(e.response?.data?.error || '操作失败')
  }
}

// ---- Social login & forgot password (登录页美化) ----
// Visual-only social login buttons; show a toast on tap for the demo.
const socials = [
  { key: 'wechat', label: '微信', emoji: '💚' },
  { key: 'alipay', label: '支付宝', emoji: '💙' },
  { key: 'qq', label: 'QQ', emoji: '🐧' },
]
function socialLogin(s) {
  showToast(s.label + '登录（演示）')
}
function forgotPassword() {
  showToast('找回密码（演示）')
}

onUnmounted(() => {
  if (confettiTimer) {
    clearInterval(confettiTimer)
    confettiTimer = null
  }
})
</script>

<template>
  <div class="login-page">
    <!-- Gradient top banner with JD wordmark -->
    <div class="banner">
      <van-nav-bar left-arrow @click-left="router.back()" :border="false" background="transparent" />
      <div class="banner-inner">
        <div class="jd-wordmark">JD 京东</div>
        <div class="banner-sub">多 · 快 · 好 · 省</div>
      </div>
    </div>

    <!-- Form card (slides up on mount) -->
    <div class="form-card slide-up">
      <div class="welcome-back">欢迎回来👋</div>
      <p class="welcome">{{ mode === 'login' ? '欢迎登录京东' : '注册京东账号' }}</p>
      <van-cell-group inset>
        <van-field v-model="username" label="用户名" placeholder="请输入用户名" clearable />
        <van-field v-model="password" type="password" label="密码" placeholder="请输入密码" clearable />
        <van-field v-if="mode === 'register'" v-model="nickname" label="昵称" placeholder="选填" clearable />
      </van-cell-group>
      <div class="form-row">
        <span class="forgot-link" @click="forgotPassword">忘记密码</span>
      </div>
      <div style="margin: 16px">
        <van-button type="danger" block round @click="submit">{{ mode === 'login' ? '登 录' : '注 册' }}</van-button>
      </div>
      <div class="switch" @click="mode = mode === 'login' ? 'register' : 'login'">
        {{ mode === 'login' ? '没有账号？去注册' : '已有账号？去登录' }}
      </div>
      <div class="hint">演示账号: admin / admin123</div>

      <!-- Social login buttons row -->
      <div class="social-section">
        <div class="social-divider"><span>其他登录方式</span></div>
        <div class="social-row">
          <div v-for="s in socials" :key="s.key" class="social-btn" @click="socialLogin(s)">
            <span class="social-emoji">{{ s.emoji }}</span>
            <span class="social-label">{{ s.label }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- New user welcome coupon pack (新人专享礼包) -->
    <transition name="welcome-pop">
      <div v-if="showWelcome" class="welcome-mask" @click.self="closeWelcome">
        <div class="welcome-card">
          <!-- Confetti emoji layer -->
          <div class="confetti-layer">
            <span
              v-for="c in confettiPieces"
              :key="c.id"
              class="confetti"
              :style="{
                left: c.left + 'vw',
                animationDelay: c.delay + 's',
                animationDuration: c.duration + 's',
                fontSize: c.size + 'px',
              }"
            >{{ c.emoji }}</span>
          </div>

          <div class="wc-header">
            <div class="wc-emoji">🎉</div>
            <div class="wc-title">新人专享礼包</div>
            <div class="wc-sub">专属于你的迎新福利，立即领取</div>
          </div>

          <div class="wc-coupons">
            <div v-for="(c, i) in demoCoupons" :key="i" class="wc-coupon">
              <div class="wc-cp-value">
                <span class="wc-unit">¥</span>{{ c.value }}
              </div>
              <div class="wc-cp-threshold">满{{ c.threshold }}可用</div>
            </div>
          </div>

          <div class="wc-actions">
            <van-button type="danger" block round size="large" :loading="claiming" @click="claimWelcome">
              立即领取
            </van-button>
            <div class="wc-skip" @click="closeWelcome">下次再说</div>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.login-page { min-height: 100vh; background: #fff; }

/* ---- Gradient top banner (登录页美化) ---- */
.banner {
  background: linear-gradient(135deg, #e1251b 0%, #ff4d4f 55%, #ff7a45 100%);
  color: #fff;
  padding-bottom: 36px;
  border-bottom-left-radius: 24px;
  border-bottom-right-radius: 24px;
}
.banner :deep(.van-nav-bar) { background: transparent; }
.banner :deep(.van-nav-bar .van-icon) { color: #fff; }
.banner-inner { text-align: center; padding: 8px 0 4px; }
.jd-wordmark {
  font-size: 34px;
  font-weight: bold;
  letter-spacing: 2px;
}
.banner-sub {
  margin-top: 6px;
  font-size: 13px;
  opacity: 0.9;
  letter-spacing: 2px;
}

/* ---- Form card (slides up on mount) ---- */
.form-card {
  position: relative;
  margin: -24px 12px 0;
  background: #fff;
  border-radius: 16px;
  padding: 24px 4px 16px;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.08);
  z-index: 2;
}
.slide-up {
  animation: slide-up 0.5s ease-out;
}
@keyframes slide-up {
  from { opacity: 0; transform: translateY(24px); }
  to { opacity: 1; transform: translateY(0); }
}
.welcome-back {
  text-align: center;
  font-size: 22px;
  font-weight: bold;
  color: #e1251b;
}
.welcome { margin: 6px 0 14px; font-size: 14px; color: #999; text-align: center; }
.form-row {
  display: flex;
  justify-content: flex-end;
  padding: 0 28px;
  margin-top: 8px;
}
.forgot-link {
  font-size: 13px;
  color: #e1251b;
}
.switch { text-align: center; color: #e1251b; font-size: 14px; }
.hint { text-align: center; color: #999; font-size: 12px; margin-top: 16px; }

/* ---- Social login row ---- */
.social-section { margin-top: 22px; }
.social-divider {
  display: flex;
  align-items: center;
  text-align: center;
  color: #bbb;
  font-size: 12px;
  margin: 0 24px 16px;
}
.social-divider::before,
.social-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: #eee;
}
.social-divider span { padding: 0 12px; }
.social-row {
  display: flex;
  justify-content: center;
  gap: 28px;
}
.social-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}
.social-emoji {
  width: 46px;
  height: 46px;
  line-height: 46px;
  text-align: center;
  font-size: 24px;
  background: #f7f8fa;
  border-radius: 50%;
}
.social-label {
  font-size: 12px;
  color: #666;
}

/* ---- New user welcome coupon pack (新人专享礼包) ---- */
.welcome-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  padding: 24px;
  overflow: hidden;
}
.welcome-card {
  position: relative;
  width: 100%;
  max-width: 340px;
  border-radius: 20px;
  padding: 28px 20px 20px;
  text-align: center;
  background: linear-gradient(160deg, #e1251b 0%, #ff4d4f 45%, #ff7a45 100%);
  color: #fff;
  box-shadow: 0 12px 36px rgba(225, 37, 27, 0.45);
  overflow: hidden;
  z-index: 1;
}
.wc-header { position: relative; z-index: 2; }
.wc-emoji { font-size: 46px; line-height: 1; }
.wc-title { font-size: 24px; font-weight: bold; margin-top: 8px; letter-spacing: 1px; }
.wc-sub { font-size: 13px; opacity: 0.9; margin-top: 6px; }
.wc-coupons { display: flex; gap: 10px; margin: 22px 0 18px; position: relative; z-index: 2; }
.wc-coupon {
  flex: 1;
  background: #fff;
  color: #e1251b;
  border-radius: 12px;
  padding: 14px 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}
.wc-cp-value { font-size: 28px; font-weight: bold; line-height: 1; }
.wc-cp-value .wc-unit { font-size: 15px; font-weight: bold; }
.wc-cp-threshold { font-size: 11px; color: #ff7a45; margin-top: 6px; }
.wc-actions { position: relative; z-index: 2; }
.wc-skip { font-size: 13px; opacity: 0.85; margin-top: 12px; }

/* Confetti emoji layer */
.confetti-layer { position: absolute; inset: 0; pointer-events: none; z-index: 1; }
.confetti {
  position: absolute;
  top: -40px;
  animation-name: confetti-fall;
  animation-timing-function: linear;
  animation-iteration-count: infinite;
  will-change: transform;
}
@keyframes confetti-fall {
  0% { transform: translateY(-40px) rotate(0deg); opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { transform: translateY(120vh) rotate(360deg); opacity: 0; }
}

/* Popup enter/leave transition */
.welcome-pop-enter-active, .welcome-pop-leave-active { transition: opacity 0.25s; }
.welcome-pop-enter-active .welcome-card, .welcome-pop-leave-active .welcome-card {
  transition: transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.welcome-pop-enter-from, .welcome-pop-leave-to { opacity: 0; }
.welcome-pop-enter-from .welcome-card, .welcome-pop-leave-to .welcome-card {
  transform: scale(0.85);
}
</style>
