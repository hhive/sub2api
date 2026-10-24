<template>
  <!-- 三条短证明文案：首页与认证页 hero 下方共用 -->
  <ul
    data-testid="hero-proof-strip"
    class="flex flex-col items-center gap-2.5 text-sm text-gray-500 sm:flex-row sm:flex-wrap sm:justify-center sm:gap-x-7 dark:text-white/55"
  >
    <li v-for="item in items" :key="item" class="flex items-center gap-2.5">
      <span class="hidden h-1 w-1 rounded-full bg-primary-500 sm:inline-block dark:bg-white/40"></span>
      <span>{{ item }}</span>
    </li>
  </ul>
</template>

<script setup lang="ts">
defineProps<{ items: string[] }>()
</script>
