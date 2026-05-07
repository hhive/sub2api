<template>
  <AppLayout>
    <div class="mx-auto flex h-[calc(100vh-7rem)] max-w-5xl flex-col rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
        <div>
          <h1 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('chat.title') }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('chat.subtitle') }}</p>
        </div>
        <div class="flex items-center gap-2">
          <select v-model="selectedModel" class="input min-w-48" :disabled="loadingModels || sending">
            <option v-for="model in models" :key="model.id" :value="model.id">
              {{ model.name }} · {{ model.provider }}
            </option>
          </select>
          <button class="btn btn-secondary" :disabled="sending" @click="resetChat">
            {{ t('chat.newChat') }}
          </button>
        </div>
      </div>

      <div ref="messagesEl" class="flex-1 space-y-4 overflow-y-auto p-4">
        <div v-if="messages.length === 0" class="flex h-full items-center justify-center text-center">
          <div class="max-w-md space-y-2">
            <div class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('chat.emptyTitle') }}</div>
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('chat.emptyDescription') }}</div>
          </div>
        </div>

        <div v-for="(message, index) in messages" :key="index" class="flex" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
          <div class="max-w-[85%] whitespace-pre-wrap rounded-2xl px-4 py-3 text-sm leading-6" :class="message.role === 'user' ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100'">
            {{ message.content }}
          </div>
        </div>

        <div v-if="sending" class="flex justify-start">
          <div class="rounded-2xl bg-gray-100 px-4 py-3 text-sm text-gray-500 dark:bg-gray-700 dark:text-gray-300">
            {{ t('chat.thinking') }}
          </div>
        </div>
      </div>

      <form class="border-t border-gray-200 p-4 dark:border-gray-700" @submit.prevent="sendMessage">
        <div v-if="error" class="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300">
          {{ error }}
        </div>
        <div class="flex gap-3">
          <textarea
            v-model="draft"
            rows="2"
            class="input min-h-[3rem] flex-1 resize-none"
            :placeholder="t('chat.inputPlaceholder')"
            :disabled="sending || loadingModels"
            @keydown.enter.exact.prevent="sendMessage"
          />
          <button class="btn btn-primary self-end" type="submit" :disabled="!canSend">
            {{ sending ? t('chat.sending') : t('chat.send') }}
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getChatModels, sendChatCompletion } from '@/api/chat'
import AppLayout from '@/components/layout/AppLayout.vue'
import type { ChatMessage, ChatModel } from '@/types/chat'

const { t } = useI18n()
const models = ref<ChatModel[]>([])
const selectedModel = ref('')
const messages = ref<ChatMessage[]>([])
const draft = ref('')
const error = ref('')
const sending = ref(false)
const loadingModels = ref(false)
const messagesEl = ref<HTMLElement | null>(null)

const canSend = computed(() => draft.value.trim() !== '' && selectedModel.value !== '' && !sending.value && !loadingModels.value)

async function scrollToBottom() {
  await nextTick()
  if (messagesEl.value) {
    messagesEl.value.scrollTop = messagesEl.value.scrollHeight
  }
}

async function loadModels() {
  loadingModels.value = true
  error.value = ''
  try {
    models.value = await getChatModels()
    selectedModel.value = models.value[0]?.id || ''
  } catch (err) {
    console.error('Failed to load chat models:', err)
    error.value = t('chat.loadModelsFailed')
  } finally {
    loadingModels.value = false
  }
}

async function sendMessage() {
  const content = draft.value.trim()
  if (!content || !selectedModel.value || sending.value) return

  error.value = ''
  draft.value = ''
  messages.value.push({ role: 'user', content })
  await scrollToBottom()

  sending.value = true
  try {
    const completion = await sendChatCompletion({
      model: selectedModel.value,
      messages: messages.value,
      stream: false,
    })
    const reply = completion.choices?.[0]?.message?.content || ''
    messages.value.push({ role: 'assistant', content: reply || t('chat.emptyReply') })
  } catch (err) {
    console.error('Failed to send chat message:', err)
    error.value = t('chat.sendFailed')
  } finally {
    sending.value = false
    await scrollToBottom()
  }
}

function resetChat() {
  messages.value = []
  draft.value = ''
  error.value = ''
}

onMounted(() => {
  loadModels()
})
</script>
