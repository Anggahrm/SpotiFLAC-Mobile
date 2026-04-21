<script setup lang="ts">
import {
  checkAvailability,
  downloadTrack,
  formatDuration,
  getFileUrl,
  getMetadata,
  getProgress,
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
  { label: "Auto", value: "auto", caption: "smart fallback" },
  { label: "Tidal", value: "tidal", caption: "premium stream" },
  { label: "Qobuz", value: "qobuz", caption: "hi-res alt" },
  { label: "Amazon", value: "amazon", caption: "legacy fallback" },
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
const queuedTrackKeys = ref<string[]>([])
const queuePaused = ref(false)
const notice = ref<{ tone: NoticeTone; text: string }>({
  tone: "neutral",
  text: "Paste a link or search by title, then move straight into the queue. No streaming clone, just a sharp download desk.",
})
const downloadState = ref<Record<string, DownloadState>>({})
const searchRequestSeq = ref(0)
const playerTrackKey = ref<string | null>(null)
const playerCurrentTime = ref(0)
const playerDuration = ref(0)
const playerPlaying = ref(false)
let playerAudio: HTMLAudioElement | null = null

useSeoMeta({
  title: "Signal Desk",
  description: "Brutalist FLAC download workspace for fast link resolution, search, and queue tracking.",
})

const trimmedQuery = computed(() => query.value.trim())
const queryLooksLikeUrl = computed(() => isSupportedUrl(trimmedQuery.value))
const tracks = computed(() => metadataToTracks(metadata.value))
const collectionLabel = computed(() => getCollectionLabel(metadata.value))
const collectionTitle = computed(() => getCollectionTitle(metadata.value))
const collectionSubtitle = computed(() => getCollectionSubtitle(metadata.value))
const collectionArtwork = computed(() => getCollectionArtwork(metadata.value))

const fieldNotes = computed(() => [
  { label: "input", value: queryLooksLikeUrl.value ? "link resolve" : "title search" },
  { label: "route", value: provider.value === "auto" ? "smart queue" : provider.value },
  { label: "surface", value: "mobile to desktop" },
])

const resultLead = computed(() => searchResults.value[0] ?? null)
const resultGrid = computed(() => searchResults.value.slice(1))

const trackCards = computed(() =>
  tracks.value.map((track) => ({
    key: buildTrackKey(track),
    track,
    state: downloadState.value[buildTrackKey(track)] ?? { status: "idle" as const },
  })),
)

const activeDownloadCards = computed(() =>
  trackCards.value.filter((entry) => entry.state.status === "downloading"),
)

const readyDownloadCards = computed(() =>
  trackCards.value.filter((entry) => entry.state.status === "success"),
)

const queuedTrackCards = computed(() =>
  queuedTrackKeys.value
    .map((key) => trackCards.value.find((entry) => entry.key === key))
    .filter((entry): entry is (typeof trackCards.value)[number] => Boolean(entry) && entry.state.status !== "downloading"),
)

const playerTrackCard = computed(() => {
  if (playerTrackKey.value) {
    return trackCards.value.find((entry) => entry.key === playerTrackKey.value) ?? null
  }
  return trackCards.value[0] ?? null
})

const playerTrack = computed(() => playerTrackCard.value?.track ?? null)

const playerSource = computed(() => {
  const track = playerTrack.value
  if (!track) return ""
  const state = stateFor(track)
  if (state.fileName) return getFileUrl(state.fileName)
  if (track.preview_url) return track.preview_url
  return ""
})

const playerSourceLabel = computed(() => {
  const track = playerTrack.value
  if (!track) return "Select a track from the desk"
  const state = stateFor(track)
  if (state.fileName) return "Downloaded file playback"
  if (track.preview_url) return "Provider preview stream"
  return "No preview available until the file is ready"
})

function setNotice(tone: NoticeTone, text: string) {
  notice.value = { tone, text }
}

function clearWorkspace() {
  query.value = ""
  metadata.value = null
  searchResults.value = []
  downloadState.value = {}
  queuedTrackKeys.value = []
  queuePaused.value = false
  stopPlayback()
  playerTrackKey.value = null
  setNotice("neutral", "Fresh desk. Drop a new link or run another search.")
}

async function resolveMetadataFromInput(input: string) {
  loading.value = true
  searchResults.value = []
  setNotice("neutral", "Resolving the pasted link into a clean track set…")

  try {
    const result = await getMetadata(input)
    metadata.value = result
    setNotice("success", "Metadata locked in. Review the track sheet and start the queue.")
  } catch (error) {
    metadata.value = null
    setNotice("error", error instanceof Error ? error.message : "Failed to resolve metadata")
  } finally {
    loading.value = false
  }
}

async function onSubmit() {
  if (!trimmedQuery.value) {
    setNotice("error", "Enter a Spotify/Deezer link or a search query first.")
    return
  }

  if (queryLooksLikeUrl.value) {
    await resolveMetadataFromInput(trimmedQuery.value)
    return
  }

  const requestId = ++searchRequestSeq.value
  loading.value = true
  metadata.value = null
  setNotice("neutral", `Searching ${searchSource.value} for the cleanest match…`)

  try {
    const result = await searchTracks(trimmedQuery.value, searchSource.value, 20)
    if (requestId !== searchRequestSeq.value) return

    searchResults.value = result.tracks
    setNotice(
      result.tracks.length ? "success" : "neutral",
      result.tracks.length
        ? `Found ${result.tracks.length} result${result.tracks.length > 1 ? "s" : ""}. Pick one to move it into the desk.`
        : "No tracks found. Try another title or switch to a direct link.",
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
  focusPlayer(track)
  setNotice("success", `Selected ${track.title}. The track is now staged for download.`)
}

function stateFor(track: TrackMetadata): DownloadState {
  return downloadState.value[buildTrackKey(track)] ?? { status: "idle" }
}

function isQueued(track: TrackMetadata) {
  return queuedTrackKeys.value.includes(buildTrackKey(track))
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
  if (isQueued(track)) return "Queued"
  return "Download"
}

function statusLabel(track: TrackMetadata): string {
  const state = stateFor(track)
  if (state.status === "downloading") return state.detail || "connecting"
  if (state.status === "success") return "ready"
  if (state.status === "error") return "needs retry"
  if (isQueued(track)) return queuePaused.value ? "queued · paused" : "queued"
  return "queued"
}

function fileUrlFor(track: TrackMetadata): string {
  const fileName = stateFor(track).fileName
  return fileName ? getFileUrl(fileName) : "#"
}

function wait(ms: number) {
  return new Promise((resolve) => globalThis.setTimeout(resolve, ms))
}

function ensurePlayerAudio() {
  if (!import.meta.client) return null
  if (playerAudio) return playerAudio

  playerAudio = new Audio()
  playerAudio.preload = "metadata"
  playerAudio.addEventListener("timeupdate", () => {
    playerCurrentTime.value = playerAudio?.currentTime ?? 0
  })
  playerAudio.addEventListener("loadedmetadata", () => {
    playerDuration.value = Number.isFinite(playerAudio?.duration) ? (playerAudio?.duration ?? 0) : 0
  })
  playerAudio.addEventListener("pause", () => {
    playerPlaying.value = false
  })
  playerAudio.addEventListener("play", () => {
    playerPlaying.value = true
  })
  playerAudio.addEventListener("ended", () => {
    playerPlaying.value = false
  })

  return playerAudio
}

function stopPlayback() {
  if (!playerAudio) return
  playerAudio.pause()
  playerCurrentTime.value = 0
  playerDuration.value = Number.isFinite(playerAudio.duration) ? playerAudio.duration : 0
  playerPlaying.value = false
}

function focusPlayer(track: TrackMetadata) {
  playerTrackKey.value = buildTrackKey(track)
}

function syncPlayerSource(autoplay = false) {
  const audio = ensurePlayerAudio()
  const source = playerSource.value
  if (!audio) return

  if (!source) {
    audio.pause()
    audio.removeAttribute("src")
    audio.load()
    playerCurrentTime.value = 0
    playerDuration.value = 0
    playerPlaying.value = false
    return
  }

  const currentSrc = audio.currentSrc || audio.src
  if (currentSrc === source) return

  audio.pause()
  audio.src = source
  audio.load()
  playerCurrentTime.value = 0
  playerDuration.value = 0
  playerPlaying.value = false

  if (autoplay) {
    void audio.play().catch(() => {
      playerPlaying.value = false
    })
  }
}

async function togglePlayback() {
  const audio = ensurePlayerAudio()
  const source = playerSource.value
  if (!audio || !source) {
    setNotice("error", "No preview is available for this track yet.")
    return
  }

  syncPlayerSource(false)

  if (audio.paused) {
    try {
      await audio.play()
    } catch (error) {
      setNotice("error", error instanceof Error ? error.message : "Failed to start playback")
    }
  } else {
    audio.pause()
  }
}

function seekPlayback(event: Event) {
  const audio = ensurePlayerAudio()
  if (!audio) return
  const target = event.target as HTMLInputElement
  const nextTime = Number(target.value)
  if (Number.isFinite(nextTime)) {
    audio.currentTime = nextTime
    playerCurrentTime.value = nextTime
  }
}

async function pollDownloadProgress(
  key: string,
  itemId: string,
  control: { active: boolean },
) {
  while (control.active) {
    try {
      const progressData = await getProgress(itemId)
      const entry = progressData[itemId]

      if (entry && downloadState.value[key]?.status === "downloading") {
        const rawProgress = typeof entry.progress === "number" ? entry.progress : 0
        const percentage = rawProgress <= 1 ? Math.round(rawProgress * 100) : Math.round(rawProgress)
        const normalizedProgress = Math.max(0, Math.min(100, percentage))
        const rawStatus = typeof entry.status === "string" ? entry.status : "downloading"

        const detail =
          rawStatus === "finalizing"
            ? "finalizing tags"
            : normalizedProgress > 0
              ? `${normalizedProgress}% transferred`
              : "connecting"

        downloadState.value = {
          ...downloadState.value,
          [key]: {
            ...downloadState.value[key],
            status: "downloading",
            progress: normalizedProgress,
            detail,
          },
        }
      }
    } catch {
      // Keep the download request active even if the progress poll flakes.
    }

    if (!control.active) break
    await wait(1200)
  }
}

async function downloadSingle(track: TrackMetadata) {
  const key = buildTrackKey(track)
  const itemId = globalThis.crypto?.randomUUID?.() ?? buildTrackKey(track)
  focusPlayer(track)
  queuedTrackKeys.value = queuedTrackKeys.value.filter((entry) => entry !== key)

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

    setNotice(
      "neutral",
      useFallback
        ? `Starting smart routing for ${track.title}…`
        : `Starting ${service} capture for ${track.title}…`,
    )

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
      [key]: { status: "success", fileName, itemId, progress: 100, detail: "ready to save" },
    }
    syncPlayerSource(false)
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

function enqueueTrack(track: TrackMetadata) {
  const key = buildTrackKey(track)
  const state = stateFor(track)

  if (state.status === "downloading" || state.status === "success" || queuedTrackKeys.value.includes(key)) {
    return
  }

  queuedTrackKeys.value = [...queuedTrackKeys.value, key]
  focusPlayer(track)
  setNotice("neutral", `${track.title} added to the queued desk.`)

  if (!queuePaused.value) {
    void processQueuedDownloads()
  }
}

function removeQueuedTrack(key: string) {
  queuedTrackKeys.value = queuedTrackKeys.value.filter((entry) => entry !== key)
}

async function processQueuedDownloads() {
  if (batchRunning.value || queuePaused.value) return

  batchRunning.value = true

  try {
    while (queuedTrackKeys.value.length && !queuePaused.value) {
      const nextKey = queuedTrackKeys.value[0]
      const nextEntry = trackCards.value.find((entry) => entry.key === nextKey)

      if (!nextEntry) {
        queuedTrackKeys.value = queuedTrackKeys.value.filter((entry) => entry !== nextKey)
        continue
      }

      await downloadSingle(nextEntry.track)
      queuedTrackKeys.value = queuedTrackKeys.value.filter((entry) => entry !== nextKey)
    }
  } finally {
    batchRunning.value = false
  }
}

function toggleQueuePaused() {
  queuePaused.value = !queuePaused.value
  if (!queuePaused.value) {
    void processQueuedDownloads()
  }
}

async function downloadAll() {
  if (!tracks.value.length) return

  for (const track of tracks.value) {
    const state = stateFor(track)
    if (state.status === "idle" || state.status === "error") {
      enqueueTrack(track)
    }
  }

  setNotice(
    "neutral",
    `Queued ${queuedTrackKeys.value.length} track${queuedTrackKeys.value.length > 1 ? "s" : ""} for sequential download.`,
  )
}

watch(
  tracks,
  (nextTracks) => {
    if (!nextTracks.length) {
      playerTrackKey.value = null
      stopPlayback()
      return
    }

    if (!playerTrackKey.value || !nextTracks.some((track) => buildTrackKey(track) === playerTrackKey.value)) {
      playerTrackKey.value = buildTrackKey(nextTracks[0])
    }
  },
  { immediate: true },
)

watch(playerSource, () => {
  syncPlayerSource(false)
})

onBeforeUnmount(() => {
  if (!playerAudio) return
  playerAudio.pause()
  playerAudio.src = ""
  playerAudio.load()
  playerAudio = null
})
</script>

<template>
  <main class="page page--index">
    <section class="signal-hero">
      <div class="signal-hero__copy brutal-panel brutal-panel--paper">
        <p class="section-kicker">editorial signal lab</p>
        <h1 class="signal-title">
          PURE
          <br />
          SIGNAL
          <br />
          DOWNLOADS.
        </h1>
        <p class="signal-lede">
          Brutalist link capture for people who already know what they want: paste, resolve, queue, download.
          No fake streaming chrome, no soft dashboard gloss.
        </p>

        <div class="signal-actions">
          <button type="button" class="action-button" :disabled="loading" @click="onSubmit">
            {{ loading ? "Reading signal…" : queryLooksLikeUrl ? "Resolve link" : "Search catalog" }}
          </button>
          <NuxtLink to="/docs" class="ghost-button ghost-button--frame">Inspect API atlas</NuxtLink>
        </div>

        <ul class="signal-notes">
          <li v-for="item in fieldNotes" :key="item.label">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </li>
        </ul>
      </div>

      <aside class="signal-poster brutal-panel brutal-panel--poster">
        <div class="poster-block poster-block--yellow" />
        <div class="poster-art">
          <img v-if="collectionArtwork" :src="collectionArtwork" alt="" loading="lazy" />
          <div v-else class="poster-geometry">
            <span />
            <span />
            <span />
          </div>
        </div>
        <div class="poster-tag">
          {{ provider === "auto" ? "SMART ROUTING" : provider.toUpperCase() }}
        </div>
      </aside>
    </section>

    <section class="workspace-grid">
      <div class="desk-stack">
        <section class="brutal-panel brutal-panel--paper composer-panel">
          <div class="panel-heading">
            <div>
              <p class="section-kicker">capture desk</p>
              <h2>Start from a link or a title.</h2>
            </div>

            <button
              v-if="trimmedQuery || metadata || searchResults.length"
              type="button"
              class="ghost-button ghost-button--frame"
              @click="clearWorkspace"
            >
              Reset
            </button>
          </div>

          <form class="composer-panel__form" @submit.prevent="onSubmit">
            <label class="field">
              <span class="field__label">Spotify / Deezer URL or search query</span>
              <textarea
                v-model="query"
                rows="4"
                class="field__input field__input--area"
                placeholder="Paste a track, album, playlist link, or type artist + song name"
              />
            </label>

            <div class="choice-grid">
              <div class="choice-group">
                <span class="field__label">Provider routing</span>
                <div class="choice-row">
                  <button
                    v-for="option in providerOptions"
                    :key="option.value"
                    type="button"
                    class="choice-pill"
                    :data-active="provider === option.value"
                    @click="provider = option.value"
                  >
                    <strong>{{ option.label }}</strong>
                    <small>{{ option.caption }}</small>
                  </button>
                </div>
              </div>

              <div class="choice-group">
                <span class="field__label">Search source</span>
                <div class="choice-row">
                  <button
                    v-for="option in sourceOptions"
                    :key="option.value"
                    type="button"
                    class="choice-pill"
                    :data-active="searchSource === option.value"
                    @click="searchSource = option.value"
                  >
                    <strong>{{ option.label }}</strong>
                    <small>{{ option.caption }}</small>
                  </button>
                </div>
              </div>
            </div>

            <div class="notice-band" :data-tone="notice.tone">
              <span>{{ notice.text }}</span>
            </div>
          </form>
        </section>

        <section v-if="searchResults.length" class="brutal-panel brutal-panel--paper result-stage">
          <div class="panel-heading">
            <div>
              <p class="section-kicker">search spread</p>
              <h2>Pick the closest hit.</h2>
            </div>
            <span class="metric-chip">{{ searchResults.length }} results</span>
          </div>

          <div class="result-layout">
            <button
              v-if="resultLead"
              type="button"
              class="result-lead"
              @click="selectTrack(resultLead)"
            >
              <div class="result-lead__body">
                <span class="metric-chip">{{ searchSource }}</span>
                <strong>{{ resultLead.title }}</strong>
                <p>{{ resultLead.artist }}</p>
                <small v-if="resultLead.album">{{ resultLead.album }}</small>
              </div>
              <span v-if="resultLead.duration_ms" class="result-lead__time">
                {{ formatDuration(resultLead.duration_ms) }}
              </span>
            </button>

            <ul class="result-grid">
              <li v-for="track in resultGrid" :key="buildTrackKey(track)">
                <button type="button" class="result-card" @click="selectTrack(track)">
                  <div>
                    <strong>{{ track.title }}</strong>
                    <p>{{ track.artist }}</p>
                  </div>
                  <small v-if="track.duration_ms">{{ formatDuration(track.duration_ms) }}</small>
                </button>
              </li>
            </ul>
          </div>
        </section>

        <section v-if="tracks.length" class="brutal-panel brutal-panel--paper collection-stage">
          <div class="collection-stage__head">
            <div class="collection-stage__copy">
              <p class="section-kicker">{{ collectionLabel }}</p>
              <h2>{{ collectionTitle }}</h2>
              <p>{{ collectionSubtitle }}</p>
            </div>

            <div class="collection-stage__meta">
              <span class="metric-chip">{{ tracks.length }} {{ tracks.length > 1 ? "tracks" : "track" }}</span>
              <button
                v-if="tracks.length > 1"
                type="button"
                class="action-button action-button--secondary"
                :disabled="batchRunning && !queuePaused"
                @click="downloadAll"
              >
                {{ batchRunning && !queuePaused ? "Queue running…" : `Queue all ${tracks.length}` }}
              </button>
            </div>
          </div>

          <ul class="track-grid">
            <li
              v-for="{ track, state, key } in trackCards"
              :key="key"
              class="track-card"
              :data-status="state.status"
            >
              <div class="track-card__slot">
                <span>{{ state.status === "success" ? "OK" : String(trackCards.findIndex((entry) => entry.key === key) + 1).padStart(2, "0") }}</span>
              </div>

              <div class="track-card__body">
                <strong>{{ track.title }}</strong>
                <p>{{ track.artist }}</p>
                <div class="track-card__meta">
                  <span v-if="track.album">{{ track.album }}</span>
                  <span v-if="track.duration_ms">{{ formatDuration(track.duration_ms) }}</span>
                  <span v-if="track.isrc">{{ track.isrc }}</span>
                </div>
                <p v-if="state.status === 'error'" class="track-card__error">
                  {{ state.error || "Download failed." }}
                </p>
              </div>

              <div class="track-card__aside">
                <a
                  v-if="state.status === 'success' && state.fileName"
                  :href="fileUrlFor(track)"
                  class="action-button action-button--secondary"
                >
                  Save file
                </a>
                <button
                  v-else
                  type="button"
                  class="action-button action-button--secondary"
                  :disabled="state.status === 'downloading' || batchRunning"
                  @click="downloadSingle(track)"
                >
                  {{ stateLabel(track) }}
                </button>
                <button
                  v-if="state.status !== 'success' && state.status !== 'downloading'"
                  type="button"
                  class="ghost-button ghost-button--frame track-card__queue"
                  :disabled="isQueued(track)"
                  @click="enqueueTrack(track)"
                >
                  {{ isQueued(track) ? "Queued" : "Add to queue" }}
                </button>
                <span class="status-pill" :data-status="state.status">{{ statusLabel(track) }}</span>
              </div>
            </li>
          </ul>
        </section>
      </div>

      <aside class="queue-stack">
        <section class="brutal-panel brutal-panel--paper player-stage">
          <div class="panel-heading">
            <div>
              <p class="section-kicker">listening desk</p>
              <h2>Player.</h2>
            </div>
            <span class="metric-chip">{{ playerTrack ? "live" : "idle" }}</span>
          </div>

          <div v-if="playerTrack" class="player-display">
            <div class="player-display__art">
              <img v-if="playerTrack.cover_url" :src="playerTrack.cover_url" alt="" loading="lazy" />
              <div v-else class="player-display__fallback">
                <span>FLAC</span>
              </div>
            </div>

            <div class="player-display__copy">
              <strong>{{ playerTrack.title }}</strong>
              <p>{{ playerTrack.artist }}</p>
              <small>{{ playerTrack.album || "Single file" }}</small>
              <span class="player-display__source">{{ playerSourceLabel }}</span>
            </div>
          </div>

          <div v-if="playerTrack" class="player-controls">
            <button
              type="button"
              class="action-button action-button--secondary"
              :disabled="!playerSource"
              @click="togglePlayback"
            >
              {{ playerPlaying ? "Pause" : "Play" }}
            </button>
            <span class="metric-chip">
              {{ formatDuration(playerCurrentTime * 1000) }} / {{ formatDuration((playerDuration || 0) * 1000) }}
            </span>
          </div>

          <input
            v-if="playerTrack"
            class="player-slider"
            type="range"
            min="0"
            :max="Math.max(playerDuration, 0)"
            :step="1"
            :value="playerCurrentTime"
            :disabled="!playerSource"
            @input="seekPlayback"
          />

          <p v-if="!playerTrack" class="queue-empty">
            Select a track from the desk to prime the player.
          </p>
        </section>

        <section class="brutal-panel brutal-panel--paper queue-stage">
          <div class="panel-heading">
            <div>
              <p class="section-kicker">download queue</p>
              <h2>Active transfers.</h2>
            </div>
            <span class="metric-chip">{{ activeDownloadCards.length }}</span>
          </div>

          <div v-if="activeDownloadCards.length" class="queue-list">
            <article
              v-for="{ track, state, key } in activeDownloadCards"
              :key="key"
              class="queue-card"
            >
              <div class="queue-card__top">
                <div>
                  <strong>{{ track.title }}</strong>
                  <p>{{ track.artist }}</p>
                </div>
                <span class="queue-card__percent">{{ state.progress || 0 }}%</span>
              </div>

              <div class="queue-card__chips">
                <span class="mini-chip mini-chip--dark">FLAC</span>
                <span class="mini-chip mini-chip--signal">{{ provider === "auto" ? "SMART" : provider.toUpperCase() }}</span>
              </div>

              <div class="progress-rail">
                <span class="progress-rail__fill" :style="{ width: `${state.progress || 0}%` }" />
                <span class="progress-rail__stripes" />
              </div>

              <div class="queue-card__foot">
                <span>{{ state.detail || "connecting" }}</span>
                <span>{{ track.duration_ms ? formatDuration(track.duration_ms) : "duration pending" }}</span>
              </div>
            </article>
          </div>
          <p v-else class="queue-empty">
            No active downloads yet. Start one track or launch a full collection queue.
          </p>
        </section>

        <section class="brutal-panel brutal-panel--paper queue-stage">
          <div class="panel-heading">
            <div>
              <p class="section-kicker">queued desk</p>
              <h2>Next in line.</h2>
            </div>
            <div class="queue-stage__controls">
              <span class="metric-chip">{{ queuedTrackCards.length }}</span>
              <button
                v-if="queuedTrackCards.length"
                type="button"
                class="ghost-button ghost-button--frame queue-stage__toggle"
                @click="toggleQueuePaused"
              >
                {{ queuePaused ? "Resume queued" : "Pause queued" }}
              </button>
            </div>
          </div>

          <ul v-if="queuedTrackCards.length" class="queued-list">
            <li v-for="{ track, key } in queuedTrackCards" :key="key" class="queued-item">
              <div class="queued-item__mark">Q</div>
              <div class="queued-item__copy">
                <strong>{{ track.title }}</strong>
                <p>{{ track.artist }}</p>
              </div>
              <div class="queued-item__actions">
                <small>{{ track.duration_ms ? formatDuration(track.duration_ms) : "—" }}</small>
                <button type="button" class="queued-item__remove" @click="removeQueuedTrack(key)">Remove</button>
              </div>
            </li>
          </ul>
          <p v-else class="queue-empty">
            Queue items appear here once a track sheet is resolved.
          </p>
        </section>

        <section v-if="readyDownloadCards.length" class="brutal-panel brutal-panel--paper queue-stage">
          <div class="panel-heading">
            <div>
              <p class="section-kicker">ready files</p>
              <h2>Captured.</h2>
            </div>
            <span class="metric-chip">{{ readyDownloadCards.length }}</span>
          </div>

          <ul class="queued-list">
            <li v-for="{ track, state, key } in readyDownloadCards" :key="key" class="queued-item queued-item--ready">
              <div class="queued-item__mark">OK</div>
              <div class="queued-item__copy">
                <strong>{{ track.title }}</strong>
                <p>{{ state.fileName }}</p>
              </div>
              <div class="queued-item__actions">
                <button type="button" class="queued-item__remove" @click="focusPlayer(track)">Listen</button>
                <a :href="fileUrlFor(track)" class="ghost-button ghost-button--frame">Save</a>
              </div>
            </li>
          </ul>
        </section>
      </aside>
    </section>
  </main>
</template>
