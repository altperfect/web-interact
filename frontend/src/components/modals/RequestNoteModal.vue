<template>
  <div class="modal-backdrop" @click="$emit('close')">
    <section class="telegram-modal compact-modal" role="dialog" aria-modal="true" aria-labelledby="request-note-title" @click.stop>
      <header class="modal-head">
        <div>
          <p class="overline">Request</p>
          <h2 id="request-note-title">Request note</h2>
        </div>
      </header>

      <form class="telegram-form" @submit.prevent="$emit('save')">
        <label class="field">
          <span>Note</span>
          <textarea
            v-model="noteModel"
            rows="5"
            :maxlength="maxSymbols"
          ></textarea>
          <small class="field-hint">{{ remaining }}/{{ maxSymbols }} symbols remaining</small>
        </label>

        <div class="modal-actions">
          <button class="small-action primary-action" type="submit" :disabled="saving">
            Save note
          </button>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { symbolCount } from '../../utils/formatters'

const props = defineProps({
  note: { type: String, default: '' },
  maxSymbols: { type: Number, required: true },
  saving: { type: Boolean, default: false }
})

const emit = defineEmits(['update:note', 'close', 'save'])

const noteModel = computed({
  get: () => props.note,
  set: (value) => emit('update:note', value)
})

const remaining = computed(() => Math.max(0, props.maxSymbols - symbolCount(props.note)))
</script>
