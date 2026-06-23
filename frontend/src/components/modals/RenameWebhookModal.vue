<template>
  <div class="modal-backdrop" @click="$emit('close')">
    <section class="telegram-modal compact-modal" role="dialog" aria-modal="true" aria-labelledby="rename-webhook-title" @click.stop>
      <header class="modal-head">
        <div>
          <p class="overline">Webhook</p>
          <h2 id="rename-webhook-title">Rename webhook</h2>
        </div>
      </header>

      <form class="telegram-form" @submit.prevent="$emit('save')">
        <label class="field">
          <span>Name</span>
          <input
            v-model="nameModel"
            type="text"
            autocomplete="off"
            required
            :maxlength="maxSymbols"
          />
        </label>

        <div class="modal-actions">
          <button class="small-action primary-action" type="submit" :disabled="saving">
            Save name
          </button>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, default: '' },
  maxSymbols: { type: Number, required: true },
  saving: { type: Boolean, default: false }
})

const emit = defineEmits(['update:name', 'close', 'save'])

const nameModel = computed({
  get: () => props.name,
  set: (value) => emit('update:name', value)
})
</script>
