<template>
  <div class="modal-backdrop search-backdrop" @click="$emit('close')">
    <section class="search-modal" role="dialog" aria-modal="true" aria-labelledby="request-search-title" @click.stop>
      <div class="search-box">
        <Search :size="19" :stroke-width="1.8" aria-hidden="true" />
        <input
          ref="inputEl"
          v-model="queryModel"
          type="search"
          autocomplete="off"
          placeholder="Search requests"
          aria-labelledby="request-search-title"
          @input="$emit('schedule-search')"
          @keydown.enter.prevent="$emit('open-first')"
          @keydown.esc.prevent="$emit('close')"
        />
        <button v-if="queryModel" class="search-clear-button" type="button" title="Clear search" @click="clearQuery">
          <X :size="16" :stroke-width="1.9" aria-hidden="true" />
        </button>
      </div>

      <div class="search-controls">
        <h2 id="request-search-title">Search</h2>
        <label class="toggle-field search-scope-toggle" :class="{ disabled: !selectedWebhook }">
          <input
            v-model="currentOnlyModel"
            type="checkbox"
            :disabled="!selectedWebhook"
            @change="$emit('schedule-search')"
          />
          <span>Current webhook</span>
        </label>
        <span v-if="loading" class="search-status">Searching</span>
      </div>

      <div class="search-results">
        <button
          v-for="result in results"
          :key="`${result.webhookSlug}:${result.request.id}`"
          class="search-result"
          type="button"
          @click="$emit('open-result', result)"
        >
          <span class="search-result-method">{{ result.request.method }}</span>
          <span class="search-result-main">
            <span class="search-result-target">{{ formatSearchResultTarget(result) }}</span>
            <span class="search-result-meta">
              <span>{{ result.webhookName || result.webhookSlug }}</span>
              <span>{{ formatTime(result.request.createdAt) }}</span>
              <span>{{ result.request.remoteIp }}</span>
            </span>
          </span>
        </button>

        <div v-if="query.trim() && !loading && results.length === 0" class="search-empty">
          No matches
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import { Search, X } from '@lucide/vue'
import { formatSearchResultTarget, formatTime } from '../../utils/formatters'

const props = defineProps({
  query: { type: String, default: '' },
  currentOnly: { type: Boolean, default: false },
  selectedWebhook: { type: Object, default: null },
  loading: { type: Boolean, default: false },
  results: { type: Array, default: () => [] }
})

const emit = defineEmits([
  'update:query',
  'update:currentOnly',
  'close',
  'clear',
  'schedule-search',
  'open-first',
  'open-result'
])

const inputEl = ref(null)

const queryModel = computed({
  get: () => props.query,
  set: (value) => emit('update:query', value)
})

const currentOnlyModel = computed({
  get: () => props.currentOnly,
  set: (value) => emit('update:currentOnly', value)
})

onMounted(() => {
  void nextTick(() => {
    inputEl.value?.focus()
  })
})

function clearQuery() {
  emit('clear')
  void nextTick(() => {
    inputEl.value?.focus()
  })
}
</script>
