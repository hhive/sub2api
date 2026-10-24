<template>
  <div class="auth-shell relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <!-- 深色科技背景：极光 / 网格 / 扫描线 / 指针光斑，所有认证页共用 -->
    <AuroraBackdrop variant="auth" pointer-glow />

    <div
      class="relative z-10 grid w-full max-w-6xl items-center gap-10 py-10 lg:grid-cols-[1.05fr_minmax(0,420px)] lg:gap-16"
    >
      <!-- 左栏：品牌与沉浸式标题 -->
      <div data-testid="auth-hero" class="text-center lg:text-left">
        <div class="flex items-center justify-center gap-3 lg:justify-start">
          <div
            class="auth-logo inline-flex h-12 w-12 items-center justify-center overflow-hidden rounded-2xl"
          >
            <img
              v-if="settingsLoaded"
              :src="siteLogo || '/logo.svg'"
              alt="Logo"
              class="h-full w-full object-contain"
            />
          </div>
          <span class="auth-brand-title truncate font-brand text-lg font-semibold tracking-tight">
            {{ siteName }}
          </span>
        </div>

        <h1 class="auth-hero-title mt-8 font-display">
          <span class="block">{{ t('auth.hero.titlePrimary') }}</span>
          <span class="auth-hero-title-accent block">{{ t('auth.hero.titleAccent') }}</span>
        </h1>

        <p v-if="settingsLoaded" class="auth-subtitle mt-5 text-sm sm:text-base">
          {{ siteSubtitle }}
        </p>

        <HeroProofStrip class="mt-8 lg:justify-start" :items="proofItems" />
      </div>

      <!-- 右栏：表单卡片 -->
      <div class="w-full max-w-md justify-self-center lg:justify-self-end">
        <div class="auth-card-shell">
          <div class="card-glass rounded-2xl p-8 shadow-glass">
            <slot />
          </div>
        </div>

        <!-- 公开可靠性指标：仅登录/注册页展示，数据失败时组件自行隐藏 -->
        <PublicStatsStrip v-if="showAuthStats" />

        <!-- Footer Links -->
        <div class="auth-footer-links mt-6 text-center text-sm">
          <slot name="footer" />
        </div>

        <!-- Copyright -->
        <div class="auth-copyright mt-8 text-center text-xs">
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted } from 'vue'
import { routeLocationKey } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import AuroraBackdrop from '@/components/common/AuroraBackdrop.vue'
import HeroProofStrip from '@/components/common/HeroProofStrip.vue'
import PublicStatsStrip from '@/components/common/PublicStatsStrip.vue'

const { t } = useI18n()

const appStore = useAppStore()

// 登录/注册页展示公开指标；判断依据为当前路由名。
const route = inject(routeLocationKey, null) as { name?: unknown } | null
const showAuthStats = route?.name === 'Login' || route?.name === 'Register'

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

// 首屏三条证明文案
const proofItems = computed(() => [
  t('auth.hero.proof.gateway'),
  t('auth.hero.proof.billing'),
  t('auth.hero.proof.console'),
])

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-shell {
  --auth-text: #0f172a;
  --auth-text-dim: #475569;
  --auth-accent: #7c3aed;
  --auth-accent-soft: #a78bfa;
  background: #f5f4fb;
}

html.dark .auth-shell {
  --auth-text: #ffffff;
  --auth-text-dim: #a9b8d4;
  --auth-accent: #c4b5fd;
  --auth-accent-soft: #8b5cf6;
  background: #000000;
}

.auth-logo {
  background: linear-gradient(150deg, rgba(123, 57, 237, 0.28), rgba(90, 32, 190, 0.22));
  border: 1px solid rgba(167, 139, 250, 0.4);
  box-shadow: 0 0 30px rgba(123, 57, 237, 0.28), 0 10px 30px rgba(90, 32, 190, 0.24);
}

.auth-brand-title {
  color: var(--auth-text);
}

.auth-hero-title {
  font-size: clamp(2.25rem, 6vw, 4.5rem);
  font-weight: 400;
  line-height: 1.02;
  letter-spacing: -0.02em;
  color: var(--auth-text);
}

.auth-hero-title-accent {
  background-image: linear-gradient(100deg, #111827 0%, #4c1d95 52%, #7b39ed 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

html.dark .auth-hero-title-accent {
  background-image: linear-gradient(100deg, #ffffff 0%, #ded2ff 48%, #a78bfa 100%);
}

.auth-subtitle {
  color: var(--auth-text-dim);
}

.auth-card-shell {
  padding: 1px;
  border-radius: 18px;
  background: linear-gradient(
    150deg,
    rgba(123, 57, 237, 0.55),
    rgba(139, 92, 246, 0.42) 46%,
    rgba(255, 255, 255, 0.08)
  );
  box-shadow: 0 24px 70px rgba(26, 12, 56, 0.35), 0 0 40px rgba(123, 57, 237, 0.14);
  animation: auth-rise 0.55s cubic-bezier(0.2, 0.7, 0.2, 1) both;
}

html.dark .auth-card-shell {
  box-shadow: 0 24px 70px rgba(3, 8, 24, 0.65), 0 0 40px rgba(123, 57, 237, 0.16);
}

.auth-card-shell > .card-glass {
  border-radius: 17px;
  background-color: rgba(255, 255, 255, 0.86);
}

/* 认证页统一强调色：卡片标题与主按钮跟随落地页的紫色口径 */
.auth-card-shell :deep(h2) {
  background-image: linear-gradient(100deg, #4c1d95, #7b39ed 42%, #a78bfa 92%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.auth-card-shell :deep(.btn.btn-primary) {
  background-image: linear-gradient(112deg, #a78bfa, #7b39ed 52%, #5a20be);
  border: 1px solid rgba(167, 139, 250, 0.45);
  color: #ffffff;
  box-shadow: 0 10px 30px rgba(123, 57, 237, 0.3), 0 8px 28px rgba(90, 32, 190, 0.26);
}

.auth-card-shell :deep(.btn.btn-primary:hover:not(:disabled)) {
  background-image: linear-gradient(112deg, #bda4ff, #8b5cf6 52%, #6d28d9);
  box-shadow: 0 14px 38px rgba(123, 57, 237, 0.4), 0 10px 34px rgba(90, 32, 190, 0.32);
}

.auth-card-shell :deep(.btn.btn-primary svg) {
  color: inherit;
}

html.dark .auth-card-shell > .card-glass {
  background-color: rgba(15, 23, 42, 0.8);
}

.auth-footer-links {
  color: var(--auth-text-dim);
}

.auth-footer-links :deep(a) {
  color: var(--auth-accent);
}

.auth-copyright {
  color: var(--auth-text-dim);
  opacity: 0.72;
}

@keyframes auth-rise {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: none; }
}

@media (prefers-reduced-motion: reduce) {
  .auth-card-shell {
    animation: none !important;
  }
}
</style>
