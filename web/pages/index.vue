<script setup lang="ts">
import {
  checkAvailability,
  downloadTrack,
  formatDuration,
  getFileUrl,
  getMetadata,
  searchTracks,
} from "~/lib/api/client"
import type { MetadataResult, TrackMetadata } from "~/lib/api/types"
import {
  buildTrackKey,
  getCollectionArtwork,
  getCollectionLabel,
  getCollectionSubtitle,
  getCollectionTitle,
  isSupportedUrl,
  metadataToTracks,
} from "~/lib/downloader"

type Provider = "auto" | "tidal" | "qobuz" | "amazon"
type SearchSource = "spotify" | "deezer"
type NoticeTone = "neutral" | "success" | "error"

type DownloadState = {
  status: "idle" | "downloading" | "success" | "error"
  fileName?: string
  error?: string
  itemId?: string
  progress?: number
  detail?: string
}

const providerOptions: { label: string; value: Provider; caption: string }[] = [
  { label: "Auto", value: "auto", caption: "smart routing" },
  { label: "Tidal", value: "tidal", caption: "max quality" },
  { label: "Qobuz", value: "qobuz", caption: "hi-res alt" },
  { label: "Amazon", value: "amazon", caption: "fallback" },
]

const sourceOptions: { label: string; value: SearchSource; caption: string }[] = [
  { label: "Spotify", value: "spotify", caption: "search source" },
  { label: "Deezer", value: "deezer", caption: "search source" },
]

const query = ref("")
const metadata = ref<MetadataResult | null>(null)
const searchResults = ref<TrackMetadata[]>([])
const loading = ref(false)
const batchRunning = ref(false)
const provider = ref<Provider>("auto")
const searchSource = ref<SearchSource>("spotify")
const notice = ref<{ tone: NoticeTone; text: string }>({
  tone: "neutral",
  text: "Search by title or paste a link, then move straight to download.",
})
const downloadState = ref<Record<string, DownloadState>>({})
const searchRequestSeq = ref(0)

useSeoMeta({
  title: "FLAC by Zumy",
  description: "Brutalist FLAC download desk for fast search, link resolve, and direct capture.",
})

const trimmedQuery = computed(() => query.value.trim())
const queryLooksLikeUrl = computed(() => isSupportedUrl(trimmedQuery.value))
const tracks = computed(() => metadataToTracks(metadata.value))
const collectionTitle = computed(() => getCollectionTitle(metadata.value))
const collectionSubtitle = computed(() => getCollectionSubtitle(metadata.value))
const collectionArtwork = computed(() => getCollectionArtwork(metadata.value))
const collectionLabel = computed(() => getCollectionLabel(metadata.value))

function setNotice(tone: NoticeTone, text: string) {
  notice.value = { tone, text }
}

function clearWorkspace() {
  query.value = ""
  metadata.value = null
  searchResults.value = []
  downloadState.value = {}
  setNotice("neutral", "Fresh desk. Search again or paste another link.")
}

async function resolveMetadataFromInput(input: string) {
  loading.value = true
  searchResults.value = []
  setNotice("neutral", "Resolving link metadata…")

  try {
    const result = await getMetadata(input)
    metadata.value = result
    setNotice("success", "Metadata resolved. Review the tracks below.")
  } catch (error) {
    metadata.value = null
    setNotice("error", error instanceof Error ? error.message : "Failed to resolve metadata")
  } finally {
    loading.value = false
  }
}

async function onSubmit() {
  if (!trimmedQuery.value) {
    setNotice("error", "Enter a link or search query first.")
    return
  }

  if (queryLooksLikeUrl.value) {
    await resolveMetadataFromInput(trimmedQuery.value)
    return
  }

  const requestId = ++searchRequestSeq.value
  loading.value = true
  metadata.value = null
  setNotice("neutral", `Searching ${searchSource.value}…`)

  try {
    const result = await searchTracks(trimmedQuery.value, searchSource.value, 20)
    if (requestId !== searchRequestSeq.value) return

    searchResults.value = result.tracks
    setNotice(
      result.tracks.length ? "success" : "neutral",
      result.tracks.length
        ? `Found ${result.tracks.length} result${result.tracks.length > 1 ? "s" : ""}. Pick one to continue.`
        : "No tracks found. Try another title or use a direct link.",
    )
  } catch (error) {
    if (requestId !== searchRequestSeq.value) return
    searchResults.value = []
    setNotice("error", error instanceof Error ? error.message : "Search failed")
  } finally {
    if (requestId === searchRequestSeq.value) loading.value = false
  }
}

function selectTrack(track: TrackMetadata) {
  metadata.value = { type: "track", track }
  searchResults.value = []
  setNotice("success", `${track.title} selected and ready to download.`)
}

function stateFor(track: TrackMetadata): DownloadState {
  return downloadState.value[buildTrackKey(track)] ?? { status: "idle" }
}

function stateLabel(track: TrackMetadata): string {
  const state = stateFor(track)
  if (state.status === "downloading") {
    if (typeof state.progress === "number" && state.progress > 0 && state.progress < 100) {
      return `${state.progress}%`
    }
    return "Working"
  }
  if (state.status === "success") return "Save file"
  if (state.status === "error") return "Retry"
  return "Download"
}

function statusLabel(track: TrackMetadata): string {
  const state = stateFor(track)
  if (state.status === "downloading") return state.detail || "in transfer"
  if (state.status === "success") return "ready"
  if (state.status === "error") return "failed"
  return "waiting"
}

function fileUrlFor(track: TrackMetadata): string {
  const fileName = stateFor(track).fileName
  return fileName ? getFileUrl(fileName) : "#"
}

function wait(ms: number) {
  return new Promise((resolve) => globalThis.setTimeout(resolve, ms))
}

async function pollDownloadProgress(key: string, itemId: string, control: { active: boolean }) {
  while (control.active) {
    try {
      const response = await getProgress(itemId)
      const entry = response[itemId]

      if (entry && downloadState.value[key]?.status === "downloading") {
        const rawProgress = typeof entry.progress === "number" ? entry.progress : 0
        const percentage = rawProgress <= 1 ? Math.round(rawProgress * 100) : Math.round(rawProgress)
        const normalizedProgress = Math.max(0, Math.min(100, percentage))
        const rawStatus = typeof entry.status === "string" ? entry.status : "downloading"

        downloadState.value = {
          ...downloadState.value,
          [key]: {
            ...downloadState.value[key],
            status: "downloading",
            progress: normalizedProgress,
            detail: rawStatus === "finalizing" ? "finalizing" : `${normalizedProgress}% transferred`,
          },
        }
      }
    } catch {
      // ignore polling flakes and keep transfer active
    }

    if (!control.active) break
    await wait(1200)
  }
}

async function downloadSingle(track: TrackMetadata) {
  const key = buildTrackKey(track)
  const itemId = globalThis.crypto?.randomUUID?.() ?? buildTrackKey(track)

  downloadState.value = {
    ...downloadState.value,
    [key]: { status: "downloading", itemId, progress: 0, detail: "connecting" },
  }

  const pollControl = { active: true }
  const progressPoll = pollDownloadProgress(key, itemId, pollControl)

  try {
    let service: Provider = provider.value
    let useFallback = false
    const resolvedSpotifyId = track.spotify_id || (track.deezer_id ? `deezer:${track.deezer_id}` : undefined)

    if (service === "auto") {
      if (!resolvedSpotifyId && !track.deezer_id && !track.isrc) {
        throw new Error("Track is missing IDs for smart routing. Pick a provider manually.")
      }

      const availability = await checkAvailability(track.spotify_id, track.isrc, track.deezer_id)
      if (availability.tidal) service = "tidal"
      else if (availability.qobuz) service = "qobuz"
      else if (availability.amazon) service = "amazon"
      else throw new Error("This track is not available on any configured provider.")
      useFallback = true
    }

    const response = await downloadTrack({
      track_name: track.title,
      artist_name: track.artist,
      album_name: track.album,
      album_artist: track.album_artist,
      cover_url: track.cover_url,
      spotify_id: resolvedSpotifyId,
      deezer_id: track.deezer_id,
      isrc: track.isrc,
      service,
      track_number: track.track_number,
      disc_number: track.disc_number,
      total_tracks: track.total_tracks,
      total_discs: track.total_discs,
      release_date: track.release_date,
      source: track.source,
      item_id: itemId,
      duration_ms: track.duration_ms,
      use_fallback: useFallback,
    })

    if (!response.success || !response.file_path) {
      throw new Error(response.error || "Download failed")
    }

    const fileName = response.file_name || response.file_path.split("/").pop() || response.file_path
    downloadState.value = {
      ...downloadState.value,
      [key]: { status: "success", fileName, itemId, progress: 100, detail: "ready" },
    }
    setNotice("success", `${track.title} is ready to save.`)
  } catch (error) {
    downloadState.value = {
      ...downloadState.value,
      [key]: {
        status: "error",
        error: error instanceof Error ? error.message : "Download failed",
        itemId,
      },
    }
    setNotice("error", error instanceof Error ? error.message : "Download failed")
  } finally {
    pollControl.active = false
    await progressPoll
  }
}

async function downloadAll() {
  if (!tracks.value.length) return

  batchRunning.value = true
  setNotice("neutral", `Starting sequential download for ${tracks.value.length} track${tracks.value.length > 1 ? "s" : ""}…`)

  try {
    for (const track of tracks.value) {
      await downloadSingle(track)
    }
  } finally {
    batchRunning.value = false
  }
}
</script>

<template>
  <main class="zumy-page">
    <section class="zumy-hero">
      <div class="zumy-hero__copy">
        <div class="zumy-block zumy-block--yellow" />
        <p class="zumy-mark">FLAC BY ZUMY</p>
        <h1 class="zumy-title">
          PURE
          <br />
          UNCUT
          <br />
          AUDIO.
        </h1>
        <p class="zumy-copy">
          Download master-quality FLAC directly from Tidal, Qobuz, and Amazon Music. No upsampling, no compression.
          Just the truth.
        </p>

        <div class="zumy-actions">
          <button type="button" class="zumy-button zumy-button--primary" :disabled="loading" @click="onSubmit">
            {{ loading ? "Working…" : queryLooksLikeUrl ? "Resolve link" : "Search catalog" }}
          </button>
          <button type="button" class="zumy-button zumy-button--ghost" @click="clearWorkspace">
            Reset desk
          </button>
        </div>
      </div>

      <div class="zumy-hero__art">
        <div class="zumy-artframe">
          <img v-if="collectionArtwork" :src="collectionArtwork" alt="" loading="lazy" />
          <div v-else class="zumy-artframe__fallback">
            <span>FLAC</span>
          </div>
          <div class="zumy-artframe__tag">24-BIT / 192kHz</div>
        </div>
      </div>
    </section>

    <section class="zumy-search">
      <div class="zumy-topbar">
        <div class="zumy-topbar__brand">FLAC BY ZUMY</div>
        <div class="zumy-topbar__links">
          <span>Tidal</span>
          <span>Qobuz</span>
          <span>Amazon</span>
        </div>
        <NuxtLink to="/docs" class="zumy-topbar__docs">API atlas</NuxtLink>
      </div>

      <div class="zumy-search__input">
        <span class="zumy-search__icon">⌕</span>
        <textarea
          v-model="query"
          rows="3"
          class="zumy-search__field"
          placeholder="ARTIST, ALBUM, TRACK OR PASTE A LINK"
        />
        <button type="button" class="zumy-search__submit" :disabled="loading" @click="onSubmit">
          FIND
        </button>
      </div>

      <div class="zumy-filters">
        <button
          v-for="option in providerOptions"
          :key="option.value"
          type="button"
          class="zumy-filter"
          :data-active="provider === option.value"
          @click="provider = option.value"
        >
          {{ option.label }}
        </button>
      </div>

      <div class="zumy-filters zumy-filters--secondary">
        <button
          v-for="option in sourceOptions"
          :key="option.value"
          type="button"
          class="zumy-filter zumy-filter--light"
          :data-active="searchSource === option.value"
          @click="searchSource = option.value"
        >
          {{ option.label }}
        </button>
      </div>

      <div class="zumy-notice" :data-tone="notice.tone">
        {{ notice.text }}
      </div>
    </section>

    <section v-if="searchResults.length" class="zumy-section">
      <div class="zumy-section__head">
        <h2>SEARCH RESULTS</h2>
        <span>{{ searchResults.length }} HITS</span>
      </div>

      <div class="zumy-card-grid">
        <button
          v-for="track in searchResults"
          :key="buildTrackKey(track)"
          type="button"
          class="zumy-result-card"
          @click="selectTrack(track)"
        >
          <div class="zumy-result-card__cover">
            <img v-if="track.cover_url" :src="track.cover_url" alt="" loading="lazy" />
            <div v-else class="zumy-result-card__cover--fallback">FLAC</div>
          </div>
          <div class="zumy-result-card__copy">
            <strong>{{ track.title }}</strong>
            <p>{{ track.artist }}</p>
            <small v-if="track.album">{{ track.album }}</small>
          </div>
          <div class="zumy-result-card__meta">
            <span v-if="track.duration_ms">{{ formatDuration(track.duration_ms) }}</span>
            <span class="zumy-mini-chip">SELECT</span>
          </div>
        </button>
      </div>
    </section>

    <section v-if="tracks.length" class="zumy-section">
      <div class="zumy-section__head">
        <div>
          <h2>{{ collectionTitle }}</h2>
          <p>{{ collectionLabel }} · {{ collectionSubtitle }}</p>
        </div>
        <button
          v-if="tracks.length > 1"
          type="button"
          class="zumy-button zumy-button--primary"
          :disabled="batchRunning"
          @click="downloadAll"
        >
          {{ batchRunning ? "Downloading…" : `Download all ${tracks.length}` }}
        </button>
      </div>

      <div class="zumy-list">
        <article
          v-for="track in tracks"
          :key="buildTrackKey(track)"
          class="zumy-list__item"
          :data-status="stateFor(track).status"
        >
          <div class="zumy-list__cover">
            <img v-if="track.cover_url" :src="track.cover_url" alt="" loading="lazy" />
            <div v-else class="zumy-list__cover--fallback">FLAC</div>
          </div>

          <div class="zumy-list__copy">
            <strong>{{ track.title }}</strong>
            <p>{{ track.artist }}</p>
            <small>
              <template v-if="track.album">{{ track.album }}</template>
              <template v-if="track.duration_ms"> · {{ formatDuration(track.duration_ms) }}</template>
              <template v-if="track.isrc"> · {{ track.isrc }}</template>
            </small>
            <em v-if="stateFor(track).status === 'error'">{{ stateFor(track).error || "Download failed" }}</em>
            <span v-else-if="stateFor(track).status === 'downloading'">{{ statusLabel(track) }}</span>
          </div>

          <div class="zumy-list__actions">
            <a
              v-if="stateFor(track).status === 'success' && stateFor(track).fileName"
              :href="fileUrlFor(track)"
              class="zumy-button zumy-button--ghost zumy-button--small"
            >
              Save
            </a>
            <button
              v-else
              type="button"
              class="zumy-button zumy-button--ghost zumy-button--small"
              :disabled="stateFor(track).status === 'downloading' || batchRunning"
              @click="downloadSingle(track)"
            >
              {{ stateLabel(track) }}
            </button>
            <span class="zumy-mini-chip" :data-status="stateFor(track).status">{{ statusLabel(track) }}</span>
          </div>
        </article>
      </div>
    </section>
  </main>
</template>
