import { onScopeDispose, ref, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue'

export interface TypewriterOptions {
  /** 每个字符的间隔毫秒数 */
  charDelayMs?: number
  /** 关闭时直接展示全文并立即完成 */
  enabled?: MaybeRefOrGetter<boolean>
}

export interface TypewriterState {
  /** 当前已显示的字符片段 */
  visible: Ref<string>
  /** 全文是否已显示完毕 */
  done: Ref<boolean>
  /** 从空串重新开始 */
  restart: () => void
  /** 停止推进，保留当前进度 */
  stop: () => void
}

const DEFAULT_CHAR_DELAY_MS = 70

function isDocumentHidden(): boolean {
  return typeof document !== 'undefined' && document.visibilityState === 'hidden'
}

/**
 * 逐字显示一段文本。文本或开关变化时自动复位重放；
 * 页面不可见时暂停推进，恢复可见后从当前进度继续。
 */
export function useTypewriter(
  text: MaybeRefOrGetter<string>,
  options: TypewriterOptions = {}
): TypewriterState {
  const charDelayMs = options.charDelayMs ?? DEFAULT_CHAR_DELAY_MS
  const enabled = options.enabled ?? true

  const visible = ref('')
  const done = ref(false)

  let timer: ReturnType<typeof setTimeout> | null = null
  let target = ''
  let paused = false

  function clearTimer(): void {
    if (timer === null) return
    clearTimeout(timer)
    timer = null
  }

  function schedule(): void {
    if (paused || timer !== null) return
    timer = setTimeout(step, charDelayMs)
  }

  function finish(): void {
    clearTimer()
    visible.value = target
    done.value = true
  }

  function step(): void {
    timer = null
    if (paused) return

    if (visible.value.length >= target.length) {
      done.value = true
      return
    }

    visible.value = target.slice(0, visible.value.length + 1)
    if (visible.value.length >= target.length) {
      done.value = true
      return
    }
    schedule()
  }

  function restart(): void {
    clearTimer()
    target = toValue(text) ?? ''
    visible.value = ''
    done.value = false
    paused = isDocumentHidden()

    if (!toValue(enabled) || target.length === 0) {
      finish()
      return
    }

    schedule()
  }

  function stop(): void {
    clearTimer()
  }

  function onVisibilityChange(): void {
    if (isDocumentHidden()) {
      clearTimer()
      paused = true
      return
    }
    if (!paused) return
    paused = false
    if (!done.value) schedule()
  }

  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisibilityChange)
  }

  watch([() => toValue(text), () => toValue(enabled)], restart, { immediate: true })

  onScopeDispose(() => {
    clearTimer()
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  })

  return { visible, done, restart, stop }
}
