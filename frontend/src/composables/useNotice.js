import { onBeforeUnmount, ref } from 'vue'

export function useNotice() {
  const notice = ref('')
  let noticeTimer = null

  function showNotice(message) {
    notice.value = message
    if (noticeTimer) window.clearTimeout(noticeTimer)
    noticeTimer = window.setTimeout(() => {
      notice.value = ''
    }, 2500)
  }

  function cleanupNotice() {
    if (noticeTimer) window.clearTimeout(noticeTimer)
  }

  onBeforeUnmount(cleanupNotice)

  return {
    notice,
    showNotice,
    cleanupNotice
  }
}
