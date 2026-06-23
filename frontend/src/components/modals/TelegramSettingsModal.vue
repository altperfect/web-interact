<template>
  <div class="modal-backdrop" @click="$emit('close')">
    <section class="telegram-modal" role="dialog" aria-modal="true" aria-labelledby="telegram-settings-title" @click.stop>
      <header class="modal-head">
        <div>
          <p class="overline">Notifications</p>
          <h2 id="telegram-settings-title">Telegram settings</h2>
        </div>
      </header>

      <form class="telegram-form" @submit.prevent="$emit('save')">
        <div class="form-grid">
          <label class="field">
            <span>Bot token</span>
            <input
              v-model.trim="form.botToken"
              type="password"
              autocomplete="off"
              :required="!settings.configured"
              :placeholder="settings.configured ? 'Configured' : '123456:ABC-DEF'"
            />
          </label>

          <label class="field">
            <span>Chat ID</span>
            <input v-model.trim="form.chatId" type="text" required autocomplete="off" placeholder="-1001234567890" />
          </label>
        </div>

        <section class="proxy-settings">
          <label class="toggle-field">
            <input v-model="form.proxyEnabled" type="checkbox" />
            <span>Use SOCKS5 proxy</span>
          </label>

          <div v-if="form.proxyEnabled" class="form-grid proxy-grid">
            <label class="field">
              <span>Hostname or IP</span>
              <input v-model.trim="form.proxyHost" type="text" required autocomplete="off" placeholder="127.0.0.1" />
            </label>

            <label class="field">
              <span>Port</span>
              <input v-model.number="form.proxyPort" type="number" min="1" max="65535" required placeholder="9050" />
            </label>

            <label class="field">
              <span>Username</span>
              <input v-model.trim="form.proxyUsername" type="text" autocomplete="off" />
            </label>

            <label class="field">
              <span>Password</span>
              <input
                v-model="form.proxyPassword"
                type="password"
                autocomplete="off"
                :placeholder="settings.proxyPasswordConfigured ? 'Configured' : ''"
              />
            </label>
          </div>
        </section>

        <div class="modal-actions">
          <button class="small-action" type="button" :disabled="testing || saving" @click="$emit('test')">
            Test notification
          </button>
          <button class="small-action primary-action" type="submit" :disabled="saving || testing">
            Save settings
          </button>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup>
defineProps({
  settings: { type: Object, required: true },
  form: { type: Object, required: true },
  saving: { type: Boolean, default: false },
  testing: { type: Boolean, default: false }
})

defineEmits(['close', 'save', 'test'])
</script>
