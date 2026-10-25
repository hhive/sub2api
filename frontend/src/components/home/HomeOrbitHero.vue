<template>
  <!-- 着陆页主视觉：轨道地球 + 逐字标题 + 轮播文案 -->
  <section class="orbit-hero" data-testid="home-orbit-hero">
    <OrbitStage mode="hero" />

    <div class="orbit-core">
      <h1 class="orbit-title" data-testid="orbit-title" :aria-label="headlineLabel">
        <span class="orbit-title-prefix" aria-hidden="true">
          <span>{{ prefixVisible }}</span>
          <span v-if="typing" class="orbit-caret"></span>
        </span>
        <span class="orbit-title-verb" aria-hidden="true">
          <span>{{ verbVisible }}</span>
          <span v-if="verbTyping" class="orbit-caret"></span>
        </span>
      </h1>

      <p class="orbit-subtitle" data-testid="orbit-subtitle">{{ activeSlide.subtitle }}</p>

      <div class="orbit-actions">
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          data-testid="home-primary-cta"
          class="orbit-btn orbit-btn-primary"
        >
          {{ isAuthenticated ? t('home.hero.primaryCtaAuthed') : t('home.hero.primaryCta') }}
        </router-link>
        <router-link
          v-if="showModelPlazaEntry"
          to="/model-plaza"
          data-testid="home-secondary-cta"
          class="orbit-btn orbit-btn-ghost"
        >
          {{ t('home.hero.secondaryCta') }}
        </router-link>
      </div>

      <div v-if="carouselAvailable" class="orbit-carousel" data-testid="home-carousel">
        <button
          type="button"
          class="orbit-carousel-btn"
          data-testid="hero-carousel-prev"
          :aria-label="t('home.hero.carouselPrev')"
          :title="t('home.hero.carouselPrev')"
          @click="goTo(slideIndex - 1)"
        >
          <Icon name="chevronLeft" size="sm" />
        </button>
        <button
          v-if="autoRotating"
          type="button"
          class="orbit-carousel-btn"
          data-testid="hero-carousel-toggle"
          :aria-label="paused ? t('home.hero.carouselPlay') : t('home.hero.carouselPause')"
          :title="paused ? t('home.hero.carouselPlay') : t('home.hero.carouselPause')"
          @click="togglePause"
        >
          <Icon :name="paused ? 'play' : 'pause'" size="sm" />
        </button>
        <button
          type="button"
          class="orbit-carousel-btn"
          data-testid="hero-carousel-next"
          :aria-label="t('home.hero.carouselNext')"
          :title="t('home.hero.carouselNext')"
          @click="goTo(slideIndex + 1)"
        >
          <Icon name="chevronRight" size="sm" />
        </button>
      </div>

      <div v-if="contactInfo" class="orbit-links">
        <button
          type="button"
          class="orbit-text-btn"
          data-testid="home-contact-cta"
          @click="emit('contact-click')"
        >
          {{ t('home.hero.contact') }}
        </button>
      </div>
    </div>

    <Transition name="orbit-focus-fade" mode="out-in">
      <div :key="slideIndex" class="orbit-focus orbit-focus-left" data-testid="orbit-focus-left">
        <span class="orbit-focus-kicker">{{ activeSlide.left.kicker }}</span>
        <span class="orbit-focus-title">{{ activeSlide.left.title }}</span>
        <span class="orbit-focus-desc">{{ activeSlide.left.desc }}</span>
      </div>
    </Transition>

    <Transition name="orbit-focus-fade" mode="out-in">
      <div :key="slideIndex" class="orbit-focus orbit-focus-right" data-testid="orbit-focus-right">
        <span class="orbit-focus-kicker">{{ activeSlide.right.kicker }}</span>
        <span class="orbit-focus-title">{{ activeSlide.right.title }}</span>
        <span class="orbit-focus-desc">{{ activeSlide.right.desc }}</span>
      </div>
    </Transition>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import OrbitStage from './OrbitStage.vue'
import Icon from '@/components/icons/Icon.vue'
import { useTypewriter } from './useTypewriter'
import type { HeroSlide } from './types'

const props = withDefaults(
  defineProps<{
    slides: HeroSlide[]
    isAuthenticated: boolean
    dashboardPath: string
    showModelPlazaEntry: boolean
    contactInfo?: string
    intervalMs?: number
  }>(),
  { contactInfo: '', intervalMs: 6200 }
)

const emit = defineEmits<{ 'slide-change': [index: number]; 'contact-click': [] }>()

const { t } = useI18n()

const PREFIX_CHAR_MS = 55
const VERB_CHAR_MS = 85
const MIN_HOLD_MS = 1200

const EMPTY_SLIDE: HeroSlide = {
  prefix: '',
  verb: '',
  subtitle: '',
  left: { kicker: '', title: '', desc: '' },
  right: { kicker: '', title: '', desc: '' }
}

const slideIndex = ref(0)
const reduceMotion = ref(false)
const paused = ref(false)

const slideCount = computed(() => props.slides.length)
const carouselAvailable = computed(() => slideCount.value > 1)
const activeSlide = computed<HeroSlide>(() => props.slides[slideIndex.value] ?? EMPTY_SLIDE)
const headlineLabel = computed(() =>
  [activeSlide.value.prefix, activeSlide.value.verb].filter(Boolean).join(' ')
)

// 减弱动效时直接展示整句，自动推进同时停用
const animate = computed(() => !reduceMotion.value)
const autoRotating = computed(() => animate.value && carouselAvailable.value)

const prefixState = useTypewriter(() => activeSlide.value.prefix, {
  charDelayMs: PREFIX_CHAR_MS,
  enabled: animate
})
const prefixVisible = prefixState.visible
const prefixDone = prefixState.done

const verbState = useTypewriter(() => (prefixDone.value ? activeSlide.value.verb : ''), {
  charDelayMs: VERB_CHAR_MS,
  enabled: animate
})
const verbVisible = verbState.visible
const verbDone = verbState.done

const typing = computed(() => animate.value && !prefixDone.value)
const verbTyping = computed(() => animate.value && prefixDone.value && !verbDone.value)

let advanceTimer: ReturnType<typeof setTimeout> | null = null
let advanceDeadline = 0
let pendingRemaining = 0

function clearAdvance(): void {
  if (advanceTimer === null) return
  clearTimeout(advanceTimer)
  advanceTimer = null
}

// 一轮总时长扣掉打字耗时，留出读文案的时间
function holdDuration(): number {
  const typingBudget =
    activeSlide.value.prefix.length * PREFIX_CHAR_MS + activeSlide.value.verb.length * VERB_CHAR_MS
  return Math.max(MIN_HOLD_MS, props.intervalMs - typingBudget)
}

function scheduleAdvance(delay = holdDuration()): void {
  clearAdvance()
  pendingRemaining = 0
  if (!autoRotating.value || paused.value) return
  advanceDeadline = Date.now() + delay
  advanceTimer = setTimeout(() => {
    advanceTimer = null
    goNext()
  }, delay)
}

function goNext(): void {
  if (slideCount.value === 0) return
  slideIndex.value = (slideIndex.value + 1) % slideCount.value
}

// 手动切换一组文案，环形取模；未暂停时自动推进由动词收尾后的排程重新接管
function goTo(index: number): void {
  if (slideCount.value === 0) return
  clearAdvance()
  slideIndex.value = ((index % slideCount.value) + slideCount.value) % slideCount.value
}

function togglePause(): void {
  paused.value = !paused.value
  if (paused.value) {
    clearAdvance()
    return
  }
  if (prefixDone.value && verbDone.value) scheduleAdvance()
}

function onVisibilityChange(): void {
  if (document.visibilityState === 'hidden') {
    if (advanceTimer !== null) pendingRemaining = Math.max(0, advanceDeadline - Date.now())
    clearAdvance()
    return
  }
  if (pendingRemaining <= 0) return
  const remaining = pendingRemaining
  pendingRemaining = 0
  scheduleAdvance(remaining)
}

watch(verbDone, (done) => {
  if (done && prefixDone.value) scheduleAdvance()
})

watch(slideIndex, (index) => {
  emit('slide-change', index)
})

watch(slideCount, (count) => {
  if (count === 0) return
  if (slideIndex.value >= count) slideIndex.value = 0
})

onMounted(() => {
  reduceMotion.value =
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches

  document.addEventListener('visibilitychange', onVisibilityChange)
  emit('slide-change', slideIndex.value)
})

onBeforeUnmount(() => {
  clearAdvance()
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<style scoped>
.orbit-hero {
  position: relative;
  isolation: isolate;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  min-height: 100vh;
  background: #0b0d0e;
  color: #f5f4f9;
  color-scheme: dark;
}

.orbit-core {
  position: relative;
  z-index: 4;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: min(540px, 86vw);
  text-align: center;
  pointer-events: none;
}

.orbit-core a,
.orbit-core button {
  pointer-events: auto;
}

/* ---------- 标题 ---------- */
.orbit-title {
  display: grid;
  gap: 4px;
  margin: 0;
  font-family: Lexend, 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif;
  color: #f1f4f2;
  text-shadow:
    0 2px 34px rgba(0, 0, 0, 0.94),
    0 0 90px rgba(0, 0, 0, 0.72);
}

.orbit-title-prefix {
  display: block;
  font-size: clamp(1.85rem, 2.9vw, 2.6rem);
  font-weight: 600;
  line-height: 1.19;
  letter-spacing: -0.02em;
}

.orbit-title-verb {
  display: block;
  min-height: 1.1em;
  font-size: clamp(2.4rem, 4.4vw, 4rem);
  font-weight: 700;
  line-height: 1.08;
  letter-spacing: -0.02em;
  color: #b9f0ec;
}

.orbit-caret {
  display: inline-block;
  width: 0.06em;
  height: 0.86em;
  margin-left: 0.06em;
  vertical-align: -0.06em;
  background: currentColor;
  animation: orbit-caret 1s steps(2, start) infinite;
}

@keyframes orbit-caret {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0;
  }
}

.orbit-subtitle {
  margin-top: 18px;
  max-width: 24rem;
  white-space: pre-line;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 14px;
  line-height: 1.71;
  color: #b7bdc5;
  text-shadow: 0 1px 20px rgba(0, 0, 0, 0.94);
}

/* ---------- 轮播控件 ---------- */
.orbit-carousel {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 18px;
}

.orbit-carousel-btn {
  display: inline-flex;
  height: 30px;
  width: 30px;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: rgba(11, 13, 14, 0.42);
  color: #b7bdc5;
  cursor: pointer;
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.orbit-carousel-btn:hover {
  color: #f1f4f2;
  border-color: rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.08);
}

/* ---------- 按钮 ---------- */
.orbit-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 26px;
}

.orbit-btn {
  display: inline-flex;
  min-width: 132px;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  border-radius: 2px;
  padding: 0 22px;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 13px;
  font-weight: 500;
  transition:
    background-color 0.2s ease,
    border-color 0.2s ease,
    color 0.2s ease;
}

.orbit-btn-primary {
  background: #1c2228;
  border: 1px solid #657782;
  color: #d6e9fb;
  box-shadow: inset 0 1px 0 rgba(127, 187, 239, 0.12);
}

.orbit-btn-primary:hover {
  background: #29333e;
}

.orbit-btn-ghost {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #b7bdc5;
}

.orbit-btn-ghost:hover {
  border-color: rgba(255, 255, 255, 0.3);
  color: #f1f4f2;
}

.orbit-links {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 14px;
}

.orbit-text-btn {
  border: 0;
  background: transparent;
  padding: 3px 8px;
  border-radius: 10px;
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 12px;
  color: #b7bdc5;
  text-decoration: none;
  cursor: pointer;
  transition:
    color 0.2s ease,
    background-color 0.2s ease;
}

.orbit-text-btn:hover {
  color: #f1f4f2;
  background: rgba(255, 255, 255, 0.06);
}

/* ---------- 左右焦点卡 ---------- */
.orbit-focus {
  position: absolute;
  top: 50%;
  z-index: 2;
  display: flex;
  width: 190px;
  flex-direction: column;
  gap: 10px;
  transform: translateY(-50%);
  text-align: center;
  pointer-events: none;
}

.orbit-focus-left {
  left: 6.5%;
}

.orbit-focus-right {
  right: 6.5%;
}

.orbit-focus-kicker {
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.5);
}

.orbit-focus-title {
  font-family: Lexend, system-ui, -apple-system, sans-serif;
  font-size: 18px;
  font-weight: 600;
  line-height: 1.35;
  color: #d1e5f8;
  text-shadow: 0 1px 20px rgba(0, 0, 0, 0.9);
}

.orbit-focus-desc {
  font-size: 13px;
  line-height: 1.85;
  color: rgba(255, 255, 255, 0.62);
  text-shadow: 0 1px 18px rgba(0, 0, 0, 0.9);
}

.orbit-focus-fade-enter-active,
.orbit-focus-fade-leave-active {
  transition:
    opacity 0.32s ease,
    transform 0.32s ease;
}

.orbit-focus-fade-enter-from,
.orbit-focus-fade-leave-to {
  opacity: 0;
  transform: translateY(calc(-50% + 6px));
}

/* 侧卡只在中心文案之外有位置时出现 */
@media (max-width: 1199px) {
  .orbit-focus {
    display: none;
  }
}

@media (max-width: 640px) {
  .orbit-hero {
    min-height: 100svh;
  }

  .orbit-subtitle {
    max-width: 20rem;
  }

  .orbit-actions {
    gap: 10px;
  }

  .orbit-btn {
    min-width: 108px;
    min-height: 42px;
    padding: 0 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .orbit-caret {
    animation: none;
  }

  .orbit-focus-fade-enter-active,
  .orbit-focus-fade-leave-active {
    transition: none;
  }
}
</style>
