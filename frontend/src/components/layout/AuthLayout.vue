<template>
  <div class="auth-shell dark">
    <!-- 认证页固定深色，与首页同一套轨道地球口径 -->
    <OrbitStage mode="auth" />

    <header class="auth-topbar">
      <div class="auth-brand">
        <img
          v-if="settingsLoaded"
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="auth-brand-logo"
        />
        <span class="auth-brand-name">{{ siteName }}</span>
      </div>

      <LocaleSwitcher class="auth-locale" />
    </header>

    <main class="auth-main">
      <div class="auth-card" data-testid="auth-card">
        <slot />
      </div>

      <!-- 公开可靠性指标：仅登录/注册页展示，数据失败时组件自行隐藏 -->
      <PublicStatsStrip v-if="showAuthStats" class="auth-stats" />

      <!-- Footer Links -->
      <div class="auth-footer-links">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="auth-copyright">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted } from 'vue'
import { routeLocationKey } from 'vue-router'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import PublicStatsStrip from '@/components/common/PublicStatsStrip.vue'
import OrbitStage from '@/components/home/OrbitStage.vue'

const appStore = useAppStore()

// 登录/注册页展示公开指标；判断依据为当前路由名。
const route = inject(routeLocationKey, null) as { name?: unknown } | null
const showAuthStats = route?.name === 'Login' || route?.name === 'Register'

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-shell {
  --auth-text: #f1f4f2;
  --auth-dim: rgba(255, 255, 255, 0.62);
  --auth-faint: rgba(255, 255, 255, 0.42);
  --auth-line: rgba(255, 255, 255, 0.12);
  --auth-hover: rgba(255, 255, 255, 0.08);

  position: relative;
  isolation: isolate;
  display: flex;
  min-height: 100vh;
  min-height: 100svh;
  flex-direction: column;
  overflow-x: hidden;
  overflow-y: auto;
  background: #08090b;
  color: var(--auth-text);
  color-scheme: dark;
}

/* 地球背景固定铺满视口，卡片区域独立滚动 */
.auth-shell :deep(.orbit-stage) {
  position: fixed;
}

/* ---------- 顶栏 ---------- */
.auth-topbar {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 18px 22px;
}

@media (min-width: 640px) {
  .auth-topbar {
    padding: 26px 40px;
  }
}

.auth-brand {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.auth-brand-logo {
  height: 33px;
  width: 33px;
  flex-shrink: 0;
  border-radius: 9999px;
  object-fit: contain;
  filter: drop-shadow(0 0 22px rgba(123, 57, 237, 0.55));
}

.auth-brand-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.015em;
  color: var(--auth-text);
}

/* 语言切换器收进顶栏的细边圆角口径，只留旗帜 */
.auth-locale :deep(> button) {
  height: 36px;
  width: 36px;
  justify-content: center;
  gap: 0;
  border-radius: 12px;
  border: 1px solid var(--auth-line);
  background: rgba(255, 255, 255, 0.04);
  padding: 0;
  color: var(--auth-dim);
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.auth-locale :deep(> button:hover) {
  color: var(--auth-text);
  border-color: rgba(255, 255, 255, 0.26);
  background: var(--auth-hover);
}

.auth-locale :deep(> button > span:nth-child(2)),
.auth-locale :deep(> button > svg) {
  display: none;
}

/* ---------- 主体 ---------- */
.auth-main {
  position: relative;
  z-index: 1;
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 28px 18px 56px;
}

.auth-card {
  width: min(411px, 92vw);
  border-radius: 24px;
  border: 1px solid var(--auth-line);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.05), rgba(255, 255, 255, 0.015)),
    rgba(11, 12, 15, 0.78);
  padding: 28px;
  box-shadow:
    0 30px 80px rgba(0, 0, 0, 0.62),
    inset 0 1px 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(18px);
  animation: auth-rise 0.5s cubic-bezier(0.2, 0.7, 0.2, 1) both;
}

@keyframes auth-rise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

/* ---------- 卡片内表单口径 ---------- */
.auth-card :deep(h2) {
  font-family: 'Instrument Serif', 'Songti SC', 'Noto Serif SC', Georgia, serif;
  font-size: 44px;
  font-weight: 400;
  line-height: 1.08;
  letter-spacing: -0.015em;
  text-align: center;
  background: linear-gradient(180deg, #ffffff 12%, rgba(214, 224, 255, 0.72) 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.auth-card :deep(.input-label) {
  margin-bottom: 7px;
  font-size: 12px;
  font-weight: 500;
  color: var(--auth-dim);
}

.auth-card :deep(.input) {
  height: 42px;
  border-radius: 14px;
  border: 1px solid var(--auth-line);
  background: linear-gradient(180deg, rgba(33, 33, 33, 0.38), rgba(7, 7, 7, 0.54));
  padding: 0 16px;
  font-size: 14px;
  color: #ffffff;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.auth-card :deep(.input::placeholder) {
  color: rgba(255, 255, 255, 0.36);
}

.auth-card :deep(.input:focus) {
  border-color: rgba(255, 255, 255, 0.32);
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.06);
  outline: none;
}

.auth-card :deep(.input:disabled) {
  background: rgba(255, 255, 255, 0.03);
}

.auth-card :deep(.input-error) {
  border-color: rgba(248, 113, 113, 0.7);
}

.auth-card :deep(.input-hint) {
  margin-top: 7px;
  font-size: 12px;
  color: var(--auth-faint);
}

.auth-card :deep(.input-error-text) {
  color: #fca5a5;
}

.auth-card :deep(.btn) {
  min-height: 42px;
  border-radius: 14px;
  font-size: 14px;
  font-weight: 600;
}

.auth-card :deep(.btn.btn-primary) {
  border: 1px solid rgba(255, 255, 255, 0.3);
  background-image: linear-gradient(180deg, rgba(255, 255, 255, 0.2), rgba(255, 255, 255, 0.11));
  background-color: transparent;
  color: #ffffff;
  box-shadow:
    0 10px 26px -8px rgba(0, 0, 0, 0.4),
    inset 0 1px 0 rgba(255, 255, 255, 0.3);
}

.auth-card :deep(.btn.btn-primary:hover:not(:disabled)) {
  background-image: linear-gradient(180deg, rgba(255, 255, 255, 0.27), rgba(255, 255, 255, 0.15));
  box-shadow:
    0 12px 30px -8px rgba(0, 0, 0, 0.45),
    inset 0 1px 0 rgba(255, 255, 255, 0.36);
}

.auth-card :deep(.btn.btn-secondary) {
  border: 1px solid var(--auth-line);
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.84);
  box-shadow: none;
}

.auth-card :deep(.btn.btn-secondary:hover:not(:disabled)) {
  background: rgba(255, 255, 255, 0.09);
  border-color: rgba(255, 255, 255, 0.22);
}

/* 卡片内的说明文字与分割线统一到深色口径 */
.auth-card :deep(.text-gray-500),
.auth-card :deep(.text-gray-600),
.auth-card :deep(.text-dark-400),
.auth-card :deep(.text-dark-300) {
  color: var(--auth-dim);
}

.auth-card :deep(.text-gray-900),
.auth-card :deep(.text-gray-700) {
  color: var(--auth-text);
}

/* 输入框内前缀/后缀图标 */
.auth-card :deep(.text-gray-400),
.auth-card :deep(.text-dark-500) {
  color: rgba(255, 255, 255, 0.46);
}

.auth-card :deep(.bg-gray-200),
.auth-card :deep(.bg-dark-700) {
  background-color: rgba(255, 255, 255, 0.1);
}

.auth-card :deep(.auth-accent-link) {
  color: #d6e9fb;
}

.auth-card :deep(.auth-accent-link:hover) {
  color: #ffffff;
}

/* ---------- 卡片下方信息 ---------- */
.auth-stats {
  width: min(411px, 92vw);
}

.auth-footer-links {
  width: min(411px, 92vw);
  text-align: center;
  font-size: 0.875rem;
  color: var(--auth-dim);
}

.auth-footer-links :deep(a) {
  color: #d6e9fb;
}

.auth-footer-links :deep(a:hover) {
  color: #ffffff;
}

.auth-copyright {
  text-align: center;
  font-size: 0.75rem;
  color: var(--auth-faint);
}

@media (max-width: 640px) {
  .auth-card {
    padding: 22px;
  }

  .auth-card :deep(h2) {
    font-size: 34px;
  }

  /* 窄屏只留球体轮廓，网格与噪点收起，避免压住表单 */
  .auth-shell :deep(.orbit-stage-grid),
  .auth-shell :deep(.orbit-stage-grain) {
    display: none;
  }

  .auth-shell :deep(.orbit-stage-canvas) {
    opacity: 0.45;
  }
}

@media (prefers-reduced-motion: reduce) {
  .auth-card {
    animation: none !important;
  }
}
</style>
