import { ref } from 'vue'
import { clamp } from '../utils/formatters'

export function useRequestNoteTooltip() {
  const requestNoteTooltip = ref({ visible: false, text: '', left: 0, top: 0 })

  function showRequestNoteTooltip(event, note) {
    if (!note) return
    const rect = event.currentTarget.getBoundingClientRect()
    requestNoteTooltip.value = {
      visible: true,
      text: note,
      left: clamp(rect.left + rect.width / 2, 24, window.innerWidth - 24),
      top: Math.max(14, rect.top - 8)
    }
  }

  function positionRequestNoteTooltip(event) {
    if (!requestNoteTooltip.value.visible) return
    const rect = event.currentTarget.getBoundingClientRect()
    requestNoteTooltip.value = {
      ...requestNoteTooltip.value,
      left: clamp(rect.left + rect.width / 2, 24, window.innerWidth - 24),
      top: Math.max(14, rect.top - 8)
    }
  }

  function hideRequestNoteTooltip() {
    if (!requestNoteTooltip.value.visible) return
    requestNoteTooltip.value = { visible: false, text: '', left: 0, top: 0 }
  }

  return {
    requestNoteTooltip,
    showRequestNoteTooltip,
    positionRequestNoteTooltip,
    hideRequestNoteTooltip
  }
}
