<script setup>
import { computed } from 'vue'
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
</script>

<template>
  <div class="app-wrap">
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
