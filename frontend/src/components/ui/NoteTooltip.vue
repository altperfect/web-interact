<template>
  <Teleport to="body">
    <div
      v-if="tooltip.visible"
      class="note-tooltip"
      :style="{ left: `${tooltip.left}px`, top: `${tooltip.top}px` }"
      role="tooltip"
    >
      {{ tooltip.text }}
    </div>
  </Teleport>
</template>

<script setup>
defineProps({
  tooltip: { type: Object, required: true }
})
</script>
