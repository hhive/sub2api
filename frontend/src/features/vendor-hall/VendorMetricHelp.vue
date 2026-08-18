<template>
  <span class="vendor-metric-help">
    <button
      ref="trigger"
      type="button"
      class="vendor-metric-help__trigger"
      :aria-label="label"
      :aria-expanded="open"
      @click.stop="toggle"
      @keydown.esc.stop="close"
    >
      <Icon name="exclamationCircle" size="xs" />
    </button>
    <Teleport to="body">
      <div
        v-if="open"
        ref="popover"
        class="vendor-metric-help__popover"
        role="dialog"
        :aria-label="label"
        :style="popoverStyle"
      >
        <strong>{{ label }}</strong>
        <p>{{ description }}</p>
      </div>
    </Teleport>
  </span>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{ label: string; description: string }>()

const trigger = ref<HTMLElement | null>(null)
const popover = ref<HTMLElement | null>(null)
const open = ref(false)
const position = ref({ left: 12, top: 12 })
const popoverStyle = computed(() => ({ left: `${position.value.left}px`, top: `${position.value.top}px` }))

const updatePosition = () => {
  const element = trigger.value
  if (!element) return
  const rect = element.getBoundingClientRect()
  const width = Math.min(260, window.innerWidth - 24)
  const left = Math.min(Math.max(12, rect.left), window.innerWidth - width - 12)
  const estimatedHeight = 118
  const top = rect.bottom + 8 + estimatedHeight <= window.innerHeight
    ? rect.bottom + 8
    : Math.max(12, rect.top - estimatedHeight - 8)
  position.value = { left, top }
}

const onOutsideClick = (event: MouseEvent) => {
  const target = event.target as Node
  if (!trigger.value?.contains(target) && !popover.value?.contains(target)) close()
}

const addListeners = () => {
  document.addEventListener('click', onOutsideClick)
  window.addEventListener('resize', updatePosition)
  window.addEventListener('scroll', updatePosition, true)
}

const removeListeners = () => {
  document.removeEventListener('click', onOutsideClick)
  window.removeEventListener('resize', updatePosition)
  window.removeEventListener('scroll', updatePosition, true)
}

const close = () => {
  open.value = false
  removeListeners()
}

const toggle = async () => {
  if (open.value) {
    close()
    return
  }
  open.value = true
  await nextTick()
  updatePosition()
  addListeners()
}

onBeforeUnmount(removeListeners)
</script>

<style scoped>
.vendor-metric-help { display: inline-flex; align-items: center; }
.vendor-metric-help__trigger { display: grid; width: 20px; height: 20px; flex: none; place-items: center; border-radius: 50%; color: #9ca3af; }
.vendor-metric-help__trigger:hover, .vendor-metric-help__trigger:focus-visible { background: #e5e7eb; color: #2563eb; outline: none; }
.vendor-metric-help__popover { position: fixed; z-index: 1000; width: min(260px, calc(100vw - 24px)); border: 1px solid #d1d5db; border-radius: 6px; background: white; padding: 12px 14px; color: #374151; box-shadow: 0 10px 28px rgb(15 23 42 / 18%); }
.vendor-metric-help__popover strong { display: block; color: #111827; font-size: 12px; }
.vendor-metric-help__popover p { margin-top: 5px; font-size: 12px; line-height: 1.55; }
:global(.dark .vendor-metric-help__trigger:hover), :global(.dark .vendor-metric-help__trigger:focus-visible) { background: #374151; color: #60a5fa; }
:global(.dark .vendor-metric-help__popover) { border-color: #4b5563; background: #1f2937; color: #d1d5db; }
:global(.dark .vendor-metric-help__popover strong) { color: #f9fafb; }
</style>
