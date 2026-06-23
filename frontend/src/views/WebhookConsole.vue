<template>
  <div class="app-shell" @click="closeMenus">
    <AppSidebar
      :share-mode="shareMode"
      :busy="busy"
      :selected-webhook="selectedWebhook"
      :selected-slug="selectedSlug"
      :webhooks="webhooks"
      :requests="requests"
      :selected-request="selectedRequest"
      :loading="loading"
      :show-request-list-skeleton="showRequestListSkeleton"
      :action-menu-open="actionMenuOpen"
      :selector-open="selectorOpen"
      :request-count-label="requestCountLabel"
      :selected-auto-switch="selectedAutoSwitch"
      :selected-sound-enabled="selectedSoundEnabled"
      :selected-share-url="selectedShareUrl"
      :copied-target="copiedTarget"
      :deleting-request-id="deletingRequestId"
      :highlighted-webhook-slugs="highlightedWebhookSlugs"
      @create-webhook="createWebhook"
      @open-telegram-modal="openTelegramModal"
      @copy="copy"
      @toggle-action-menu="toggleActionMenu"
      @toggle-auto-switch="toggleAutoSwitch"
      @toggle-sound-notifications="toggleSoundNotifications"
      @toggle-webhook-telegram="toggleWebhookTelegram"
      @open-response-modal="openResponseModal"
      @open-rename-modal="openRenameModal"
      @set-share-enabled="setShareEnabled"
      @delete-webhook="deleteWebhook"
      @toggle-selector="toggleSelector"
      @select-webhook="selectWebhookFromMenu"
      @open-search-modal="openSearchModal"
      @hide-note-tooltip="hideRequestNoteTooltip"
      @select-request="selectRequest"
      @show-note-tooltip="showRequestNoteTooltip"
      @position-note-tooltip="positionRequestNoteTooltip"
      @delete-request="deleteRequest"
    />

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
                <ClipboardCopy v-else key="full-request-icon" :size="15" :stroke-width="1.85" aria-hidden="true" />
              </Transition>
            </button>
            <button
              v-if="!shareMode"
              class="small-action icon-only"
              type="button"
              :title="selectedRequest.note ? 'Edit request note' : 'Add request note'"
              @click="openRequestNoteModal"
            >
              <StickyNote :size="15" :stroke-width="1.85" aria-hidden="true" />
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
            <span class="section-label body-section-label">
              Request body
            </span>
            <span class="disclosure-actions">
              <button
                v-if="bodyHasEnhancedView"
                class="small-action body-view-toggle"
                type="button"
                :title="enhancedBodyView ? 'Show default body view' : 'Show formatted body view'"
                @click.stop.prevent="toggleBodyView"
              >
                {{ enhancedBodyView ? 'Formatted' : 'Default' }}
              </button>
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
          <div v-if="bodyView.kind === 'image'" class="body-image-view">
            <img :src="bodyView.src" :alt="bodyView.alt" />
          </div>
          <div v-else-if="bodyView.kind === 'multipart'" class="multipart-view">
            <section v-for="(part, index) in bodyView.parts" :key="index" class="multipart-part">
              <header class="multipart-part-head">
                <span>{{ part.title }}</span>
                <span>{{ formatBytes(part.size) }}</span>
              </header>
              <div v-if="part.headers.length" class="multipart-part-headers">
                <div v-for="[name, value] in part.headers" :key="name" class="multipart-part-header">
                  <span>{{ name }}</span>
                  <code>{{ value }}</code>
                </div>
              </div>
              <div v-if="part.imageSrc" class="body-image-view multipart-image-view">
                <img :src="part.imageSrc" :alt="part.title" />
              </div>
              <pre v-else class="body-view multipart-body-view">{{ part.body }}</pre>
            </section>
          </div>
          <pre v-else class="body-view" :class="{ 'body-view-formatted': bodyView.kind !== 'default' }">{{ bodyView.text }}</pre>
        </details>
      </section>

      <section v-else-if="selectedWebhook" class="blank-detail">
        <p class="overline">Ready</p>
        <h2>Waiting for requests</h2>
        <p>{{ displayWebhookName(selectedWebhook) }}</p>
      </section>

      <section v-else class="blank-detail">
        <p class="overline">Loading</p>
        <h2>Preparing your workspace</h2>
      </section>
    </main>

    <NoteTooltip :tooltip="requestNoteTooltip" />

    <SearchModal
      v-if="searchOpen"
      v-model:query="searchQuery"
      v-model:current-only="searchCurrentOnly"
      :selected-webhook="selectedWebhook"
      :loading="searchLoading"
      :results="searchResults"
      @close="closeSearchModal"
      @clear="clearSearchQuery"
      @schedule-search="scheduleSearch()"
      @open-first="openFirstSearchResult"
      @open-result="openSearchResult"
    />

    <RenameWebhookModal
      v-if="renameModalOpen"
      v-model:name="renameForm.name"
      :initial-name="renameInitialName"
      :max-symbols="maxWebhookNameSymbols"
      :saving="renameSaving"
      @close="closeRenameModal"
      @save="saveWebhookName"
    />

    <RequestNoteModal
      v-if="requestNoteModalOpen"
      v-model:note="requestNoteForm.note"
      :max-symbols="maxRequestNoteSymbols"
      :saving="requestNoteSaving"
      @close="closeRequestNoteModal"
      @save="saveRequestNote"
    />

    <ResponseSettingsModal
      v-if="responseModalOpen"
      :webhook-name="displayWebhookName(selectedWebhook)"
      :form="responseForm"
      :is-redirect="responseIsRedirect"
      :saving="responseSaving"
      @close="closeResponseModal"
      @save="saveResponseSettings"
      @add-header="addResponseHeader"
      @remove-header="removeResponseHeader"
    />

    <TelegramSettingsModal
      v-if="telegramModalOpen"
      :settings="telegramSettings"
      :form="telegramForm"
      :saving="telegramSaving"
      :testing="telegramTesting"
      @close="closeTelegramModal"
      @save="saveTelegramSettings"
      @test="testTelegramSettings"
    />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Check,
  ChevronDown,
  ClipboardCopy,
  Copy,
  Link2,
  StickyNote,
  Trash2
} from '@lucide/vue'
import AppSidebar from '../components/layout/AppSidebar.vue'
import SearchModal from '../components/modals/SearchModal.vue'
import RenameWebhookModal from '../components/modals/RenameWebhookModal.vue'
import RequestNoteModal from '../components/modals/RequestNoteModal.vue'
import ResponseSettingsModal from '../components/modals/ResponseSettingsModal.vue'
import TelegramSettingsModal from '../components/modals/TelegramSettingsModal.vue'
import NoteTooltip from '../components/ui/NoteTooltip.vue'
import { useBodyViewer } from '../composables/useBodyViewer'
import { useClipboard } from '../composables/useClipboard'
import { useNotice } from '../composables/useNotice'
import { useRequestNoteTooltip } from '../composables/useRequestNoteTooltip'
import * as webhookApi from '../services/webhookApi'
import {
  displayWebhookName,
  formatBreakableUri,
  formatBytes,
  formatReadableDate,
  formatTime,
  symbolCount
} from '../utils/formatters'
import {
  formatHTTPClipboardRequest,
  headerNameCopyTarget,
  headerValueCopyTarget,
  headerValueText
} from '../utils/requestFormatting'
import { ownerRootPath, parseRoute, requestDetailPath, shareWebhookPath } from '../utils/route'
import { loadStoredPrefs, loadStoredValue, saveStoredPrefs, saveStoredValue } from '../utils/storage'

const autoSwitchStorageKey = 'webhook-auto-switch'
const soundStorageKey = 'webhook-sound-alerts'
const selectedWebhookStorageKey = 'webhook-selected-slug'
const maxWebhookNameSymbols = 80
const maxRequestNoteSymbols = 200

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
const actionMenuOpen = ref(false)
const selectorOpen = ref(false)
const searchOpen = ref(false)
const searchQuery = ref('')
const searchResults = ref([])
const searchLoading = ref(false)
const searchCurrentOnly = ref(false)
const autoSwitchPrefs = ref(loadAutoSwitchPrefs())
const soundPrefs = ref(loadSoundPrefs())
const highlightedWebhookSlugs = ref([])
const deletingRequestId = ref('')
const renameModalOpen = ref(false)
const renameForm = ref({ name: '' })
const renameInitialName = ref('')
const renameSaving = ref(false)
const requestNoteModalOpen = ref(false)
const requestNoteForm = ref({ note: '' })
const requestNoteSaving = ref(false)
const responseModalOpen = ref(false)
const responseForm = ref({ body: '', contentType: '', statusCode: 200, location: '', headers: [] })
const responseSaving = ref(false)
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
let searchTimer = null
let searchAbortController = null
let audioContext = null
let soundQueue = 0
let playingSound = false
let responseHeaderId = 0

const { notice, showNotice } = useNotice()
const { copiedTarget, copy } = useClipboard(showNotice)
const {
  requestNoteTooltip,
  showRequestNoteTooltip,
  positionRequestNoteTooltip,
  hideRequestNoteTooltip
} = useRequestNoteTooltip()
const {
  enhancedBodyView,
  headerRows,
  bodyDisplay,
  hasBody,
  bodyHasEnhancedView,
  bodyView,
  toggleBodyView
} = useBodyViewer(selectedRequest)

const selectedWebhook = computed(() => webhooks.value.find((hook) => hook.slug === selectedSlug.value) || null)

const responseIsRedirect = computed(() => isRedirectStatus(responseForm.value.statusCode))

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
  if (searchTimer) window.clearTimeout(searchTimer)
  abortSearch()
  window.removeEventListener('click', closeMenus)
})

async function loadOwned() {
  session.value = await webhookApi.getSession()
  const data = await webhookApi.listWebhooks()
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
  const data = await webhookApi.getSharedWebhook(initialRoute.slug, shareToken)
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
    if (!session.value) session.value = await webhookApi.getSession()
    const data = await webhookApi.createWebhook(session.value)
    webhooks.value = [data.webhook, ...webhooks.value]
    setSelectedSlug(data.webhook.slug)
    requests.value = []
    selectedRequest.value = null
    history.replaceState({}, '', ownerRootPath())
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
    await webhookApi.deleteWebhook(selectedWebhook.value.slug, session.value)
    webhooks.value = webhooks.value.filter((hook) => hook.slug !== selectedWebhook.value.slug)
    setSelectedSlug(webhooks.value[0]?.slug || '')
    requests.value = []
    selectedRequest.value = null
    history.replaceState({}, '', ownerRootPath())
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
    const data = await webhookApi.setShareEnabled(selectedWebhook.value.slug, enabled, session.value)
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
    const data = await webhookApi.setWebhookTelegramEnabled(selectedWebhook.value.slug, enabled, session.value)
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

function openRenameModal() {
  if (!selectedWebhook.value || shareMode) return
  closeMenus()
  renameInitialName.value = selectedWebhook.value.slug
  renameForm.value = { name: displayWebhookName(selectedWebhook.value) }
  renameModalOpen.value = true
  error.value = ''
}

function closeRenameModal() {
  if (renameSaving.value) return
  renameModalOpen.value = false
}

async function saveWebhookName() {
  if (!selectedWebhook.value || shareMode) return
  const slug = selectedWebhook.value.slug
  const name = renameForm.value.name.trim()
  if (!name) {
    error.value = 'Webhook name is required'
    return
  }
  if (symbolCount(name) > maxWebhookNameSymbols) {
    error.value = `Webhook name must be at most ${maxWebhookNameSymbols} symbols`
    return
  }
  renameSaving.value = true
  error.value = ''
  try {
    const data = await webhookApi.setWebhookName(slug, name, session.value)
    webhooks.value = webhooks.value.map((hook) => (hook.slug === data.webhook.slug ? data.webhook : hook))
    renameModalOpen.value = false
    showNotice('Webhook renamed')
  } catch (err) {
    error.value = err.message
  } finally {
    renameSaving.value = false
  }
}

function openResponseModal() {
  if (!selectedWebhook.value || shareMode) return
  closeMenus()
  resetResponseForm()
  responseModalOpen.value = true
}

function closeResponseModal() {
  if (responseSaving.value) return
  responseModalOpen.value = false
}

function resetResponseForm() {
  responseForm.value = {
    body: selectedWebhook.value?.responseBody ?? 'ok\n',
    contentType: selectedWebhook.value?.responseContentType || 'text/plain; charset=utf-8',
    statusCode: selectedWebhook.value?.responseStatusCode || 200,
    location: selectedWebhook.value?.responseLocation || '',
    headers: (selectedWebhook.value?.responseHeaders || []).map((header) => ({
      id: nextResponseHeaderId(),
      name: header.name || '',
      value: header.value || ''
    }))
  }
}

async function saveResponseSettings() {
  if (!selectedWebhook.value || shareMode) return
  responseSaving.value = true
  error.value = ''
  try {
    const data = await webhookApi.saveWebhookResponse(selectedWebhook.value.slug, responsePayload(), session.value)
    webhooks.value = webhooks.value.map((hook) => (hook.slug === data.webhook.slug ? data.webhook : hook))
    responseModalOpen.value = false
    showNotice('Response settings saved')
  } catch (err) {
    error.value = err.message
  } finally {
    responseSaving.value = false
  }
}

function responsePayload() {
  return {
    body: responseForm.value.body,
    contentType: responseForm.value.contentType,
    statusCode: Number(responseForm.value.statusCode),
    location: responseIsRedirect.value ? responseForm.value.location : '',
    headers: responseForm.value.headers
      .filter((header) => header.name || header.value)
      .map((header) => ({ name: header.name, value: header.value }))
  }
}

function addResponseHeader() {
  responseForm.value.headers = [
    ...responseForm.value.headers,
    { id: nextResponseHeaderId(), name: '', value: '' }
  ]
}

function removeResponseHeader(id) {
  responseForm.value.headers = responseForm.value.headers.filter((header) => header.id !== id)
}

function nextResponseHeaderId() {
  responseHeaderId += 1
  return responseHeaderId
}

function isRedirectStatus(value) {
  const status = Number(value)
  return status >= 300 && status <= 399
}

async function selectWebhook(slug) {
  clearWebhookHighlight(slug)
  if (selectedSlug.value === slug) return
  setSelectedSlug(slug)
  selectedRequest.value = null
  requests.value = []
  history.replaceState({}, '', shareMode ? shareWebhookPath(slug, shareToken) : ownerRootPath())
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
  try {
    const data = await webhookApi.searchRequests({
      query,
      limit: 40,
      webhook: searchCurrentOnly.value ? selectedSlug.value : ''
    }, controller.signal)
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
    const data = await webhookApi.listRequests(slug, { shareMode, shareToken })
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
    const data = await webhookApi.listWebhooks()
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
  try {
    const data = await webhookApi.getRequest(slug, requestId, { shareMode, shareToken })
    return data.request || null
  } catch {
    return null
  }
}

function selectRequest(request) {
  hideRequestNoteTooltip()
  selectedRequest.value = request
  replaceRequestRoute(request)
}

async function deleteRequest(request = selectedRequest.value) {
  if (!selectedWebhook.value || !request || shareMode) return
  hideRequestNoteTooltip()
  const slug = selectedWebhook.value.slug
  const requestId = request.id
  const currentIndex = requests.value.findIndex((item) => item.id === requestId)
  const deletingSelected = selectedRequest.value?.id === requestId
  deletingRequestId.value = requestId
  error.value = ''
  try {
    await webhookApi.deleteRequest(slug, requestId, session.value)
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
        history.replaceState({}, '', ownerRootPath())
      }
    }
  } catch (err) {
    error.value = err.message
  } finally {
    if (deletingRequestId.value === requestId) deletingRequestId.value = ''
  }
}

function openRequestNoteModal() {
  if (!selectedRequest.value || !selectedWebhook.value || shareMode) return
  requestNoteForm.value = { note: selectedRequest.value.note || '' }
  requestNoteModalOpen.value = true
  error.value = ''
}

function closeRequestNoteModal() {
  if (requestNoteSaving.value) return
  requestNoteModalOpen.value = false
}

async function saveRequestNote() {
  if (!selectedRequest.value || !selectedWebhook.value || shareMode) return
  const slug = selectedWebhook.value.slug
  const requestId = selectedRequest.value.id
  const note = requestNoteForm.value.note.trim()
  if (symbolCount(note) > maxRequestNoteSymbols) {
    error.value = `Request note must be at most ${maxRequestNoteSymbols} symbols`
    return
  }
  requestNoteSaving.value = true
  error.value = ''
  try {
    const data = await webhookApi.saveRequestNote(slug, requestId, note, session.value)
    if (selectedWebhook.value?.slug === slug) {
      updateRequestInState(data.request)
    }
    requestNoteModalOpen.value = false
    showNotice(note ? 'Note saved' : 'Note removed')
  } catch (err) {
    error.value = err.message
  } finally {
    requestNoteSaving.value = false
  }
}

function updateRequestInState(updatedRequest) {
  if (!updatedRequest?.id) return
  requests.value = requests.value.map((request) => (request.id === updatedRequest.id ? updatedRequest : request))
  if (selectedRequest.value?.id === updatedRequest.id) {
    selectedRequest.value = updatedRequest
  }
}

function replaceRequestRoute(request) {
  if (!request || !selectedWebhook.value) return
  history.replaceState({}, '', requestDetailPath({
    shareMode,
    slug: selectedWebhook.value.slug,
    requestId: request.id,
    shareToken
  }))
}

function startPolling() {
  if (pollTimer) window.clearInterval(pollTimer)
  pollTimer = window.setInterval(() => {
    if (selectedWebhook.value) loadRequests()
    if (!shareMode) loadWebhookSummaries()
  }, 3000)
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
    const data = await webhookApi.getTelegramSettings()
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
    const data = await webhookApi.saveTelegramSettings(telegramPayload(), session.value)
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
    await webhookApi.testTelegramSettings(telegramPayload(), session.value, controller.signal)
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
  hideRequestNoteTooltip()
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

function loadSelectedWebhookSlug() {
  return loadStoredValue(selectedWebhookStorageKey)
}

function setSelectedSlug(slug, options = {}) {
  selectedSlug.value = slug
  if (options.persist === false || shareMode) return
  saveSelectedWebhookSlug(slug)
}

function saveSelectedWebhookSlug(slug) {
  saveStoredValue(selectedWebhookStorageKey, slug)
}

function saveAutoSwitchPrefs(value) {
  saveStoredPrefs(autoSwitchStorageKey, value)
}

function saveSoundPrefs(value) {
  saveStoredPrefs(soundStorageKey, value)
}

</script>
