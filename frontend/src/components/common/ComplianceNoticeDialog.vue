<template>
  <Teleport to="body">
    <Transition name="compliance-fade">
      <div
        v-if="visible"
        class="fixed inset-0 z-[140] flex items-start justify-center overflow-y-auto bg-black/35 p-3 pt-[5vh] backdrop-blur-sm sm:p-5"
      >
        <section
          class="w-full max-w-[760px] overflow-hidden rounded-[24px] border border-emerald-200 bg-white text-gray-900 shadow-2xl"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
        >
          <header class="flex items-center gap-4 border-b border-emerald-200 bg-emerald-50 px-6 py-5 sm:px-7">
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-emerald-600 text-white shadow-lg shadow-emerald-700/20">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 3l7 4v5c0 4.5-2.8 7.6-7 9-4.2-1.4-7-4.5-7-9V7l7-4z" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v5" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 16h.01" />
              </svg>
            </div>
            <div class="min-w-0">
              <p class="text-sm font-medium text-emerald-700">{{ badge }}</p>
              <h2 class="mt-1 text-xl font-bold leading-tight sm:text-2xl">{{ title }}</h2>
            </div>
          </header>

          <div class="max-h-[58vh] overflow-y-auto bg-white px-6 py-6 sm:px-7">
            <div class="compliance-markdown" v-html="renderedContent"></div>
          </div>

          <footer class="flex flex-col-reverse gap-3 border-t border-gray-200 bg-white px-6 py-5 sm:flex-row sm:justify-end sm:px-7">
            <button
              type="button"
              class="min-h-11 rounded-full border border-emerald-600 bg-white px-6 text-sm font-semibold text-emerald-700 transition hover:bg-emerald-50 focus:outline-none focus:ring-2 focus:ring-emerald-600 focus:ring-offset-2"
              @click="$emit('decline')"
            >
              {{ declineText }}
            </button>
            <button
              type="button"
              class="min-h-11 rounded-full bg-emerald-600 px-7 text-sm font-semibold text-white shadow-lg shadow-emerald-700/20 transition hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-emerald-600 focus:ring-offset-2"
              @click="$emit('accept')"
            >
              {{ acceptText }}
            </button>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps<{
  visible: boolean
  badge: string
  title: string
  content: string
  acceptText: string
  declineText: string
}>()

defineEmits<{
  accept: []
  decline: []
}>()

marked.setOptions({
  breaks: true,
  gfm: true
})

const renderedContent = computed(() => {
  const html = marked.parse(props.content || '') as string
  return DOMPurify.sanitize(html)
})
</script>

<style scoped>
.compliance-fade-enter-active,
.compliance-fade-leave-active {
  transition: opacity 0.2s ease;
}

.compliance-fade-enter-from,
.compliance-fade-leave-to {
  opacity: 0;
}

.compliance-markdown :deep(p) {
  margin: 0 0 1rem;
  line-height: 1.8;
}

.compliance-markdown :deep(p:last-child) {
  margin-bottom: 0;
}
</style>
