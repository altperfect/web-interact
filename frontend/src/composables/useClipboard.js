import { onBeforeUnmount, ref } from 'vue'

export function useClipboard(showNotice) {
  const copiedTarget = ref('')
  let copiedTimer = null

  async function copy(value, message) {
    if (!value) return
    try {
      await navigator.clipboard.writeText(value)
      markCopied(message)
    } catch {
      showNotice('Copy failed')
    }
  }

  function markCopied(target) {
    copiedTarget.value = target
    if (copiedTimer) window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
      copiedTarget.value = ''
    }, 1400)
  }

  function cleanupClipboard() {
    if (copiedTimer) window.clearTimeout(copiedTimer)
  }

  onBeforeUnmount(cleanupClipboard)

  return {
    copiedTarget,
    copy,
    markCopied,
    cleanupClipboard
  }
}
