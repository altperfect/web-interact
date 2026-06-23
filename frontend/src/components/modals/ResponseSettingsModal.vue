<template>
  <div class="modal-backdrop" @click="$emit('close')">
    <section class="telegram-modal response-modal" role="dialog" aria-modal="true" aria-labelledby="response-settings-title" @click.stop>
      <header class="modal-head">
        <div>
          <p class="overline">Webhook response</p>
          <h2 id="response-settings-title">{{ webhookName }}</h2>
        </div>
      </header>

      <form class="telegram-form response-form" @submit.prevent="$emit('save')">
        <div class="form-grid response-grid">
          <label class="field">
            <span>Status code</span>
            <input
              v-model.number="form.statusCode"
              type="number"
              min="100"
              max="599"
              required
            />
          </label>

          <label class="field">
            <span>Content-Type</span>
            <input
              v-model.trim="form.contentType"
              type="text"
              required
              autocomplete="off"
              placeholder="text/plain; charset=utf-8"
            />
          </label>
        </div>

        <label v-if="isRedirect" class="field">
          <span>Location</span>
          <input
            v-model.trim="form.location"
            type="text"
            required
            autocomplete="off"
            placeholder="https://example.com/next"
          />
        </label>

        <section class="response-headers">
          <div class="response-headers-head">
            <span>Response headers</span>
            <button class="small-action icon-only" type="button" title="Add response header" @click="$emit('add-header')">
              <Plus :size="15" :stroke-width="1.8" aria-hidden="true" />
            </button>
          </div>

          <div
            v-for="header in form.headers"
            :key="header.id"
            class="response-header-row"
          >
            <label class="field">
              <span>Name</span>
              <input v-model.trim="header.name" type="text" autocomplete="off" placeholder="X-Webhook" />
            </label>
            <label class="field">
              <span>Value</span>
              <input v-model="header.value" type="text" autocomplete="off" placeholder="debug" />
            </label>
            <button class="small-action icon-only danger-action" type="button" title="Remove response header" @click="$emit('remove-header', header.id)">
              <Trash2 :size="15" :stroke-width="1.85" aria-hidden="true" />
            </button>
          </div>
        </section>

        <label class="field">
          <span>Response body</span>
          <textarea
            v-model="form.body"
            rows="10"
            spellcheck="false"
          ></textarea>
        </label>

        <div class="modal-actions">
          <button class="small-action primary-action" type="submit" :disabled="saving">
            Save response
          </button>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup>
import { Plus, Trash2 } from '@lucide/vue'

defineProps({
  webhookName: { type: String, default: '' },
  form: { type: Object, required: true },
  isRedirect: { type: Boolean, default: false },
  saving: { type: Boolean, default: false }
})

defineEmits(['close', 'save', 'add-header', 'remove-header'])
</script>
