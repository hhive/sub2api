<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useId, watch, type CSSProperties } from 'vue'

import type { UserSupportedModel } from '@/types'
import { platformLabel } from '@/utils/platformColors'

const props = defineProps<{
  models: UserSupportedModel[]
  emptyLabel: string
}>()

const PREVIEW_COUNT = 3
const POPOVER_WIDTH = 288
const POPOVER_MARGIN = 8
const CLOSE_DELAY = 150

const triggerRef = ref<HTMLButtonElement | null>(null)
const popoverRef = ref<HTMLElement | null>(null)
const popoverId = `key-models-popover-${useId()}`

const hovered = ref(false)
const focused = ref(false)
const pinned = ref(false)
const popoverStyle = ref<CSSProperties>({ width: `${POPOVER_WIDTH}px` })

const open = computed(() => hovered.value || focused.value || pinned.value)
const visibleModels = computed(() => props.models.slice(0, PREVIEW_COUNT))
const hiddenModelCount = computed(() => Math.max(props.models.length - PREVIEW_COUNT, 0))

const groupedModels = computed(() => {
  const buckets = new Map<string, UserSupportedModel[]>()
  for (const model of props.models) {
    const bucket = buckets.get(model.platform) ?? []
    bucket.push(model)
    buckets.set(model.platform, bucket)
  }
  return Array.from(buckets.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([platform, models]) => ({
      platform,
      models: [...models].sort((left, right) =>
        left.name.localeCompare(right.name, undefined, { numeric: true })
      ),
    }))
})

let closeTimer: number | undefined

const clearCloseTimer = () => {
  if (closeTimer !== undefined) {
    window.clearTimeout(closeTimer)
    closeTimer = undefined
  }
}

const scheduleClose = () => {
  clearCloseTimer()
  closeTimer = window.setTimeout(() => {
    hovered.value = false
    closeTimer = undefined
  }, CLOSE_DELAY)
}

const close = () => {
  clearCloseTimer()
  hovered.value = false
  focused.value = false
  pinned.value = false
}

const handleMouseEnter = () => {
  clearCloseTimer()
  hovered.value = true
}

const handleMouseLeave = () => {
  if (!pinned.value) scheduleClose()
}

const handleFocus = () => {
  clearCloseTimer()
  focused.value = true
}

const handleBlur = () => {
  focused.value = false
}

const togglePinned = () => {
  if (pinned.value) {
    close()
    return
  }
  clearCloseTimer()
  hovered.value = true
  pinned.value = true
}

const updatePosition = () => {
  const trigger = triggerRef.value
  if (!trigger) return

  const rect = trigger.getBoundingClientRect()
  const popoverHeight = popoverRef.value?.offsetHeight ?? 0
  const spaceBelow = window.innerHeight - rect.bottom
  const openUp = popoverHeight > 0
    && spaceBelow < popoverHeight + POPOVER_MARGIN
    && rect.top > popoverHeight + POPOVER_MARGIN
  const width = Math.min(POPOVER_WIDTH, window.innerWidth - POPOVER_MARGIN * 2)
  const left = Math.min(
    Math.max(POPOVER_MARGIN, rect.left),
    Math.max(POPOVER_MARGIN, window.innerWidth - width - POPOVER_MARGIN),
  )
  const top = openUp ? rect.top - popoverHeight - 6 : rect.bottom + 6

  popoverStyle.value = {
    left: `${left}px`,
    top: `${Math.max(POPOVER_MARGIN, top)}px`,
    width: `${width}px`,
  }
}

const handleDocumentPointerDown = (event: PointerEvent) => {
  const target = event.target as Node | null
  const path = event.composedPath()
  const isInside = (element: HTMLElement | null): boolean => {
    if (!element) return false
    return path.includes(element) || (target !== null && element.contains(target))
  }
  if (isInside(triggerRef.value) || isInside(popoverRef.value)) return
  close()
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') close()
}

watch(open, (isOpen) => {
  if (isOpen) {
    document.addEventListener('pointerdown', handleDocumentPointerDown, true)
    window.addEventListener('keydown', handleKeydown)
    window.addEventListener('scroll', updatePosition, true)
    window.addEventListener('resize', updatePosition)
    void nextTick(updatePosition)
  } else {
    document.removeEventListener('pointerdown', handleDocumentPointerDown, true)
    window.removeEventListener('keydown', handleKeydown)
    window.removeEventListener('scroll', updatePosition, true)
    window.removeEventListener('resize', updatePosition)
  }
})

watch(() => props.models.length, (length) => {
  if (length === 0) close()
})

onBeforeUnmount(() => {
  clearCloseTimer()
  document.removeEventListener('pointerdown', handleDocumentPointerDown, true)
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('scroll', updatePosition, true)
  window.removeEventListener('resize', updatePosition)
})
</script>

<template>
  <div class="relative">
    <span
      v-if="models.length === 0"
      data-test="key-models-empty"
      class="text-[11px] text-gray-400 dark:text-dark-500"
    >
      {{ emptyLabel }}
    </span>
    <button
      v-else
      ref="triggerRef"
      type="button"
      data-test="key-models-trigger"
      class="-mx-1 flex max-w-[320px] cursor-pointer flex-wrap items-center gap-1 rounded px-1 py-0.5 text-left transition-colors hover:bg-gray-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-300 dark:hover:bg-dark-700"
      aria-haspopup="dialog"
      :aria-expanded="open"
      :aria-controls="popoverId"
      @mouseenter="handleMouseEnter"
      @mouseleave="handleMouseLeave"
      @focus="handleFocus"
      @blur="handleBlur"
      @click="togglePinned"
      @keydown.esc.stop="close"
    >
      <span
        v-for="model in visibleModels"
        :key="`${model.platform}:${model.name}`"
        data-test="model-preview-pill"
        class="max-w-[150px] truncate rounded bg-gray-100 px-1.5 py-0.5 text-[11px] leading-4 text-gray-600 dark:bg-dark-600 dark:text-dark-300"
      >
        {{ model.name }}
      </span>
      <span
        v-if="hiddenModelCount > 0"
        data-test="model-hidden-count"
        class="rounded bg-gray-100 px-1.5 py-0.5 text-[11px] leading-4 text-gray-500 dark:bg-dark-600 dark:text-dark-400"
      >
        +{{ hiddenModelCount }}
      </span>
    </button>

    <Teleport to="body">
      <div
        v-if="open && models.length > 0"
        :id="popoverId"
        ref="popoverRef"
        data-test="key-models-popover"
        :style="popoverStyle"
        class="fixed z-50 max-h-64 overflow-y-auto rounded-xl border border-gray-200 bg-white p-3 shadow-lg dark:border-dark-600 dark:bg-dark-800"
        @mouseenter="handleMouseEnter"
        @mouseleave="handleMouseLeave"
      >
        <div
          v-for="group in groupedModels"
          :key="group.platform"
          class="mt-2 first:mt-0"
        >
          <p
            data-test="model-platform"
            class="mb-1 text-[10px] font-semibold uppercase tracking-wide text-gray-400 dark:text-dark-400"
          >
            {{ platformLabel(group.platform) }}
          </p>
          <div class="flex flex-wrap gap-1">
            <span
              v-for="model in group.models"
              :key="`${model.platform}:${model.name}`"
              data-test="model-popover-pill"
              class="max-w-full truncate rounded bg-gray-100 px-1.5 py-0.5 text-[11px] leading-4 text-gray-600 dark:bg-dark-600 dark:text-dark-300"
            >
              {{ model.name }}
            </span>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
