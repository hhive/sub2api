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

  <!-- Immersive Home Page -->
  <div
    v-else
    class="relative flex min-h-screen flex-col overflow-hidden bg-white text-gray-900 dark:bg-black dark:text-white"
  >
    <AuroraBackdrop />

    <!-- Header -->
    <header class="relative z-20 px-5 py-5 sm:px-8">
      <nav class="mx-auto flex max-w-6xl items-center justify-between gap-3">
        <div class="flex min-w-0 items-center gap-2.5">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-xl object-contain"
          />
          <span class="min-w-0 truncate font-brand text-base font-semibold tracking-tight">{{
            siteName
          }}</span>
        </div>

        <div class="flex shrink-0 items-center gap-1.5 sm:gap-2">
          <LocaleSwitcher />

          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="home-header-link"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="home-header-link" :title="t('nav.modelPlaza')">
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>

          <button
            class="home-header-link"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <router-link v-if="isAuthenticated" :to="dashboardPath" class="home-dashboard-pill">
            <span class="home-dashboard-avatar">{{ userInitial }}</span>
            <span>{{ t('home.dashboard') }}</span>
          </router-link>
          <router-link v-else to="/login" class="home-login-pill">{{ t('home.login') }}</router-link>
        </div>
      </nav>
    </header>

    <!-- Hero -->
    <main
      class="relative z-10 flex flex-1 flex-col items-center justify-center px-5 py-16 text-center sm:px-8 sm:py-20"
    >
      <h1 class="home-hero-title font-display">
        <span class="block">{{ t('home.hero.titlePrimary') }}</span>
        <span class="home-hero-title-accent block">{{ t('home.hero.titleAccent') }}</span>
      </h1>

      <p class="home-hero-subtitle">{{ t('home.hero.subtitle') }}</p>

      <div class="mt-10 flex flex-col items-center gap-3 sm:flex-row">
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          data-testid="home-primary-cta"
          class="home-cta-primary"
        >
          {{ isAuthenticated ? t('home.hero.primaryCtaAuthed') : t('home.hero.primaryCta') }}
          <Icon name="arrowRight" size="sm" class="ml-1.5" :stroke-width="2" />
        </router-link>
        <router-link
          v-if="showModelPlazaEntry"
          to="/model-plaza"
          data-testid="home-secondary-cta"
          class="home-cta-secondary"
        >
          {{ t('home.hero.secondaryCta') }}
        </router-link>
      </div>

      <HeroProofStrip class="mt-14" :items="proofItems" />
    </main>

    <!-- Footer -->
    <footer class="relative z-10 px-5 py-6 text-center text-xs text-gray-400 sm:px-8 dark:text-white/35">
      &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import AuroraBackdrop from '@/components/common/AuroraBackdrop.vue'
import HeroProofStrip from '@/components/common/HeroProofStrip.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
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

// 首屏三条证明文案
const proofItems = computed(() => [
  t('home.hero.proof.gateway'),
  t('home.hero.proof.billing'),
  t('home.hero.proof.console'),
])

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

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
})
</script>

<style scoped>
.home-header-link {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  border-radius: 0.5rem;
  padding: 0.5rem;
  color: #6b7280;
  font-size: 0.875rem;
  transition: color 0.2s ease, background-color 0.2s ease;
}

.home-header-link:hover {
  color: #111827;
  background-color: rgba(17, 24, 39, 0.06);
}

html.dark .home-header-link {
  color: rgba(255, 255, 255, 0.6);
}

html.dark .home-header-link:hover {
  color: #ffffff;
  background-color: rgba(255, 255, 255, 0.1);
}

.home-login-pill,
.home-dashboard-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  border-radius: 9999px;
  padding: 0.4rem 0.9rem;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 0.8125rem;
  font-weight: 600;
  background-color: #111827;
  color: #ffffff;
  transition: background-color 0.2s ease, transform 0.2s ease;
}

.home-login-pill:hover,
.home-dashboard-pill:hover {
  background-color: #1f2937;
  transform: translateY(-1px);
}

html.dark .home-login-pill,
html.dark .home-dashboard-pill {
  background-color: #ffffff;
  color: #000000;
}

html.dark .home-login-pill:hover,
html.dark .home-dashboard-pill:hover {
  background-color: #ede9fe;
}

.home-dashboard-pill {
  padding-left: 0.3rem;
}

.home-dashboard-avatar {
  display: inline-flex;
  height: 1.25rem;
  width: 1.25rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background-image: linear-gradient(135deg, #a78bfa, #7b39ed);
  font-size: 0.625rem;
  font-weight: 700;
  color: #ffffff;
}

.home-hero-title {
  font-size: clamp(2.75rem, 9vw, 6.75rem);
  font-weight: 400;
  line-height: 0.98;
  letter-spacing: -0.02em;
}

.home-hero-title-accent {
  background-image: linear-gradient(100deg, #111827 0%, #4c1d95 52%, #7b39ed 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

html.dark .home-hero-title-accent {
  background-image: linear-gradient(100deg, #ffffff 0%, #ded2ff 48%, #a78bfa 100%);
}

.home-hero-subtitle {
  margin-top: 1.5rem;
  max-width: 34rem;
  font-size: 1rem;
  line-height: 1.7;
  color: #4b5563;
}

html.dark .home-hero-subtitle {
  color: rgba(255, 255, 255, 0.62);
}

@media (min-width: 640px) {
  .home-hero-subtitle {
    font-size: 1.125rem;
  }
}

.home-cta-primary,
.home-cta-secondary {
  display: inline-flex;
  min-height: 3rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  padding: 0 1.75rem;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 0.95rem;
  font-weight: 600;
  transition: transform 0.2s ease, box-shadow 0.2s ease, background-color 0.2s ease,
    border-color 0.2s ease;
}

.home-cta-primary {
  background-color: #111827;
  color: #ffffff;
  box-shadow: 0 14px 38px rgba(123, 57, 237, 0.24);
}

.home-cta-primary:hover {
  background-color: #1f2937;
  transform: translateY(-1px);
  box-shadow: 0 18px 44px rgba(123, 57, 237, 0.32);
}

html.dark .home-cta-primary {
  background-color: #ffffff;
  color: #000000;
  box-shadow: 0 14px 38px rgba(123, 57, 237, 0.35);
}

html.dark .home-cta-primary:hover {
  background-color: #ede9fe;
  box-shadow: 0 18px 44px rgba(123, 57, 237, 0.45);
}

.home-cta-secondary {
  border: 1px solid rgba(123, 57, 237, 0.32);
  color: #4c1d95;
  background-color: transparent;
}

.home-cta-secondary:hover {
  border-color: rgba(123, 57, 237, 0.55);
  background-color: rgba(123, 57, 237, 0.08);
  transform: translateY(-1px);
}

html.dark .home-cta-secondary {
  border-color: rgba(255, 255, 255, 0.24);
  color: rgba(255, 255, 255, 0.88);
}

html.dark .home-cta-secondary:hover {
  border-color: rgba(255, 255, 255, 0.45);
  background-color: rgba(255, 255, 255, 0.1);
}
</style>
