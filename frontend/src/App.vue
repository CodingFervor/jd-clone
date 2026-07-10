<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const showTabbar = computed(() => route.meta.tab !== undefined)

const tabs = [
  { name: 'home', icon: 'wap-home-o', label: '首页' },
  { name: 'category', icon: 'apps-o', label: '分类' },
  { name: 'discover', icon: 'eye-o', label: '发现' },
  { name: 'cart', icon: 'shopping-cart-o', label: '购物车' },
  { name: 'mine', icon: 'contact', label: '我的' },
]
const active = computed(() => route.meta.tab ?? 0)

// ---- Dark mode (深色模式) ----
// Single source of truth for dark mode. Read from localStorage on init so
// the correct class is applied before first paint avoids a flash.
const darkMode = ref(localStorage.getItem('jd_dark_mode') === 'true')

function applyDark(val) {
  // Toggle the dark-mode class on the root .app-wrap element. The actual
  // color overrides live in style.css (global) so they affect all views.
  const el = document.querySelector('.app-wrap')
  if (el) el.classList.toggle('dark-mode', val)
  // Also toggle on body so fixed/sticky chrome (nav bars, tabbar) matches.
  document.body.classList.toggle('jd-dark', val)
}

watch(darkMode, (val) => {
  localStorage.setItem('jd_dark_mode', String(val))
  applyDark(val)
})

// React to changes from other tabs/components (e.g. the toggle in Mine.vue).
function syncFromStorage() {
  darkMode.value = localStorage.getItem('jd_dark_mode') === 'true'
}

onMounted(() => {
  applyDark(darkMode.value)
  window.addEventListener('storage', syncFromStorage)
  // Expose a global so Mine.vue can flip the ref without importing it.
  window.__setJdDarkMode = (val) => {
    darkMode.value = !!val
  }
})

function toggleDark() {
  darkMode.value = !darkMode.value
}
</script>

<template>
  <div class="app-wrap" :class="{ 'dark-mode': darkMode }">
    <router-view v-slot="{ Component }">
      <keep-alive include="Home,Category">
        <component :is="Component" />
      </keep-alive>
    </router-view>
    <van-tabbar v-if="showTabbar" v-model="active" route active-color="#e1251b" inactive-color="#7d7e80">
      <van-tabbar-item v-for="t in tabs" :key="t.name" :to="{ name: t.name }" :icon="t.icon">
        {{ t.label }}
      </van-tabbar-item>
    </van-tabbar>
  </div>
</template>
