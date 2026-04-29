<template>
  <AppLayout>
    <div class="mx-auto flex h-[calc(100vh-7rem)] max-w-5xl flex-col rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
        <div>
          <h1 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('chat.title') }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('chat.subtitle') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div class="flex rounded-lg bg-gray-100 p-1 dark:bg-gray-700">
            <button
              class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="mode === 'chat' ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-800 dark:text-white' : 'text-gray-600 dark:text-gray-300'"
              type="button"
              @click="mode = 'chat'"
            >
              {{ t('chat.modes.chat') }}
            </button>
            <button
              class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="mode === 'image' ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-800 dark:text-white' : 'text-gray-600 dark:text-gray-300'"
              type="button"
              @click="mode = 'image'"
            >
              {{ t('chat.modes.image') }}
            </button>
          </div>
          <select v-if="mode === 'chat'" v-model="selectedChatModel" class="input min-w-48" :disabled="loadingChatModels || sending">
            <option v-for="model in chatModels" :key="model.id" :value="model.id">
              {{ model.name }} · {{ model.provider }}
            </option>
          </select>
          <select v-else v-model="selectedImageModel" class="input min-w-48" :disabled="loadingImageModels || generatingImage">
            <option v-for="model in imageModels" :key="model.id" :value="model.id">
              {{ model.name }} · {{ model.provider }}
            </option>
          </select>
          <button class="btn btn-secondary" :disabled="busy" @click="resetConsole">
            {{ t('chat.newChat') }}
          </button>
        </div>
      </div>

      <div ref="messagesEl" class="flex-1 space-y-4 overflow-y-auto p-4">
        <template v-if="mode === 'chat'">
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
        </template>

        <template v-else>
          <div v-if="imageResults.length === 0 && !generatingImage" class="flex h-full items-center justify-center text-center">
            <div class="max-w-md space-y-2">
              <div class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('chat.modes.image') }}</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('chat.imagePromptPlaceholder') }}</div>
            </div>
          </div>

          <div v-if="imageResults.length > 0" class="grid gap-4 sm:grid-cols-2">
            <div v-for="(image, index) in imageResults" :key="index" class="overflow-hidden rounded-xl border border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900">
              <img v-if="imageSrc(image)" :src="imageSrc(image)" class="h-auto w-full" :alt="image.revised_prompt || imagePrompt" />
              <div v-if="image.revised_prompt" class="px-3 py-2 text-sm text-gray-600 dark:text-gray-300">
                {{ image.revised_prompt }}
              </div>
            </div>
          </div>

          <div v-if="generatingImage" class="flex justify-start">
            <div class="rounded-2xl bg-gray-100 px-4 py-3 text-sm text-gray-500 dark:bg-gray-700 dark:text-gray-300">
              {{ t('chat.generatingImage') }}
            </div>
          </div>
        </template>
      </div>

      <form v-if="mode === 'chat'" class="border-t border-gray-200 p-4 dark:border-gray-700" @submit.prevent="sendMessage">
        <div v-if="error" class="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300">
          {{ error }}
        </div>
        <div class="flex gap-3">
          <textarea
            v-model="draft"
            rows="2"
            class="input min-h-[3rem] flex-1 resize-none"
            :placeholder="t('chat.inputPlaceholder')"
            :disabled="sending || loadingChatModels"
            @keydown.enter.exact.prevent="sendMessage"
          />
          <button class="btn btn-primary self-end" type="submit" :disabled="!canSendMessage">
            {{ sending ? t('chat.sending') : t('chat.send') }}
          </button>
        </div>
      </form>

      <form v-else class="border-t border-gray-200 p-4 dark:border-gray-700" @submit.prevent="generateImage">
        <div v-if="error" class="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300">
          {{ error }}
        </div>
        <div class="flex gap-3">
          <textarea
            v-model="imagePrompt"
            rows="2"
            class="input min-h-[3rem] flex-1 resize-none"
            :placeholder="t('chat.imagePromptPlaceholder')"
            :disabled="generatingImage || loadingImageModels"
            @keydown.enter.exact.prevent="generateImage"
          />
          <button class="btn btn-primary self-end" type="submit" :disabled="!canGenerateImage">
            {{ generatingImage ? t('chat.generatingImage') : t('chat.generateImage') }}
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getImageGenerationModels, getTextChatModels, sendChatCompletion, sendImageGeneration } from '@/api/chat'
import AppLayout from '@/components/layout/AppLayout.vue'
import type { ChatConsoleMode, ChatMessage, ChatModel, ImageGenerationData } from '@/types/chat'

const { t } = useI18n()
const mode = ref<ChatConsoleMode>('chat')
const chatModels = ref<ChatModel[]>([])
const imageModels = ref<ChatModel[]>([])
const selectedChatModel = ref('')
const selectedImageModel = ref('')
const messages = ref<ChatMessage[]>([])
const draft = ref('')
const imagePrompt = ref('')
const imageResults = ref<ImageGenerationData[]>([])
const error = ref('')
const sending = ref(false)
const generatingImage = ref(false)
const loadingChatModels = ref(false)
const loadingImageModels = ref(false)
const messagesEl = ref<HTMLElement | null>(null)

const busy = computed(() => sending.value || generatingImage.value)
const canSendMessage = computed(() => draft.value.trim() !== '' && selectedChatModel.value !== '' && !sending.value && !loadingChatModels.value)
const canGenerateImage = computed(() => imagePrompt.value.trim() !== '' && selectedImageModel.value !== '' && !generatingImage.value && !loadingImageModels.value)

async function scrollToBottom() {
  await nextTick()
  if (messagesEl.value) {
    messagesEl.value.scrollTop = messagesEl.value.scrollHeight
  }
}

async function loadChatModels() {
  loadingChatModels.value = true
  error.value = ''
  try {
    chatModels.value = await getTextChatModels()
    selectedChatModel.value = chatModels.value[0]?.id || ''
  } catch (err) {
    console.error('Failed to load chat models:', err)
    error.value = t('chat.loadModelsFailed')
  } finally {
    loadingChatModels.value = false
  }
}

async function loadImageModels() {
  loadingImageModels.value = true
  error.value = ''
  try {
    imageModels.value = await getImageGenerationModels()
    selectedImageModel.value = imageModels.value[0]?.id || ''
  } catch (err) {
    console.error('Failed to load image models:', err)
    error.value = t('chat.imageModelsLoadFailed')
  } finally {
    loadingImageModels.value = false
  }
}

async function sendMessage() {
  const content = draft.value.trim()
  if (!content || !selectedChatModel.value || sending.value) return

  error.value = ''
  draft.value = ''
  messages.value.push({ role: 'user', content })
  await scrollToBottom()

  sending.value = true
  try {
    const completion = await sendChatCompletion({
      model: selectedChatModel.value,
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

async function generateImage() {
  const prompt = imagePrompt.value.trim()
  if (!prompt || !selectedImageModel.value || generatingImage.value) return

  error.value = ''
  generatingImage.value = true
  try {
    const response = await sendImageGeneration({
      model: selectedImageModel.value,
      prompt,
      response_format: 'url',
    })
    imageResults.value = response.data || []
    if (imageResults.value.length === 0) {
      error.value = t('chat.noImageResult')
    }
  } catch (err) {
    console.error('Failed to generate image:', err)
    error.value = t('chat.imageGenerationFailed')
  } finally {
    generatingImage.value = false
    await scrollToBottom()
  }
}

function imageSrc(image: ImageGenerationData) {
  if (image.url) return image.url
  if (image.b64_json) return `data:image/png;base64,${image.b64_json}`
  return ''
}

function resetConsole() {
  messages.value = []
  draft.value = ''
  imagePrompt.value = ''
  imageResults.value = []
  error.value = ''
}

watch(mode, async value => {
  error.value = ''
  if (value === 'image' && imageModels.value.length === 0) {
    await loadImageModels()
  }
})

onMounted(() => {
  loadChatModels()
})
</script>
