<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import Toast from '@/components/common/Toast.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import AdminComplianceDialog from '@/components/admin/AdminComplianceDialog.vue'
import { resolveRouteDocumentTitle } from '@/router/title'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import ComplianceNoticeDialog from '@/components/common/ComplianceNoticeDialog.vue'
import { useAppStore, useAuthStore, useSubscriptionStore, useAnnouncementStore, useAdminComplianceStore, useAdminSettingsStore } from '@/stores'
import { getSetupStatus } from '@/api/setup'
import {
  getComplianceNoticeRevision,
  hasAcceptedComplianceNotice,
  saveComplianceNoticeDecision
} from '@/utils/complianceNotice'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()
const announcementStore = useAnnouncementStore()
const adminComplianceStore = useAdminComplianceStore()
const adminSettingsStore = useAdminSettingsStore()
const complianceNoticeDismissed = ref(false)

const complianceSettings = computed(() => appStore.cachedPublicSettings)
const showComplianceNotice = computed(() => {
  const settings = complianceSettings.value
  return (
    !complianceNoticeDismissed.value &&
    settings?.compliance_notice_enabled === true &&
    !hasAcceptedComplianceNotice(settings)
  )
})
const complianceNoticeBadge = computed(
  () => complianceSettings.value?.compliance_notice_badge || '平台公告'
)
const complianceNoticeTitle = computed(
  () => complianceSettings.value?.compliance_notice_title || 'Codex中转站平台安全与合规管理公告'
)
const complianceNoticeContent = computed(
  () => complianceSettings.value?.compliance_notice_content_md || ''
)
const complianceNoticeAcceptText = computed(
  () => complianceSettings.value?.compliance_notice_accept_text || '以上内容均已看过，本人自愿承担产生后果'
)
const complianceNoticeDeclineText = computed(
  () => complianceSettings.value?.compliance_notice_decline_text || '本人不愿意承担，已拒绝'
)

function updateDocumentTitle() {
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  document.title = resolveRouteDocumentTitle(route, appStore.siteName, customMenuItems)
}

/**
 * Update favicon dynamically
 * @param logoUrl - URL of the logo to use as favicon
 */
function updateFavicon(logoUrl: string) {
  // Find existing favicon link or create new one
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.type = logoUrl.endsWith('.svg') ? 'image/svg+xml' : 'image/x-icon'
  link.href = logoUrl
}

// Watch for site settings changes and update favicon/title
watch(
  () => appStore.siteLogo,
  (newLogo) => {
    if (newLogo) {
      updateFavicon(newLogo)
    }
  },
  { immediate: true }
)

watch(
  [
    () => route.fullPath,
    () => route.meta.title,
    () => route.meta.titleKey,
    () => appStore.siteName,
    () => appStore.cachedPublicSettings?.custom_menu_items,
    () => authStore.isAdmin,
    () => adminSettingsStore.customMenuItems,
  ],
  updateDocumentTitle,
  { deep: true }
)

// Watch for authentication state and manage subscription data + announcements
function onVisibilityChange() {
  if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
}

function onAdminComplianceRequired(event: Event) {
  const detail = (event as CustomEvent<Record<string, string>>).detail || {}
  adminComplianceStore.requireAcknowledgement(detail)
}

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated, oldValue) => {
    if (isAuthenticated) {
      if (authStore.isAdmin) {
        adminComplianceStore.fetchStatus().catch((error) => {
          console.error('Failed to fetch admin compliance status:', error)
        })
      }

      // User logged in: preload subscriptions and start polling
      subscriptionStore.fetchActiveSubscriptions().catch((error) => {
        console.error('Failed to preload subscriptions:', error)
      })
      subscriptionStore.startPolling()

      // Announcements: new login vs page refresh restore
      if (oldValue === false) {
        // New login: delay 3s then force fetch
        setTimeout(() => announcementStore.fetchAnnouncements(true), 3000)
      } else {
        // Page refresh restore (oldValue was undefined)
        announcementStore.fetchAnnouncements()
      }

      // Register visibility change listener
      document.addEventListener('visibilitychange', onVisibilityChange)
    } else {
      // User logged out: clear data and stop polling
      subscriptionStore.clear()
      announcementStore.reset()
      adminComplianceStore.reset()
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  },
  { immediate: true }
)

// Route change trigger (throttled by store)
router.afterEach(() => {
  if (authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
  window.removeEventListener('admin-compliance-required', onAdminComplianceRequired)
})

function handleComplianceAccept() {
  const revision = getComplianceNoticeRevision(complianceSettings.value)
  if (revision) {
    saveComplianceNoticeDecision(revision, 'accepted')
  }
  complianceNoticeDismissed.value = true
}

function handleComplianceDecline() {
  const revision = getComplianceNoticeRevision(complianceSettings.value)
  if (revision) {
    saveComplianceNoticeDecision(revision, 'declined')
  }
  complianceNoticeDismissed.value = true
  appStore.showWarning('已拒绝平台公告内容，登录和注册操作将被拦截。')
}

onMounted(async () => {
  window.addEventListener('admin-compliance-required', onAdminComplianceRequired)

  // Check if setup is needed
  try {
    const status = await getSetupStatus()
    if (status.needs_setup && route.path !== '/setup') {
      router.replace('/setup')
      return
    }
  } catch {
    // If setup endpoint fails, assume normal mode and continue
  }

  // Load public settings into appStore (will be cached for other components)
  await appStore.fetchPublicSettings()

  // Re-resolve document title now that site settings are available
  updateDocumentTitle()
})
</script>

<template>
  <NavigationProgress />
  <RouterView />
  <Toast />
  <AnnouncementPopup />
  <ComplianceNoticeDialog
    :visible="showComplianceNotice"
    :badge="complianceNoticeBadge"
    :title="complianceNoticeTitle"
    :content="complianceNoticeContent"
    :accept-text="complianceNoticeAcceptText"
    :decline-text="complianceNoticeDeclineText"
    @accept="handleComplianceAccept"
    @decline="handleComplianceDecline"
  />
  <AdminComplianceDialog />
</template>
