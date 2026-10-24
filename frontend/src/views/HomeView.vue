<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Immersive Home Page: 轨道地球主视觉 -->
  <div v-else class="home-page">
    <!-- Header -->
    <header class="home-header">
      <nav class="home-header-nav">
        <div class="home-brand">
          <img :src="siteLogo || '/logo.svg'" alt="Logo" class="home-brand-logo" />
          <span class="home-brand-name">{{ siteName }}</span>
        </div>

        <div class="home-header-actions">
          <LocaleSwitcher class="home-locale" />

          <div ref="moreRef" class="home-more">
            <button
              type="button"
              class="home-icon-btn"
              data-testid="home-more-button"
              :title="t('home.hero.moreMenu')"
              aria-haspopup="menu"
              :aria-expanded="moreOpen"
              @click="moreOpen = !moreOpen"
            >
              <Icon name="more" size="md" />
            </button>

            <div v-if="moreOpen" class="home-more-panel" role="menu" data-testid="home-more-panel">
              <a
                v-if="docUrl"
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="home-more-item"
                role="menuitem"
                @click="moreOpen = false"
              >
                <Icon name="book" size="sm" />
                <span>{{ t('home.viewDocs') }}</span>
              </a>

              <router-link
                v-if="showModelPlazaEntry"
                to="/model-plaza"
                class="home-more-item"
                role="menuitem"
                @click="moreOpen = false"
              >
                <Icon name="grid" size="sm" />
                <span>{{ t('nav.modelPlaza') }}</span>
              </router-link>

              <button type="button" class="home-more-item" role="menuitem" @click="onToggleTheme">
                <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
                <span>{{ isDark ? t('home.switchToLight') : t('home.switchToDark') }}</span>
              </button>
            </div>
          </div>

          <router-link v-if="isAuthenticated" :to="dashboardPath" class="home-account-btn">
            <span class="home-account-avatar">{{ userInitial }}</span>
            <span>{{ t('home.dashboard') }}</span>
          </router-link>
          <router-link v-else to="/login" class="home-login-btn">{{ t('home.login') }}</router-link>
        </div>
      </nav>
    </header>

    <HomeOrbitHero
      :slides="heroSlides"
      :is-authenticated="isAuthenticated"
      :dashboard-path="dashboardPath"
      :show-model-plaza-entry="showModelPlazaEntry"
      :contact-info="contactInfo"
      @slide-change="onSlideChange"
      @contact-click="onContactClick"
    />

    <!-- Footer -->
    <footer class="home-footer">
      <span>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</span>
    </footer>

    <div class="home-transport">
      <span class="home-counter" data-testid="home-slide-counter">
        <b>{{ slideLabel }}</b>
        <span class="home-counter-total">/ {{ slideTotalLabel }}</span>
      </span>
      <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="home-plaza-link">
        <Icon name="grid" size="sm" />
        <span>{{ t('home.hero.reelLabel') }}</span>
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import HomeOrbitHero from '@/components/home/HomeOrbitHero.vue'
import type { HeroSlide } from '@/components/home/types'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t, tm, rt } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const contactInfo = computed(() => (appStore.cachedPublicSettings?.contact_info || '').trim())
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// 二级菜单
const moreOpen = ref(false)
const moreRef = ref<HTMLElement | null>(null)

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(
  () => appStore.cachedPublicSettings?.model_plaza_require_auth === true,
)
const showModelPlazaEntry = computed(
  () => modelPlazaEnabled.value && (isAuthenticated.value || !modelPlazaRequiresAuth.value),
)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// 首屏轮播文案
const slideIndex = ref(0)

// vue-i18n 的消息返回值类型递归很深，这里只当作结构未知的数据处理
function readMessage(key: string): unknown {
  return (tm as unknown as (messageKey: string) => unknown)(key)
}

function field(source: unknown, key: string): unknown {
  if (!source || typeof source !== 'object') return undefined
  return (source as Record<string, unknown>)[key]
}

function asText(value: unknown): string {
  if (typeof value === 'string') return value
  if (value === undefined || value === null) return ''
  return String(rt(value as never))
}

const heroSlides = computed<HeroSlide[]>(() => {
  const raw = readMessage('home.hero.slides')
  if (!raw || typeof raw !== 'object') return []

  // i18n 叶子只能是字符串，故轮播文案以 s1..s5 键对象存放，按序号排序取用
  const entries = raw as Record<string, unknown>

  return Object.keys(entries)
    .sort()
    .map((key) => {
      const item = entries[key]
      const left = field(item, 'left')
      const right = field(item, 'right')

      return {
        prefix: asText(field(item, 'prefix')),
        verb: asText(field(item, 'verb')),
        subtitle: asText(field(item, 'subtitle')),
        left: {
          kicker: asText(field(left, 'kicker')),
          title: asText(field(left, 'title')),
          desc: asText(field(left, 'desc'))
        },
        right: {
          kicker: asText(field(right, 'kicker')),
          title: asText(field(right, 'title')),
          desc: asText(field(right, 'desc'))
        }
      }
    })
})

const slideTotal = computed(() => heroSlides.value.length)
const slideTotalLabel = computed(() => String(slideTotal.value).padStart(2, '0'))
const slideLabel = computed(() => String(slideIndex.value + 1).padStart(2, '0'))

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

function onSlideChange(index: number): void {
  slideIndex.value = index
}

async function onContactClick(): Promise<void> {
  const value = contactInfo.value
  if (!value) return

  try {
    await navigator.clipboard.writeText(value)
    appStore.showSuccess(t('home.hero.contactCopied'))
  } catch {
    appStore.showWarning(t('home.hero.contactCopyFailed'))
  }
}

function onToggleTheme(): void {
  moreOpen.value = false
  toggleTheme()
}

function onDocumentPointerDown(event: PointerEvent): void {
  if (!moreOpen.value) return
  const root = moreRef.value
  if (root && event.target instanceof Node && root.contains(event.target)) return
  moreOpen.value = false
}

function onDocumentKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && moreOpen.value) {
    moreOpen.value = false
  }
}

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }

  document.addEventListener('pointerdown', onDocumentPointerDown)
  document.addEventListener('keydown', onDocumentKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  document.removeEventListener('keydown', onDocumentKeydown)
})
</script>

<style scoped>
.home-page {
  --pg-bg: #0a0a0c;
  --pg-text: #f4f3f7;
  --pg-dim: rgba(255, 255, 255, 0.56);
  --pg-faint: rgba(255, 255, 255, 0.38);
  --pg-line: rgba(255, 255, 255, 0.12);
  --pg-hover: rgba(255, 255, 255, 0.08);

  position: relative;
  min-height: 100vh;
  background: var(--pg-bg);
  color-scheme: dark;
  color: var(--pg-text);
}

/* ---------- Header ---------- */
.home-header {
  position: absolute;
  inset: 0 0 auto;
  z-index: 20;
  padding: 18px 22px;
}

@media (min-width: 640px) {
  .home-header {
    padding: 26px 40px;
  }
}

.home-header-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 0 auto;
  max-width: 1560px;
}

.home-brand {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.home-brand-logo {
  height: 33px;
  width: 33px;
  flex-shrink: 0;
  border-radius: 9999px;
  object-fit: contain;
  filter: drop-shadow(0 0 22px rgba(123, 57, 237, 0.55));
}

.home-brand-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.015em;
  color: var(--pg-text);
}

.home-header-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
}

/* 顶栏图标按钮：细边圆形，对齐目标站的地球按钮口径 */
.home-icon-btn {
  display: inline-flex;
  height: 36px;
  width: 36px;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  border: 1px solid var(--pg-line);
  background: rgba(255, 255, 255, 0.04);
  color: var(--pg-dim);
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.home-icon-btn:hover,
.home-icon-btn[aria-expanded='true'] {
  color: var(--pg-text);
  border-color: rgba(255, 255, 255, 0.26);
  background: var(--pg-hover);
}

/* 语言切换器：把它自带的那套浅色控件拉进本页的圆形细边口径 */
.home-locale :deep(> button) {
  height: 36px;
  width: 36px;
  justify-content: center;
  gap: 0;
  border-radius: 12px;
  border: 1px solid var(--pg-line);
  background: rgba(255, 255, 255, 0.04);
  padding: 0;
  color: var(--pg-dim);
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.home-locale :deep(> button:hover) {
  color: var(--pg-text);
  border-color: rgba(255, 255, 255, 0.26);
  background: var(--pg-hover);
}

/* 圆形按钮里塞不下语言代码与箭头，只保留旗帜 */
.home-locale :deep(> button > span:nth-child(2)),
.home-locale :deep(> button > svg) {
  display: none;
}

/* ---------- 二级菜单 ---------- */
.home-more {
  position: relative;
}

.home-more-panel {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  z-index: 30;
  display: flex;
  min-width: 168px;
  flex-direction: column;
  gap: 2px;
  border-radius: 12px;
  border: 1px solid var(--pg-line);
  background: rgba(14, 16, 20, 0.94);
  padding: 6px;
  box-shadow: 0 20px 48px rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(14px);
}

.home-more-item {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 9px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  padding: 8px 10px;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 12.5px;
  color: var(--pg-dim);
  text-align: left;
  cursor: pointer;
  transition:
    color 0.18s ease,
    background-color 0.18s ease;
}

.home-more-item:hover {
  color: var(--pg-text);
  background: rgba(255, 255, 255, 0.07);
}

.home-login-btn,
.home-account-btn {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 2px;
  padding: 0 18px;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 13px;
  font-weight: 500;
  transition:
    background-color 0.2s ease,
    border-color 0.2s ease,
    opacity 0.2s ease;
}

.home-login-btn {
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: transparent;
  color: var(--pg-text);
}

.home-login-btn:hover {
  border-color: rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.06);
}

.home-account-btn {
  padding-left: 5px;
  border: 1px solid var(--pg-line);
  color: var(--pg-text);
}

.home-account-btn:hover {
  background: var(--pg-hover);
}

.home-account-avatar {
  display: inline-flex;
  height: 26px;
  width: 26px;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background-image: linear-gradient(135deg, #a78bfa, #7c3aed);
  font-size: 11px;
  font-weight: 700;
  color: #ffffff;
}

/* ---------- Footer ---------- */
.home-footer {
  position: absolute;
  left: 22px;
  bottom: 18px;
  z-index: 12;
  font-size: 11.5px;
  color: var(--pg-faint);
}

@media (min-width: 640px) {
  .home-footer {
    left: 40px;
    bottom: 44px;
  }
}

/* ---------- 右下：轮播计数与模型广场入口 ---------- */
.home-transport {
  position: absolute;
  right: 22px;
  bottom: 18px;
  z-index: 12;
  display: flex;
  align-items: center;
  gap: 10px;
}

@media (min-width: 640px) {
  .home-transport {
    right: 40px;
    bottom: 44px;
  }
}

.home-counter {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  font-family: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.42);
  font-variant-numeric: tabular-nums;
}

.home-counter b {
  font-size: 15px;
  font-weight: 400;
  color: rgba(255, 255, 255, 0.82);
}

.home-counter-total {
  font-size: 12px;
}

@media (max-width: 639px) {
  .home-counter {
    display: none;
  }
}

.home-plaza-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: transparent;
  padding: 8px 14px;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 12px;
  color: #b7bdc5;
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.home-plaza-link:hover {
  color: var(--pg-text);
  border-color: rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.06);
}
</style>
