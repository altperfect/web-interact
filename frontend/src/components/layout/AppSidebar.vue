<template>
  <aside class="left-rail" aria-label="Webhook navigation">
    <header class="brand-row">
      <div>
        <p class="overline">Webhook Console</p>
        <h1>{{ shareMode ? 'Shared stream' : 'Request capture' }}</h1>
      </div>
      <div v-if="!shareMode" class="brand-actions">
        <button class="new-button" type="button" :disabled="busy" @click.stop="$emit('create-webhook')">
          <Plus :size="15" :stroke-width="1.8" aria-hidden="true" />
          New
        </button>
        <button class="icon-button telegram-settings-trigger" type="button" title="Telegram settings" @click.stop="$emit('open-telegram-modal')">
          <Send :size="15" :stroke-width="1.8" aria-hidden="true" />
        </button>
      </div>
    </header>

    <section v-if="selectedWebhook" class="current-hook">
      <div class="hook-main">
        <div class="hook-title">
          <span class="hook-name">{{ displayWebhookName(selectedWebhook) }}</span>
          <span class="hook-count">{{ requestCountLabel }}</span>
        </div>

        <div class="hook-actions">
          <button class="icon-button" type="button" title="Copy endpoint" @click.stop="$emit('copy', selectedWebhook.url, 'endpoint')">
            <Transition name="icon-fade" mode="out-in">
              <Check v-if="copiedTarget === 'endpoint'" key="endpoint-check" :size="16" :stroke-width="1.9" aria-hidden="true" />
              <Copy v-else key="endpoint-copy" :size="16" :stroke-width="1.85" aria-hidden="true" />
            </Transition>
          </button>

          <div v-if="!shareMode" class="menu-wrap">
            <button class="icon-button" type="button" title="Webhook actions" @click.stop="$emit('toggle-action-menu')">
              <MoreHorizontal :size="17" :stroke-width="1.8" aria-hidden="true" />
            </button>
            <div v-if="actionMenuOpen" class="popover action-menu" @click.stop>
              <button type="button" @click="$emit('toggle-auto-switch')">
                <RotateCw :size="15" :stroke-width="1.8" aria-hidden="true" />
                {{ selectedAutoSwitch ? 'Auto-switch enabled' : 'Auto-switch disabled' }}
              </button>
              <button type="button" @click="$emit('toggle-sound-notifications')">
                <Volume2 v-if="selectedSoundEnabled" :size="15" :stroke-width="1.8" aria-hidden="true" />
                <VolumeX v-else :size="15" :stroke-width="1.8" aria-hidden="true" />
                {{ selectedSoundEnabled ? 'Sound enabled' : 'Sound disabled' }}
              </button>
              <button type="button" @click="$emit('toggle-webhook-telegram')">
                <Send :size="15" :stroke-width="1.8" aria-hidden="true" />
                {{ selectedWebhook.telegramEnabled ? 'Telegram enabled' : 'Telegram disabled' }}
              </button>
              <button type="button" @click="$emit('open-response-modal')">
                <Settings2 :size="15" :stroke-width="1.8" aria-hidden="true" />
                Response settings
              </button>
              <button type="button" @click="$emit('open-rename-modal')">
                <Pencil :size="15" :stroke-width="1.8" aria-hidden="true" />
                Rename webhook
              </button>
              <button type="button" @click="$emit('set-share-enabled', !selectedWebhook.shareEnabled)">
                <Share2 :size="15" :stroke-width="1.8" aria-hidden="true" />
                {{ selectedWebhook.shareEnabled ? 'Disable guest access' : 'Enable guest access' }}
              </button>
              <button
                v-if="selectedWebhook.shareEnabled && selectedWebhook.shareUrl"
                type="button"
                @click="$emit('copy', selectedWebhook.shareUrl, 'share')"
              >
                <Transition name="icon-fade" mode="out-in">
                  <Check v-if="copiedTarget === 'share'" key="menu-share-check" :size="15" :stroke-width="1.9" aria-hidden="true" />
                  <Link2 v-else key="menu-share-link" :size="15" :stroke-width="1.8" aria-hidden="true" />
                </Transition>
                Copy share link
              </button>
              <button class="menu-danger" type="button" @click="$emit('delete-webhook')">
                <Trash2 :size="15" :stroke-width="1.8" aria-hidden="true" />
                Delete webhook
              </button>
            </div>
          </div>

          <div class="menu-wrap">
            <button class="icon-button" type="button" title="Select webhook" @click.stop="$emit('toggle-selector')">
              <span v-if="highlightedWebhookSlugs.length" class="webhook-alert-dot selector-alert-dot" aria-hidden="true"></span>
              <ChevronDown :size="16" :stroke-width="1.8" aria-hidden="true" />
            </button>
            <div v-if="selectorOpen" class="popover selector-menu" @click.stop>
              <button
                v-for="webhook in webhooks"
                :key="webhook.slug"
                type="button"
                :class="{ active: selectedSlug === webhook.slug, attention: highlightedWebhookSlugs.includes(webhook.slug) }"
                @pointerdown.stop="$emit('select-webhook', webhook.slug)"
                @click.stop="$emit('select-webhook', webhook.slug)"
              >
                <span class="selector-name">
                  <span>{{ displayWebhookName(webhook) }}</span>
                  <span v-if="highlightedWebhookSlugs.includes(webhook.slug)" class="selector-badge">New</span>
                </span>
                <small>{{ selectorWebhookMeta(webhook) }}</small>
              </button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="selectedWebhook.shareEnabled && selectedWebhook.shareUrl" class="share-strip">
        <span>Guest access</span>
        <button type="button" @click.stop="$emit('copy', selectedShareUrl, 'share')">
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
      <button v-if="!shareMode" type="button" @click.stop="$emit('create-webhook')">Create webhook</button>
    </section>

    <section class="request-nav" aria-label="Captured requests">
      <div class="rail-section-head">
        <span>Requests</span>
        <button v-if="!shareMode" class="icon-button search-trigger" type="button" title="Search requests" @click.stop="$emit('open-search-modal')">
          <Search :size="15" :stroke-width="1.8" aria-hidden="true" />
        </button>
      </div>

      <div
        class="request-scroll"
        :class="{ 'request-scroll-empty': selectedWebhook && requests.length === 0 && !showRequestListSkeleton }"
        @scroll.passive="$emit('hide-note-tooltip')"
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
            @click.stop="$emit('select-request', request)"
            @keydown.enter.prevent="$emit('select-request', request)"
            @keydown.space.prevent="$emit('select-request', request)"
          >
            <span class="request-method">{{ request.method }}</span>
            <span class="request-path">
              <span>{{ formatRequestPath(request, selectedWebhook?.slug || '') }}</span>
            </span>
            <span class="request-row-actions">
              <span
                v-if="request.note && !shareMode"
                class="request-note-indicator"
                aria-label="Request note"
                tabindex="0"
                @pointerenter.stop="$emit('show-note-tooltip', $event, request.note)"
                @pointermove.stop="$emit('position-note-tooltip', $event)"
                @pointerleave.stop="$emit('hide-note-tooltip')"
                @focus="$emit('show-note-tooltip', $event, request.note)"
                @blur="$emit('hide-note-tooltip')"
              >
                <StickyNote :size="13" :stroke-width="1.85" aria-hidden="true" />
              </span>
              <span class="request-time">{{ formatTime(request.createdAt) }}</span>
              <span v-if="!shareMode" class="request-delete-slot">
                <button
                  v-if="selectedRequest?.id === request.id"
                  class="request-delete-button"
                  type="button"
                  title="Delete request"
                  :disabled="deletingRequestId === request.id"
                  @click.stop="$emit('delete-request', request)"
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
</template>

<script setup>
import {
  Check,
  ChevronDown,
  Copy,
  Link2,
  MoreHorizontal,
  Pencil,
  Plus,
  RotateCw,
  Search,
  Send,
  Settings2,
  Share2,
  StickyNote,
  Trash2,
  Volume2,
  VolumeX
} from '@lucide/vue'
import { displayWebhookName, formatRequestPath, formatTime, selectorWebhookMeta } from '../../utils/formatters'

defineProps({
  shareMode: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  selectedWebhook: { type: Object, default: null },
  selectedSlug: { type: String, default: '' },
  webhooks: { type: Array, default: () => [] },
  requests: { type: Array, default: () => [] },
  selectedRequest: { type: Object, default: null },
  loading: { type: Boolean, default: false },
  showRequestListSkeleton: { type: Boolean, default: false },
  actionMenuOpen: { type: Boolean, default: false },
  selectorOpen: { type: Boolean, default: false },
  requestCountLabel: { type: String, default: '' },
  selectedAutoSwitch: { type: Boolean, default: true },
  selectedSoundEnabled: { type: Boolean, default: false },
  selectedShareUrl: { type: String, default: '' },
  copiedTarget: { type: String, default: '' },
  deletingRequestId: { type: String, default: '' },
  highlightedWebhookSlugs: { type: Array, default: () => [] }
})

defineEmits([
  'create-webhook',
  'open-telegram-modal',
  'copy',
  'toggle-action-menu',
  'toggle-auto-switch',
  'toggle-sound-notifications',
  'toggle-webhook-telegram',
  'open-response-modal',
  'open-rename-modal',
  'set-share-enabled',
  'delete-webhook',
  'toggle-selector',
  'select-webhook',
  'open-search-modal',
  'hide-note-tooltip',
  'select-request',
  'show-note-tooltip',
  'position-note-tooltip',
  'delete-request'
])
</script>
