<template>
  <div class="app-shell" @click="closeMenus">
    <aside class="left-rail" aria-label="Webhook navigation">
      <header class="brand-row">
        <div>
          <p class="overline">Webhook Console</p>
          <h1>{{ shareMode ? 'Shared stream' : 'Request capture' }}</h1>
        </div>
        <div v-if="!shareMode" class="brand-actions">
          <button class="new-button" type="button" :disabled="busy" @click.stop="createWebhook">
            <Plus :size="15" :stroke-width="1.8" aria-hidden="true" />
            New
          </button>
          <button class="icon-button telegram-settings-trigger" type="button" title="Telegram settings" @click.stop="openTelegramModal">
            <Send :size="15" :stroke-width="1.8" aria-hidden="true" />
          </button>
        </div>
      </header>

      <section v-if="selectedWebhook" class="current-hook">
        <div class="hook-main">
          <div class="hook-title">
            <span class="hook-name">{{ selectedWebhook.slug }}</span>
            <span class="hook-count">{{ requestCountLabel }}</span>
          </div>

          <div class="hook-actions">
            <button class="icon-button" type="button" title="Copy endpoint" @click.stop="copy(selectedWebhook.url, 'endpoint')">
              <Transition name="icon-fade" mode="out-in">
                <Check v-if="copiedTarget === 'endpoint'" key="endpoint-check" :size="16" :stroke-width="1.9" aria-hidden="true" />
                <Copy v-else key="endpoint-copy" :size="16" :stroke-width="1.85" aria-hidden="true" />
              </Transition>
            </button>

            <div v-if="!shareMode" class="menu-wrap">
              <button class="icon-button" type="button" title="Webhook actions" @click.stop="toggleActionMenu">
                <MoreHorizontal :size="17" :stroke-width="1.8" aria-hidden="true" />
              </button>
              <div v-if="actionMenuOpen" class="popover action-menu" @click.stop>
                <button type="button" @click="toggleAutoSwitch">
                  <RotateCw :size="15" :stroke-width="1.8" aria-hidden="true" />
                  {{ selectedAutoSwitch ? 'Auto-switch enabled' : 'Auto-switch disabled' }}
                </button>
                <button type="button" @click="toggleSoundNotifications">
                  <Volume2 v-if="selectedSoundEnabled" :size="15" :stroke-width="1.8" aria-hidden="true" />
                  <VolumeX v-else :size="15" :stroke-width="1.8" aria-hidden="true" />
                  {{ selectedSoundEnabled ? 'Sound enabled' : 'Sound disabled' }}
                </button>
                <button type="button" @click="toggleWebhookTelegram">
                  <Send :size="15" :stroke-width="1.8" aria-hidden="true" />
                  {{ selectedWebhook.telegramEnabled ? 'Telegram enabled' : 'Telegram disabled' }}
                </button>
                <button type="button" @click="setShareEnabled(!selectedWebhook.shareEnabled)">
                  <Share2 :size="15" :stroke-width="1.8" aria-hidden="true" />
                  {{ selectedWebhook.shareEnabled ? 'Disable guest access' : 'Enable guest access' }}
                </button>
                <button
                  v-if="selectedWebhook.shareEnabled && selectedWebhook.shareUrl"
                  type="button"
                  @click="copy(selectedWebhook.shareUrl, 'share')"
                >
                  <Transition name="icon-fade" mode="out-in">
                    <Check v-if="copiedTarget === 'share'" key="menu-share-check" :size="15" :stroke-width="1.9" aria-hidden="true" />
                    <Link2 v-else key="menu-share-link" :size="15" :stroke-width="1.8" aria-hidden="true" />
                  </Transition>
                  Copy share link
                </button>
                <button class="menu-danger" type="button" @click="deleteWebhook">
                  <Trash2 :size="15" :stroke-width="1.8" aria-hidden="true" />
                  Delete webhook
                </button>
              </div>
            </div>

            <div class="menu-wrap">
              <button class="icon-button" type="button" title="Select webhook" @click.stop="toggleSelector">
                <ChevronDown :size="16" :stroke-width="1.8" aria-hidden="true" />
              </button>
              <div v-if="selectorOpen" class="popover selector-menu" @click.stop>
                <button
                  v-for="webhook in webhooks"
                  :key="webhook.slug"
                  type="button"
                  :class="{ active: selectedSlug === webhook.slug, attention: isWebhookHighlighted(webhook.slug) }"
                  @pointerdown.stop="selectWebhookFromMenu(webhook.slug)"
                  @click.stop="selectWebhookFromMenu(webhook.slug)"
                >
                  <span class="selector-name">
                    <span>{{ webhook.slug }}</span>
                    <span v-if="isWebhookHighlighted(webhook.slug)" class="selector-badge">New</span>
                  </span>
                  <small>{{ webhook.requestCount }} requests</small>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="selectedWebhook.shareEnabled && selectedWebhook.shareUrl" class="share-strip">
          <span>Guest access</span>
          <button type="button" @click.stop="copy(selectedShareUrl, 'share')">
            <Transition name="icon-fade" mode="out-in">
              <Check v-if="copiedTarget === 'share'" key="share-check" :size="14" :stroke-width="1.9" aria-hidden="true" />
              <Link2 v-else key="share-link" :size="14" :stroke-width="1.85" aria-hidden="true" />
            </Transition>
            Copy link
          </button>
        </div>
      </section>

      <section v-else-if="loading" class="current-hook current-hook-skeleton" aria-hidden="true">
        <div class="hook-main">
          <div class="hook-title">
            <span class="skeleton-block skeleton-hook-name"></span>
            <span class="skeleton-block skeleton-hook-count"></span>
          </div>
          <div class="hook-actions">
            <span class="skeleton-block skeleton-icon"></span>
            <span v-if="!shareMode" class="skeleton-block skeleton-icon"></span>
            <span class="skeleton-block skeleton-icon"></span>
          </div>
        </div>
      </section>

      <section v-else-if="!loading" class="empty-panel">
        <p>No webhooks.</p>
        <button v-if="!shareMode" type="button" @click.stop="createWebhook">Create webhook</button>
      </section>

      <section class="request-nav" aria-label="Captured requests">
        <div class="rail-section-head">
          <span>Requests</span>
          <button v-if="!shareMode" class="icon-button search-trigger" type="button" title="Search requests" @click.stop="openSearchModal">
            <Search :size="15" :stroke-width="1.8" aria-hidden="true" />
          </button>
        </div>

        <div
          class="request-scroll"
          :class="{ 'request-scroll-empty': selectedWebhook && requests.length === 0 && !showRequestListSkeleton }"
        >
          <div v-if="showRequestListSkeleton" class="request-skeleton-list" aria-hidden="true">
            <div v-for="index in 6" :key="index" class="request-item request-skeleton-row">
              <span class="skeleton-block skeleton-method"></span>
              <span class="skeleton-block skeleton-request-path"></span>
              <span class="request-row-actions">
                <span class="skeleton-block skeleton-request-time"></span>
                <span v-if="!shareMode" class="request-delete-slot"></span>
              </span>
            </div>
          </div>

          <template v-else>
            <div
              v-for="request in requests"
              :key="request.id"
              class="request-item"
              :class="{ active: selectedRequest?.id === request.id }"
              role="button"
              tabindex="0"
              @click.stop="selectRequest(request)"
              @keydown.enter.prevent="selectRequest(request)"
              @keydown.space.prevent="selectRequest(request)"
            >
              <span class="request-method">{{ request.method }}</span>
              <span class="request-path">{{ formatRequestPath(request) }}</span>
              <span class="request-row-actions">
                <span class="request-time">{{ formatTime(request.createdAt) }}</span>
                <span v-if="!shareMode" class="request-delete-slot">
                  <button
                    v-if="selectedRequest?.id === request.id"
                    class="request-delete-button"
                    type="button"
                    title="Delete request"
                    :disabled="deletingRequestId === request.id"
                    @click.stop="deleteRequest(request)"
                    @keydown.stop
                  >
                    <Trash2 :size="14" :stroke-width="1.85" aria-hidden="true" />
                  </button>
                </span>
              </span>
            </div>

            <div v-if="selectedWebhook && requests.length === 0" class="empty-requests">
              <p>No requests yet</p>
            </div>
          </template>
        </div>
      </section>
    </aside>

    <main class="detail-pane">
      <div v-if="error" class="toast error">{{ error }}</div>
      <div v-if="notice" class="toast">{{ notice }}</div>

      <section v-if="showDetailSkeleton" class="request-detail request-detail-skeleton" aria-hidden="true">
        <header class="detail-head">
          <div class="detail-title">
            <span class="skeleton-block skeleton-detail-method"></span>
            <span class="skeleton-block skeleton-detail-uri"></span>
          </div>
          <div class="detail-actions">
            <span class="skeleton-block skeleton-icon"></span>
            <span class="skeleton-block skeleton-icon"></span>
            <span v-if="!shareMode" class="skeleton-block skeleton-icon"></span>
          </div>
        </header>
        <span class="skeleton-block skeleton-origin"></span>
        <div class="skeleton-headers">
          <span class="skeleton-block skeleton-section-title"></span>
          <div v-for="index in 5" :key="index" class="header-row skeleton-header-row">
            <span class="skeleton-block skeleton-header-name"></span>
            <span class="skeleton-block skeleton-header-value"></span>
          </div>
        </div>
        <div class="skeleton-body">
          <div class="skeleton-body-head">
            <span class="skeleton-block skeleton-section-title"></span>
            <span class="skeleton-block skeleton-body-meta"></span>
          </div>
          <span class="skeleton-block skeleton-body-box"></span>
        </div>
      </section>

      <section v-else-if="selectedRequest" :key="selectedRequest.id" class="request-detail" aria-label="Request details">
        <header class="detail-head">
          <div class="detail-title">
            <span class="detail-method">{{ selectedRequest.method }}</span>
            <h2>{{ formatBreakableUri(selectedRequest.target) }}</h2>
          </div>
          <div class="detail-actions">
            <button class="small-action icon-only" type="button" title="Copy request link" @click="copy(selectedRequest.detailUrl, 'request-link')">
              <Transition name="icon-fade" mode="out-in">
                <Check v-if="copiedTarget === 'request-link'" key="request-check" :size="15" :stroke-width="1.9" aria-hidden="true" />
                <Link2 v-else key="request-link-icon" :size="15" :stroke-width="1.85" aria-hidden="true" />
              </Transition>
            </button>
            <button class="small-action icon-only" type="button" title="Copy request" @click="copyFullRequest">
              <Transition name="icon-fade" mode="out-in">
                <Check v-if="copiedTarget === 'full-request'" key="full-request-check" :size="15" :stroke-width="1.9" aria-hidden="true" />
                <FileText v-else key="full-request-icon" :size="15" :stroke-width="1.85" aria-hidden="true" />
              </Transition>
            </button>
            <button
              v-if="!shareMode"
              class="small-action icon-only danger-action"
              type="button"
              title="Delete request"
              :disabled="deletingRequestId === selectedRequest.id"
              @click="deleteRequest(selectedRequest)"
            >
              <Trash2 :size="15" :stroke-width="1.85" aria-hidden="true" />
            </button>
          </div>
        </header>

        <p class="request-origin">
          From IP <span>{{ selectedRequest.remoteIp }}</span> at <span>{{ formatReadableDate(selectedRequest.createdAt) }}</span>
        </p>

        <div class="headers-list">
          <div class="section-label">Headers</div>
          <div v-for="[name, values] in headerRows" :key="name" class="header-row">
            <span
              class="copyable-cell header-name-copy"
              role="button"
              tabindex="0"
              title="Copy header name"
              :class="{ copied: copiedTarget === headerNameCopyTarget(name) }"
              @click="copyHeaderName(name)"
              @keydown.enter.prevent="copyHeaderName(name)"
              @keydown.space.prevent="copyHeaderName(name)"
            >
              {{ name }}
            </span>
            <code
              class="copyable-cell header-value-copy"
              role="button"
              tabindex="0"
              title="Copy header value"
              :class="{ copied: copiedTarget === headerValueCopyTarget(name) }"
              @click="copyHeaderValue(name, values)"
              @keydown.enter.prevent="copyHeaderValue(name, values)"
              @keydown.space.prevent="copyHeaderValue(name, values)"
            >
              {{ headerValueText(values) }}
            </code>
          </div>
        </div>

        <details v-if="hasBody" class="data-disclosure" open>
          <summary>
            <span class="section-label body-section-label">Request body</span>
            <span class="disclosure-actions">
              <button class="body-copy-button" type="button" title="Copy request body" @click.stop.prevent="copyRequestBody">
                <Transition name="icon-fade" mode="out-in">
                  <Check v-if="copiedTarget === 'request-body'" key="body-check" :size="14" :stroke-width="1.9" aria-hidden="true" />
                  <Copy v-else key="body-copy" :size="14" :stroke-width="1.85" aria-hidden="true" />
                </Transition>
              </button>
              <span>{{ formatBytes(selectedRequest.bodySize) }}</span>
              <ChevronDown class="disclosure-chevron" :size="15" :stroke-width="1.8" aria-hidden="true" />
            </span>
          </summary>
          <pre class="body-view">{{ bodyDisplay }}</pre>
        </details>
      </section>

      <section v-else-if="selectedWebhook" class="blank-detail">
        <p class="overline">Ready</p>
        <h2>Waiting for requests</h2>
        <p>{{ selectedWebhook.slug }}</p>
      </section>

      <section v-else class="blank-detail">
        <p class="overline">Loading</p>
        <h2>Preparing your workspace</h2>
      </section>
    </main>

    <div v-if="searchOpen" class="modal-backdrop search-backdrop" @click="closeSearchModal">
      <section class="search-modal" role="dialog" aria-modal="true" aria-labelledby="request-search-title" @click.stop>
        <div class="search-box">
          <Search :size="19" :stroke-width="1.8" aria-hidden="true" />
          <input
            ref="searchInput"
            v-model="searchQuery"
            type="search"
            autocomplete="off"
            placeholder="Search requests"
            aria-labelledby="request-search-title"
            @input="scheduleSearch()"
            @keydown.enter.prevent="openFirstSearchResult"
            @keydown.esc.prevent="closeSearchModal"
          />
          <button v-if="searchQuery" class="search-clear-button" type="button" title="Clear search" @click="clearSearchQuery">
            <X :size="16" :stroke-width="1.9" aria-hidden="true" />
          </button>
        </div>

        <div class="search-controls">
          <h2 id="request-search-title">Search</h2>
          <label class="toggle-field search-scope-toggle" :class="{ disabled: !selectedWebhook }">
            <input
              v-model="searchCurrentOnly"
              type="checkbox"
              :disabled="!selectedWebhook"
              @change="scheduleSearch()"
            />
            <span>Current webhook</span>
          </label>
          <span v-if="searchLoading" class="search-status">Searching</span>
        </div>

        <div class="search-results">
          <button
            v-for="result in searchResults"
            :key="`${result.webhookSlug}:${result.request.id}`"
            class="search-result"
            type="button"
            @click="openSearchResult(result)"
          >
            <span class="search-result-method">{{ result.request.method }}</span>
            <span class="search-result-main">
              <span class="search-result-target">{{ formatSearchResultTarget(result) }}</span>
              <span class="search-result-meta">
                <span>{{ result.webhookSlug }}</span>
                <span>{{ formatTime(result.request.createdAt) }}</span>
                <span>{{ result.request.remoteIp }}</span>
              </span>
            </span>
          </button>

          <div v-if="searchQuery.trim() && !searchLoading && searchResults.length === 0" class="search-empty">
            No matches
          </div>
        </div>
      </section>
    </div>

    <div v-if="telegramModalOpen" class="modal-backdrop" @click="closeTelegramModal">
      <section class="telegram-modal" role="dialog" aria-modal="true" aria-labelledby="telegram-settings-title" @click.stop>
        <header class="modal-head">
          <div>
            <p class="overline">Notifications</p>
            <h2 id="telegram-settings-title">Telegram settings</h2>
          </div>
        </header>

        <form class="telegram-form" @submit.prevent="saveTelegramSettings">
          <div class="form-grid">
            <label class="field">
              <span>Bot token</span>
              <input
                v-model.trim="telegramForm.botToken"
                type="password"
                autocomplete="off"
                :required="!telegramSettings.configured"
                :placeholder="telegramSettings.configured ? 'Configured' : '123456:ABC-DEF'"
              />
            </label>

            <label class="field">
              <span>Chat ID</span>
              <input v-model.trim="telegramForm.chatId" type="text" required autocomplete="off" placeholder="-1001234567890" />
            </label>
          </div>

          <section class="proxy-settings">
            <label class="toggle-field">
              <input v-model="telegramForm.proxyEnabled" type="checkbox" />
              <span>Use SOCKS5 proxy</span>
            </label>

            <div v-if="telegramForm.proxyEnabled" class="form-grid proxy-grid">
              <label class="field">
                <span>Hostname or IP</span>
                <input v-model.trim="telegramForm.proxyHost" type="text" required autocomplete="off" placeholder="127.0.0.1" />
              </label>

              <label class="field">
                <span>Port</span>
                <input v-model.number="telegramForm.proxyPort" type="number" min="1" max="65535" required placeholder="9050" />
              </label>

              <label class="field">
                <span>Username</span>
                <input v-model.trim="telegramForm.proxyUsername" type="text" autocomplete="off" />
              </label>

              <label class="field">
                <span>Password</span>
                <input
                  v-model="telegramForm.proxyPassword"
                  type="password"
                  autocomplete="off"
                  :placeholder="telegramSettings.proxyPasswordConfigured ? 'Configured' : ''"
                />
              </label>
            </div>
          </section>

          <div class="modal-actions">
            <button class="small-action" type="button" :disabled="telegramTesting || telegramSaving" @click="testTelegramSettings">
              Test notification
            </button>
            <button class="small-action primary-action" type="submit" :disabled="telegramSaving || telegramTesting">
              Save settings
            </button>
          </div>
        </form>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Check,
  ChevronDown,
  Copy,
  FileText,
  Link2,
  MoreHorizontal,
  Plus,
  RotateCw,
  Search,
  Send,
  Share2,
  Trash2,
  Volume2,
  VolumeX,
  X
} from '@lucide/vue'

const autoSwitchStorageKey = 'webhook-auto-switch'
const soundStorageKey = 'webhook-sound-alerts'
const selectedWebhookStorageKey = 'webhook-selected-slug'

const emptyTelegramSettings = {
  configured: false,
  chatId: '',
  proxyEnabled: false,
  proxyHost: '',
  proxyPort: '',
  proxyUsername: '',
  proxyPasswordConfigured: false
}

const emptyTelegramForm = {
  botToken: '',
  chatId: '',
  proxyEnabled: false,
  proxyHost: '',
  proxyPort: '',
  proxyUsername: '',
  proxyPassword: ''
}

const session = ref(null)
const webhooks = ref([])
const selectedSlug = ref('')
const requests = ref([])
const selectedRequest = ref(null)
const loading = ref(true)
const loadingRequests = ref(false)
const showingRequestSkeleton = ref(false)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const actionMenuOpen = ref(false)
const selectorOpen = ref(false)
const searchOpen = ref(false)
const searchQuery = ref('')
const searchResults = ref([])
const searchLoading = ref(false)
const searchCurrentOnly = ref(false)
const searchInput = ref(null)
const autoSwitchPrefs = ref(loadAutoSwitchPrefs())
const soundPrefs = ref(loadSoundPrefs())
const highlightedWebhookSlugs = ref([])
const copiedTarget = ref('')
const deletingRequestId = ref('')
const telegramModalOpen = ref(false)
const telegramSettingsLoaded = ref(false)
const telegramSettings = ref({ ...emptyTelegramSettings })
const telegramForm = ref({ ...emptyTelegramForm })
const telegramSaving = ref(false)
const telegramTesting = ref(false)
const initialRoute = parseRoute()
const shareMode = initialRoute.mode === 'share'
const shareToken = initialRoute.shareToken
let pollTimer = null
let noticeTimer = null
let copiedTimer = null
let searchTimer = null
let searchAbortController = null
let audioContext = null
let soundQueue = 0
let playingSound = false

const selectedWebhook = computed(() => webhooks.value.find((hook) => hook.slug === selectedSlug.value) || null)

const showRequestListSkeleton = computed(() => loading.value || showingRequestSkeleton.value)

const showDetailSkeleton = computed(() => loading.value || showingRequestSkeleton.value)

const selectedAutoSwitch = computed(() => {
  if (!selectedSlug.value) return true
  return autoSwitchPrefs.value[selectedSlug.value] !== false
})

const selectedSoundEnabled = computed(() => {
  if (!selectedSlug.value) return false
  return soundPrefs.value[selectedSlug.value] === true
})

const requestCountLabel = computed(() => {
  const count = requests.value.length || selectedWebhook.value?.requestCount || 0
  return `${count} ${count === 1 ? 'request' : 'requests'}`
})

const selectedShareUrl = computed(() => {
  if (!selectedWebhook.value?.shareUrl) return ''
  if (!selectedRequest.value) return selectedWebhook.value.shareUrl
  const url = new URL(selectedWebhook.value.shareUrl, window.location.origin)
  return `${window.location.origin}/share/${selectedWebhook.value.slug}/${selectedRequest.value.id}?id=${url.searchParams.get('id')}`
})

const headerRows = computed(() => {
  if (!selectedRequest.value?.headers) return []
  return Object.entries(selectedRequest.value.headers).sort(([a], [b]) => a.localeCompare(b))
})

const bodyDisplay = computed(() => {
  if (!selectedRequest.value) return ''
  if (selectedRequest.value.bodyEncoding === 'base64') {
    return selectedRequest.value.bodyBase64 || 'No body'
  }
  const suffix = selectedRequest.value.bodyTruncated ? '\n\n[body truncated]' : ''
  return (selectedRequest.value.bodyText || 'No body') + suffix
})

const hasBody = computed(() => {
  if (!selectedRequest.value) return false
  return selectedRequest.value.bodySize > 0 || selectedRequest.value.bodyText !== '' || selectedRequest.value.bodyBase64 !== ''
})

onMounted(async () => {
  try {
    if (shareMode) {
      await loadShared()
    } else {
      await loadOwned()
    }
    startPolling()
    window.addEventListener('click', closeMenus)
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})

onBeforeUnmount(() => {
  if (pollTimer) window.clearInterval(pollTimer)
  if (noticeTimer) window.clearTimeout(noticeTimer)
  if (copiedTimer) window.clearTimeout(copiedTimer)
  if (searchTimer) window.clearTimeout(searchTimer)
  abortSearch()
  window.removeEventListener('click', closeMenus)
})

async function loadOwned() {
  session.value = await api('/api/session')
  const data = await api('/api/webhooks')
  webhooks.value = data.webhooks
  if (webhooks.value.length === 0 && !initialRoute.slug) {
    await createWebhook(true)
    return
  }
  const nextSlug = resolveOwnedInitialSlug()
  setSelectedSlug(nextSlug, { persist: !initialRoute.slug || hasWebhookSlug(nextSlug) })
  if (selectedSlug.value) {
    await loadRequests(initialRoute.requestId)
  }
}

async function loadShared() {
  if (!initialRoute.slug || !shareToken) {
    throw new Error('Invalid share link')
  }
  const data = await api(`/api/share/${initialRoute.slug}?id=${encodeURIComponent(shareToken)}`)
  webhooks.value = [data.webhook]
  setSelectedSlug(data.webhook.slug, { persist: false })
  requests.value = data.requests
  await selectInitialRequest(initialRoute.requestId)
}

async function createWebhook(silent = false) {
  if (shareMode) return
  const silentCreate = silent === true
  busy.value = true
  error.value = ''
  closeMenus()
  try {
    if (!session.value) session.value = await api('/api/session')
    const data = await api('/api/webhooks', {
      method: 'POST',
      headers: csrfHeaders()
    })
    webhooks.value = [data.webhook, ...webhooks.value]
    setSelectedSlug(data.webhook.slug)
    requests.value = []
    selectedRequest.value = null
    history.replaceState({}, '', '/')
    if (!silentCreate) showNotice('Webhook created')
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}

async function deleteWebhook() {
  if (!selectedWebhook.value || shareMode) return
  busy.value = true
  error.value = ''
  closeMenus()
  try {
    await api(`/api/webhooks/${selectedWebhook.value.slug}`, {
      method: 'DELETE',
      headers: csrfHeaders()
    })
    webhooks.value = webhooks.value.filter((hook) => hook.slug !== selectedWebhook.value.slug)
    setSelectedSlug(webhooks.value[0]?.slug || '')
    requests.value = []
    selectedRequest.value = null
    history.replaceState({}, '', '/')
    if (selectedSlug.value) await loadRequests(undefined, { skeleton: true })
    showNotice('Webhook deleted')
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}

async function setShareEnabled(enabled) {
  if (!selectedWebhook.value || shareMode) return
  busy.value = true
  error.value = ''
  try {
    const data = await api(`/api/webhooks/${selectedWebhook.value.slug}/share`, {
      method: 'PATCH',
      headers: csrfHeaders(),
      body: JSON.stringify({ enabled })
    })
    webhooks.value = webhooks.value.map((hook) => (hook.slug === data.webhook.slug ? data.webhook : hook))
    showNotice(data.webhook.shareEnabled ? 'Guest access enabled' : 'Guest access disabled')
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}

async function setWebhookTelegramEnabled(enabled) {
  if (!selectedWebhook.value || shareMode) return
  busy.value = true
  error.value = ''
  try {
    const data = await api(`/api/webhooks/${selectedWebhook.value.slug}/telegram`, {
      method: 'PATCH',
      headers: csrfHeaders(),
      body: JSON.stringify({ enabled })
    })
    webhooks.value = webhooks.value.map((hook) => (hook.slug === data.webhook.slug ? data.webhook : hook))
  } catch (err) {
    if (err.message === 'telegram settings required') {
      await openTelegramModal()
    } else {
      error.value = err.message
    }
  } finally {
    busy.value = false
  }
}

async function selectWebhook(slug) {
  clearWebhookHighlight(slug)
  if (selectedSlug.value === slug) return
  setSelectedSlug(slug)
  selectedRequest.value = null
  requests.value = []
  history.replaceState({}, '', shareMode ? `/share/${slug}?id=${encodeURIComponent(shareToken)}` : '/')
  await loadRequests(undefined, { skeleton: true })
}

function selectWebhookFromMenu(slug) {
  closeMenus()
  void selectWebhook(slug)
}

function openSearchModal() {
  if (shareMode) return
  closeMenus()
  searchCurrentOnly.value = false
  searchOpen.value = true
  error.value = ''
  void nextTick(() => {
    searchInput.value?.focus()
  })
  if (searchQuery.value.trim()) {
    scheduleSearch(0)
  }
}

function closeSearchModal() {
  searchOpen.value = false
  if (searchTimer) {
    window.clearTimeout(searchTimer)
    searchTimer = null
  }
  abortSearch()
  searchLoading.value = false
}

function clearSearchQuery() {
  searchQuery.value = ''
  searchResults.value = []
  searchLoading.value = false
  if (searchTimer) {
    window.clearTimeout(searchTimer)
    searchTimer = null
  }
  abortSearch()
  void nextTick(() => {
    searchInput.value?.focus()
  })
}

function scheduleSearch(delay = 180) {
  if (!searchOpen.value || shareMode) return
  if (searchTimer) window.clearTimeout(searchTimer)
  if (!searchQuery.value.trim()) {
    searchTimer = null
    searchResults.value = []
    searchLoading.value = false
    abortSearch()
    return
  }
  searchTimer = window.setTimeout(() => {
    searchTimer = null
    void runSearch()
  }, delay)
}

async function runSearch() {
  const query = searchQuery.value.trim()
  if (!query) {
    searchResults.value = []
    searchLoading.value = false
    abortSearch()
    return
  }

  abortSearch()
  const controller = new AbortController()
  searchAbortController = controller
  searchLoading.value = true
  error.value = ''
  const params = new URLSearchParams({
    q: query,
    limit: '40'
  })
  if (searchCurrentOnly.value && selectedSlug.value) {
    params.set('webhook', selectedSlug.value)
  }

  try {
    const data = await api(`/api/search?${params.toString()}`, { signal: controller.signal })
    if (controller.signal.aborted) return
    searchResults.value = data.results || []
  } catch (err) {
    if (err.name !== 'AbortError') error.value = err.message
  } finally {
    if (searchAbortController === controller) {
      searchAbortController = null
      searchLoading.value = false
    }
  }
}

function abortSearch() {
  if (!searchAbortController) return
  searchAbortController.abort()
  searchAbortController = null
}

function openFirstSearchResult() {
  if (searchResults.value.length === 0) {
    scheduleSearch(0)
    return
  }
  void openSearchResult(searchResults.value[0])
}

async function openSearchResult(result) {
  const slug = result?.webhookSlug || ''
  const request = result?.request || null
  if (!slug || !request) return

  closeSearchModal()
  if (selectedSlug.value !== slug) {
    setSelectedSlug(slug)
    requests.value = []
  }
  selectedRequest.value = request
  replaceRequestRoute(request)
  await loadRequests(request.id, { skeleton: true })
}

async function loadRequests(preferredId = selectedRequest.value?.id, options = {}) {
  if (!selectedWebhook.value) return
  loadingRequests.value = true
  if (options.skeleton) showingRequestSkeleton.value = true
  error.value = ''
  const previousNewestId = requests.value[0]?.id || ''
  try {
    const slug = selectedWebhook.value.slug
    const endpoint = shareMode
      ? `/api/share/${slug}/requests?id=${encodeURIComponent(shareToken)}`
      : `/api/webhooks/${slug}/requests`
    const data = await api(endpoint)
    const newRequestCount = countNewRequests(data.requests, previousNewestId)
    if (previousNewestId && newRequestCount > 0) {
      queueRequestSound(slug, newRequestCount)
    }
    requests.value = data.requests
    updateSelectedWebhookCount()
    await selectInitialRequest(preferredId, previousNewestId)
  } catch (err) {
    error.value = err.message
  } finally {
    loadingRequests.value = false
    if (options.skeleton) showingRequestSkeleton.value = false
  }
}

function updateSelectedWebhookCount() {
  if (!selectedWebhook.value) return
  webhooks.value = webhooks.value.map((hook) => {
    if (hook.slug !== selectedWebhook.value.slug) return hook
    return {
      ...hook,
      requestCount: requests.value.length,
      lastRequestAt: requests.value[0]?.createdAt || null
    }
  })
}

async function loadWebhookSummaries() {
  if (shareMode) return
  try {
    const data = await api('/api/webhooks')
    mergeWebhookSummaries(data.webhooks)
  } catch (err) {
    error.value = err.message
  }
}

function mergeWebhookSummaries(nextWebhooks) {
  const previousBySlug = new Map(webhooks.value.map((webhook) => [webhook.slug, webhook]))
  nextWebhooks.forEach((webhook) => {
    const previous = previousBySlug.get(webhook.slug)
    if (!previous || webhook.slug === selectedSlug.value || !isSoundEnabled(webhook.slug)) return
    if (!hasNewWebhookRequest(previous, webhook)) return
    flagWebhookHighlight(webhook.slug)
    queueRequestSound(webhook.slug, Math.max(1, webhook.requestCount - previous.requestCount))
  })
  webhooks.value = nextWebhooks
}

function hasNewWebhookRequest(previous, next) {
  if (next.requestCount > previous.requestCount) return true
  if (!next.lastRequestAt) return false
  if (!previous.lastRequestAt) return true
  return new Date(next.lastRequestAt).getTime() > new Date(previous.lastRequestAt).getTime()
}

async function selectInitialRequest(preferredId, previousNewestId = '') {
  const newest = requests.value[0] || null
  if (previousNewestId && newest?.id && newest.id !== previousNewestId && selectedAutoSwitch.value) {
    selectedRequest.value = newest
    replaceRequestRoute(newest)
    return
  }
  const match = preferredId ? requests.value.find((request) => request.id === preferredId) : null
  if (match) {
    selectedRequest.value = match
    replaceRequestRoute(match)
    return
  }
  if (preferredId) {
    const request = await loadRequestDetail(preferredId)
    if (request) {
      selectedRequest.value = request
      replaceRequestRoute(request)
      return
    }
  }
  selectedRequest.value = newest
  if (selectedRequest.value) {
    replaceRequestRoute(selectedRequest.value)
  }
}

async function loadRequestDetail(requestId) {
  if (!selectedWebhook.value || !requestId) return null
  const slug = selectedWebhook.value.slug
  const endpoint = shareMode
    ? `/api/share/${slug}/requests/${requestId}?id=${encodeURIComponent(shareToken)}`
    : `/api/webhooks/${slug}/requests/${requestId}`
  try {
    const data = await api(endpoint)
    return data.request || null
  } catch {
    return null
  }
}

function selectRequest(request) {
  selectedRequest.value = request
  replaceRequestRoute(request)
}

async function deleteRequest(request = selectedRequest.value) {
  if (!selectedWebhook.value || !request || shareMode) return
  const slug = selectedWebhook.value.slug
  const requestId = request.id
  const currentIndex = requests.value.findIndex((item) => item.id === requestId)
  const deletingSelected = selectedRequest.value?.id === requestId
  deletingRequestId.value = requestId
  error.value = ''
  try {
    await api(`/api/webhooks/${slug}/requests/${requestId}`, {
      method: 'DELETE',
      headers: csrfHeaders()
    })
    if (selectedWebhook.value?.slug !== slug) return
    const remaining = requests.value.filter((item) => item.id !== requestId)
    requests.value = remaining
    updateSelectedWebhookCount()
    if (deletingSelected) {
      const nextRequest = remaining[currentIndex] || remaining[currentIndex - 1] || remaining[0] || null
      selectedRequest.value = nextRequest
      if (nextRequest) {
        replaceRequestRoute(nextRequest)
      } else {
        history.replaceState({}, '', '/')
      }
    }
  } catch (err) {
    error.value = err.message
  } finally {
    if (deletingRequestId.value === requestId) deletingRequestId.value = ''
  }
}

function replaceRequestRoute(request) {
  if (!request || !selectedWebhook.value) return
  const path = shareMode
    ? `/share/${selectedWebhook.value.slug}/${request.id}?id=${encodeURIComponent(shareToken)}`
    : `/at/${selectedWebhook.value.slug}/${request.id}`
  history.replaceState({}, '', path)
}

function startPolling() {
  if (pollTimer) window.clearInterval(pollTimer)
  pollTimer = window.setInterval(() => {
    if (selectedWebhook.value) loadRequests()
    if (!shareMode) loadWebhookSummaries()
  }, 3000)
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    credentials: 'same-origin',
    ...options,
    headers: {
      Accept: 'application/json',
      ...(options.headers || {})
    }
  })
  if (response.status === 204) return null
  const contentType = response.headers.get('content-type') || ''
  const data = contentType.includes('application/json') ? await response.json() : {}
  if (!response.ok) {
    throw new Error(data.error || `Request failed with ${response.status}`)
  }
  return data
}

function csrfHeaders() {
  return {
    'Content-Type': 'application/json',
    'X-CSRF-Token': session.value?.csrfToken || ''
  }
}

async function copy(value, message) {
  if (!value) return
  try {
    await navigator.clipboard.writeText(value)
    markCopied(message)
  } catch {
    showNotice('Copy failed')
  }
}

function copyHeaderName(name) {
  void copy(name, headerNameCopyTarget(name))
}

function copyHeaderValue(name, values) {
  void copy(headerValueText(values), headerValueCopyTarget(name))
}

function copyRequestBody() {
  if (!selectedRequest.value) return
  const value = selectedRequest.value.bodyEncoding === 'base64'
    ? selectedRequest.value.bodyBase64
    : selectedRequest.value.bodyText
  void copy(value || '', 'request-body')
}

function copyFullRequest() {
  if (!selectedRequest.value) return
  void copy(formatHTTPClipboardRequest(selectedRequest.value), 'full-request')
}

function formatHTTPClipboardRequest(request) {
  const target = request.target || request.path || '/'
  const lines = [`${request.method || 'GET'} ${target} HTTP/1.1`]
  const host = requestHost(request)
  if (host) lines.push(`Host: ${host}`)

  const headers = Object.entries(request.headers || {})
    .filter(([name]) => !nameEquals(name, 'Host'))
    .sort(([a], [b]) => a.toLowerCase().localeCompare(b.toLowerCase()))

  for (const [name, values] of headers) {
    for (const value of values || []) {
      lines.push(`${name}: ${value}`)
    }
  }

  if (!request.bodySize && !request.bodyTruncated) {
    return lines.join('\n')
  }

  const body = request.bodyEncoding === 'base64'
    ? '[binary body omitted]'
    : (request.bodyText || '')
  const suffix = request.bodyTruncated ? `${body ? '\n\n' : ''}[body truncated]` : ''
  return `${lines.join('\n')}\n\n${body}${suffix}`
}

function requestHost(request) {
  try {
    return new URL(request.detailUrl || window.location.href, window.location.origin).host
  } catch {
    return window.location.host
  }
}

function nameEquals(left, right) {
  return String(left).toLowerCase() === String(right).toLowerCase()
}

function headerNameCopyTarget(name) {
  return `header-name:${name}`
}

function headerValueCopyTarget(name) {
  return `header-value:${name}`
}

function headerValueText(values) {
  return values.join(', ')
}

function markCopied(target) {
  copiedTarget.value = target
  if (copiedTimer) window.clearTimeout(copiedTimer)
  copiedTimer = window.setTimeout(() => {
    copiedTarget.value = ''
  }, 1400)
}

function toggleAutoSwitch() {
  if (!selectedSlug.value) return
  autoSwitchPrefs.value = {
    ...autoSwitchPrefs.value,
    [selectedSlug.value]: !selectedAutoSwitch.value
  }
  saveAutoSwitchPrefs(autoSwitchPrefs.value)
}

async function toggleWebhookTelegram() {
  if (!selectedWebhook.value || shareMode) return
  if (selectedWebhook.value.telegramEnabled) {
    await setWebhookTelegramEnabled(false)
    return
  }
  await loadTelegramSettings()
  if (!telegramSettings.value.configured) {
    await openTelegramModal()
    return
  }
  await setWebhookTelegramEnabled(true)
}

async function toggleSoundNotifications() {
  if (!selectedSlug.value) return
  const enabled = !selectedSoundEnabled.value
  soundPrefs.value = {
    ...soundPrefs.value,
    [selectedSlug.value]: enabled
  }
  saveSoundPrefs(soundPrefs.value)
  if (!enabled) {
    clearWebhookHighlight(selectedSlug.value)
  } else {
    await unlockAudio()
  }
}

function isSoundEnabled(slug) {
  return soundPrefs.value[slug] === true
}

function isWebhookHighlighted(slug) {
  return highlightedWebhookSlugs.value.includes(slug)
}

function flagWebhookHighlight(slug) {
  if (highlightedWebhookSlugs.value.includes(slug)) return
  highlightedWebhookSlugs.value = [...highlightedWebhookSlugs.value, slug]
}

function clearWebhookHighlight(slug) {
  highlightedWebhookSlugs.value = highlightedWebhookSlugs.value.filter((item) => item !== slug)
}

function countNewRequests(nextRequests, previousNewestId) {
  if (!previousNewestId || nextRequests.length === 0) return 0
  const previousIndex = nextRequests.findIndex((request) => request.id === previousNewestId)
  return previousIndex === -1 ? 1 : previousIndex
}

function queueRequestSound(slug, count = 1) {
  if (!isSoundEnabled(slug)) return
  soundQueue = Math.min(soundQueue + Math.min(count, 3), 6)
  void playQueuedSounds()
}

async function playQueuedSounds() {
  if (playingSound) return
  playingSound = true
  try {
    while (soundQueue > 0) {
      soundQueue -= 1
      await playRequestTone()
      await wait(90)
    }
  } finally {
    playingSound = false
    if (soundQueue > 0) void playQueuedSounds()
  }
}

async function playRequestTone() {
  const context = await unlockAudio()
  if (!context) return
  const now = context.currentTime
  const oscillator = context.createOscillator()
  const gain = context.createGain()
  oscillator.type = 'sine'
  oscillator.frequency.setValueAtTime(660, now)
  oscillator.frequency.exponentialRampToValueAtTime(520, now + 0.12)
  gain.gain.setValueAtTime(0.0001, now)
  gain.gain.exponentialRampToValueAtTime(0.035, now + 0.018)
  gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.16)
  oscillator.connect(gain)
  gain.connect(context.destination)
  oscillator.start(now)
  oscillator.stop(now + 0.18)
  await wait(190)
}

async function unlockAudio() {
  const AudioContextClass = window.AudioContext || window.webkitAudioContext
  if (!AudioContextClass) return null
  if (!audioContext) audioContext = new AudioContextClass()
  if (audioContext.state === 'suspended') {
    try {
      await audioContext.resume()
    } catch {
      return null
    }
  }
  return audioContext
}

function wait(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

async function openTelegramModal() {
  if (shareMode) return
  closeMenus()
  telegramModalOpen.value = true
  await loadTelegramSettings()
}

function closeTelegramModal() {
  if (telegramSaving.value || telegramTesting.value) return
  telegramModalOpen.value = false
}

async function loadTelegramSettings(force = false) {
  if (shareMode || (telegramSettingsLoaded.value && !force)) return
  error.value = ''
  try {
    const data = await api('/api/telegram')
    telegramSettings.value = data.settings || { ...emptyTelegramSettings }
    telegramSettingsLoaded.value = true
    resetTelegramForm()
  } catch (err) {
    error.value = err.message
  }
}

function resetTelegramForm() {
  telegramForm.value = {
    ...emptyTelegramForm,
    chatId: telegramSettings.value.chatId || '',
    proxyEnabled: telegramSettings.value.proxyEnabled || false,
    proxyHost: telegramSettings.value.proxyHost || '',
    proxyPort: telegramSettings.value.proxyPort || '',
    proxyUsername: telegramSettings.value.proxyUsername || ''
  }
}

function telegramPayload() {
  return {
    botToken: telegramForm.value.botToken,
    chatId: telegramForm.value.chatId,
    proxyEnabled: telegramForm.value.proxyEnabled,
    proxyHost: telegramForm.value.proxyHost,
    proxyPort: Number(telegramForm.value.proxyPort) || 0,
    proxyUsername: telegramForm.value.proxyUsername,
    proxyPassword: telegramForm.value.proxyPassword
  }
}

async function saveTelegramSettings() {
  if (shareMode) return
  telegramSaving.value = true
  error.value = ''
  try {
    const data = await api('/api/telegram', {
      method: 'PATCH',
      headers: csrfHeaders(),
      body: JSON.stringify(telegramPayload())
    })
    telegramSettings.value = data.settings
    telegramSettingsLoaded.value = true
    resetTelegramForm()
    showNotice('Telegram settings saved')
  } catch (err) {
    error.value = err.message
  } finally {
    telegramSaving.value = false
  }
}

async function testTelegramSettings() {
  if (shareMode) return
  telegramTesting.value = true
  error.value = ''
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), 12000)
  try {
    await api('/api/telegram/test', {
      method: 'POST',
      headers: csrfHeaders(),
      body: JSON.stringify(telegramPayload()),
      signal: controller.signal
    })
    showNotice('Telegram test sent')
  } catch (err) {
    error.value = err.name === 'AbortError' ? 'Telegram test timed out after 12 seconds' : err.message
  } finally {
    window.clearTimeout(timeout)
    telegramTesting.value = false
  }
}

function toggleActionMenu() {
  actionMenuOpen.value = !actionMenuOpen.value
  selectorOpen.value = false
}

function toggleSelector() {
  selectorOpen.value = !selectorOpen.value
  actionMenuOpen.value = false
}

function closeMenus() {
  actionMenuOpen.value = false
  selectorOpen.value = false
}

function showNotice(message) {
  notice.value = message
  if (noticeTimer) window.clearTimeout(noticeTimer)
  noticeTimer = window.setTimeout(() => {
    notice.value = ''
  }, 2500)
}

function loadAutoSwitchPrefs() {
  return loadStoredPrefs(autoSwitchStorageKey)
}

function loadSoundPrefs() {
  return loadStoredPrefs(soundStorageKey)
}

function resolveOwnedInitialSlug() {
  if (initialRoute.slug) return initialRoute.slug
  const storedSlug = loadSelectedWebhookSlug()
  if (storedSlug && hasWebhookSlug(storedSlug)) {
    return storedSlug
  }
  return webhooks.value[0]?.slug || ''
}

function hasWebhookSlug(slug) {
  return webhooks.value.some((webhook) => webhook.slug === slug)
}

function loadStoredPrefs(key) {
  try {
    return JSON.parse(window.localStorage.getItem(key) || '{}')
  } catch {
    return {}
  }
}

function loadSelectedWebhookSlug() {
  try {
    return window.localStorage.getItem(selectedWebhookStorageKey) || ''
  } catch {
    return ''
  }
}

function setSelectedSlug(slug, options = {}) {
  selectedSlug.value = slug
  if (options.persist === false || shareMode) return
  saveSelectedWebhookSlug(slug)
}

function saveSelectedWebhookSlug(slug) {
  try {
    if (slug) {
      window.localStorage.setItem(selectedWebhookStorageKey, slug)
    } else {
      window.localStorage.removeItem(selectedWebhookStorageKey)
    }
  } catch {
    // Ignore storage failures; route-based navigation still works.
  }
}

function saveAutoSwitchPrefs(value) {
  window.localStorage.setItem(autoSwitchStorageKey, JSON.stringify(value))
}

function saveSoundPrefs(value) {
  window.localStorage.setItem(soundStorageKey, JSON.stringify(value))
}

function parseRoute() {
  const parts = window.location.pathname.split('/').filter(Boolean)
  if (parts[0] === 'share') {
    return {
      mode: 'share',
      slug: parts[1] || '',
      requestId: parts[2] || '',
      shareToken: new URLSearchParams(window.location.search).get('id') || ''
    }
  }
  if (parts[0] === 'at' && parts.length >= 3) {
    return {
      mode: 'owner',
      slug: parts[1],
      requestId: parts[2],
      shareToken: ''
    }
  }
  return { mode: 'owner', slug: '', requestId: '', shareToken: '' }
}

function formatTime(value) {
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(new Date(value))
}

function formatRequestPath(request) {
  const target = request?.target || ''
  const slug = selectedWebhook.value?.slug || ''
  if (!slug || !target.startsWith(`/at/${slug}`)) return target || '/'
  const rest = target.slice(`/at/${slug}`.length)
  return rest === '' ? '/' : rest
}

function formatSearchResultTarget(result) {
  const target = result?.request?.target || result?.request?.path || ''
  const slug = result?.webhookSlug || ''
  if (!slug || !target.startsWith(`/at/${slug}`)) return target || '/'
  const rest = target.slice(`/at/${slug}`.length)
  return rest === '' ? '/' : rest
}

function formatBreakableUri(value) {
  return (value || '').replace(/([/?&])/g, '$1\u200b')
}

function formatReadableDate(value) {
  const parts = new Intl.DateTimeFormat(undefined, {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).formatToParts(new Date(value))
  const byType = Object.fromEntries(parts.map((part) => [part.type, part.value]))
  return `${byType.day} ${byType.month} ${byType.year} ${byType.hour}:${byType.minute}:${byType.second}`
}

function formatBytes(value) {
  if (value < 0) return 'unknown'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`
  return `${(value / 1024 / 1024).toFixed(1)} MiB`
}
</script>
