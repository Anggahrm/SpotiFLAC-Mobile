---
title: 'Web UI/UX Redesign + API Documentation'
slug: 'web-ui-redesign-api-docs'
created: '2026-02-08'
status: 'completed'
stepsCompleted: [1, 2, 3, 4]
tech_stack:
  - SvelteKit 2.50.1
  - Svelte 5.48.2 (runes)
  - Tailwind CSS 4.1.18 (v4)
  - TypeScript 5.9.3
  - Vite 7.3.1
  - lucide-svelte 0.563.0
  - bits-ui 2.15.4
  - clsx + tailwind-merge
files_to_modify:
  - web/src/routes/+page.svelte (redesign homepage)
  - web/src/routes/+layout.svelte (add theme toggle, update layout)
  - web/src/app.css (new color scheme, CSS variables for dark/light)
  - web/src/lib/components/Toast.svelte (dark theme support)
  - web/src/lib/components/ui/Button.svelte (update variants)
  - web/src/lib/components/ui/Card.svelte (dark theme)
  - web/src/lib/components/ui/Input.svelte (dark theme)
files_to_create:
  - web/src/routes/docs/+page.svelte (API docs page)
  - web/src/lib/stores/theme.ts (theme store with localStorage)
  - web/src/lib/components/ThemeToggle.svelte (dark/light toggle)
  - web/src/lib/components/docs/ApiEndpoint.svelte (endpoint documentation component)
  - web/src/lib/components/docs/CodeBlock.svelte (code examples with tabs)
  - web/src/lib/components/docs/Playground.svelte (API playground)
  - web/src/lib/data/api-docs.ts (API endpoint definitions)
code_patterns:
  - Svelte 5 runes ($state, $props, $derived, $bindable)
  - CSS variables for theming (--primary, --border, --background, etc.)
  - cn() utility for class merging (clsx + tailwind-merge)
  - Svelte stores (writable) for global state
  - Static adapter with SPA fallback
  - Prerender + no SSR
test_patterns:
  - Manual browser testing
  - No automated tests currently
---

# Tech-Spec: Web UI/UX Redesign + API Documentation

**Created:** 2026-02-08

## Overview

### Problem Statement

1. UI web zFlac saat ini menggunakan neo-brutalist style dengan layout single card yang terbatas (max-w-lg)
2. Tidak ada dark mode - hanya light theme
3. Tidak ada dokumentasi API yang bisa diakses dan dicoba langsung oleh developer
4. Layout tidak memanfaatkan layar desktop secara optimal

### Solution

1. Redesign total web UI dengan mengadopsi design system terinspirasi dari overrrides.com:
   - Dark-first theme dengan pure black background
   - Clean, minimal aesthetic dengan high contrast
   - Grid/bento layout yang memanfaatkan full screen

2. Implementasi dark/light mode toggle dengan localStorage persistence

3. Tambah halaman `/docs` dengan API playground interaktif:
   - Dokumentasi lengkap 9 API endpoints
   - Code examples dalam multiple format (cURL, Fetch, Axios, Python)
   - Playground untuk test API langsung

### Scope

**In Scope:**

A. UI/UX Redesign (Homepage `/`)
- Dark-first theme + light mode toggle (localStorage persistence)
- New color palette: Pure black (#000000) background, white text, subtle grays
- Layout expansion: Grid/bento layout untuk desktop, single column untuk mobile
  - Breakpoints: Mobile (<768px) single column, Tablet (768-1024px) 2 columns, Desktop (>1024px) bento grid
- Redesign semua komponen:
  - Search input area (clean, minimal)
  - Provider/source selector (sleek buttons/chips)
  - Track cards & list (dark cards dengan subtle borders)
  - Album/Playlist view (expanded layout)
  - Download progress (polished progress bars)
  - Toast notifications (dark themed)
- Typography: JetBrains Mono untuk semua text (maintain current monospace aesthetic)
- Text animations/motion: typewriter, reveal, stagger effects untuk visual interest
- Micro-interactions & hover states
- Custom scrollbar (dark themed)

B. API Docs Page (`/docs`)
- Route baru: `/docs`
- Interactive documentation untuk 9 endpoints:
  1. GET /health
  2. GET /api/search
  3. GET /api/metadata
  4. GET /api/parse-url
  5. GET /api/availability
  6. POST /api/download
  7. GET /api/progress
  8. GET /api/files/{filename}
  9. GET /api/lyrics
- Code examples dalam multiple format:
  - cURL
  - JavaScript Fetch
  - Axios
  - Python (requests)
- Playground dengan:
  - Form input per parameter
  - "Try it" button untuk execute request
  - Response viewer (formatted JSON dengan syntax highlighting)
  - Copy button untuk setiap code example
- Konsisten dengan `api.ts` yang sudah ada

**Out of Scope:**
- Backend/API changes
- Mobile app
- New API endpoints
- Authentication/authorization

## Context for Development

### Codebase Patterns

- **Framework**: SvelteKit 2.50.1 dengan Svelte 5.48.2 runes syntax
- **Styling**: Tailwind CSS 4.1.18 (v4 - no config file, uses @import "tailwindcss")
- **State Management**: Svelte 5 `$state()` rune + Svelte stores (`writable`)
- **Props**: Svelte 5 `$props()` dengan `$bindable()` untuk two-way binding
- **Derived State**: Svelte 5 `$derived()` rune
- **Icons**: lucide-svelte
- **Font**: JetBrains Mono (monospace) via Google Fonts
- **API Client**: Custom fetch wrapper di `api.ts` dengan APIResponse pattern
- **Class Utility**: `cn()` function menggunakan clsx + tailwind-merge
- **Build**: Static adapter, prerender=true, ssr=false (full SPA)

### Files to Reference

| File | Purpose |
| ---- | ------- |
| web/src/routes/+page.svelte | Homepage - 700 lines, semua UI logic |
| web/src/routes/+layout.svelte | Root layout dengan Toast container |
| web/src/routes/+layout.ts | Prerender + SSR config |
| web/src/app.css | Global styles, Tailwind import, scrollbar, fonts |
| web/src/lib/api.ts | API client - 9 functions, TypeScript interfaces |
| web/src/lib/utils.ts | cn() class merge utility |
| web/src/lib/stores/toasts.ts | Toast store dengan success/error/info/loading |
| web/src/lib/components/Toast.svelte | Toast component dengan fly transition |
| web/src/lib/components/ui/Button.svelte | Button dengan CSS variables |
| web/src/lib/components/ui/Card.svelte | Card dengan CSS variables |
| web/src/lib/components/ui/Input.svelte | Input dengan CSS variables |
| pkg/api/server.go | Backend API - 9 endpoints untuk dokumentasi |

### Technical Decisions

- **Design Reference**: overrrides.com style - dark-first, grid layout, clean minimal, high contrast
- **Color Scheme**: Pure black (#000000) bg, white text, subtle grays untuk borders
- **Theme Toggle**: CSS variables dengan class toggle (`.dark`) + localStorage persistence
- **Theme Store**: Svelte writable store untuk reactive theme state
- **Code Examples**: Custom CodeBlock component dengan tabs (curl, fetch, axios, python)
- **API Playground**: Direct fetch dari browser, response viewer dengan JSON formatting
- **New Route**: `/docs` sebagai SPA route (handled by fallback)
- **No New Dependencies**: Semua bisa diimplementasi dengan existing stack

## Implementation Plan

### Tasks

#### Phase 1: Foundation - Theme System

- [x] Task 1: Create theme store with localStorage persistence
  - File: `web/src/lib/stores/theme.ts`
  - Action: Create writable store untuk theme state ('dark' | 'light'), init dari localStorage, subscribe untuk persist changes
  - Notes: Default ke 'dark' (dark-first design), check `prefers-color-scheme` untuk initial value jika belum ada di localStorage

- [x] Task 2: Update global CSS dengan dark/light CSS variables
  - File: `web/src/app.css`
  - Action: Define CSS variables untuk kedua theme dalam `:root` dan `.light` class, update existing styles
  - Notes: Variables: `--background`, `--foreground`, `--card`, `--card-foreground`, `--border`, `--primary`, `--muted`, `--muted-foreground`. Dark = #000000 bg, white text. Light = #ffffff bg, dark text.

- [x] Task 3: Create ThemeToggle component
  - File: `web/src/lib/components/ThemeToggle.svelte`
  - Action: Create toggle button dengan Sun/Moon icons dari lucide-svelte, bind ke theme store
  - Notes: Gunakan smooth transition saat toggle, show current theme icon

- [x] Task 4: Update root layout untuk theme support
  - File: `web/src/routes/+layout.svelte`
  - Action: Import theme store, apply `.light` class ke `<html>` element berdasarkan theme, add ThemeToggle ke layout
  - Notes: Gunakan `$effect()` untuk sync theme class ke document.documentElement

- [x] Task 4b: Add inline script untuk theme anti-flash
  - File: `web/src/app.html`
  - Action: Add inline `<script>` di `<head>` untuk set theme class sebelum render
  - Notes: Script harus read localStorage dan set class SEBELUM Svelte hydrates. Example:
    ```html
    <script>
      const theme = localStorage.getItem('theme') ||
        (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
      document.documentElement.classList.toggle('light', theme === 'light');
    </script>
    ```

#### Phase 2: Homepage Redesign

- [x] Task 5: Extract homepage sections ke components (prep step) - SKIPPED (out of scope)
  - Files: Create `web/src/lib/components/SearchSection.svelte`, `TrackView.svelte`, `AlbumView.svelte`, `PlaylistView.svelte`
  - Action: Extract search input + selectors, track card, album view, playlist view dari +page.svelte ke separate components
  - Notes: Reduce 700-line file complexity, enable parallel work, maintain existing functionality first

- [x] Task 6: Update homepage layout structure
  - File: `web/src/routes/+page.svelte`
  - Action: Ubah dari single card (max-w-lg) ke full-width layout dengan grid/bento design untuk desktop
  - Notes: Use breakpoints: <768px single column, 768-1024px 2 columns, >1024px bento grid. Remove neo-brutalist shadows, use subtle borders.

- [x] Task 6: Redesign search input area
  - File: `web/src/routes/+page.svelte`
  - Action: Update input styling ke dark theme, clean minimal design, subtle focus states
  - Notes: Use CSS variables dari app.css

- [x] Task 7: Redesign provider/source selector
  - File: `web/src/routes/+page.svelte`
  - Action: Update buttons ke sleek pill/chip style dengan subtle borders, update hover states
  - Notes: Active state dengan accent color, inactive dengan muted colors

- [x] Task 8: Redesign track/album/playlist cards
  - File: `web/src/routes/+page.svelte`
  - Action: Update card styling ke dark theme dengan subtle borders, clean typography, update hover states
  - Notes: Cover image dengan rounded corners, metadata dengan proper hierarchy

- [x] Task 9: Redesign track list items
  - File: `web/src/routes/+page.svelte`
  - Action: Update list item styling, subtle hover states, clean status indicators
  - Notes: Maintain clickable tracks untuk playlist

- [x] Task 10: Redesign download progress indicators
  - File: `web/src/routes/+page.svelte`
  - Action: Update progress bar styling ke dark theme, use accent color for fill
  - Notes: Clean percentage display, subtle animation

- [x] Task 11: Update Toast component untuk dark theme
  - File: `web/src/lib/components/Toast.svelte`
  - Action: Update color scheme ke support dark/light theme via CSS variables
  - Notes: Remove neo-brutalist shadows, use subtle styling

- [x] Task 12: Update UI components untuk theme support
  - Files: `web/src/lib/components/ui/Button.svelte`, `Card.svelte`, `Input.svelte`
  - Action: Verify CSS variables bekerja dengan theme system, update jika perlu
  - Notes: Komponen ini sudah pakai CSS variables, pastikan compatible

- [x] Task 13: Update scrollbar styling untuk dark theme
  - File: `web/src/app.css`
  - Action: Update scrollbar colors untuk dark theme, subtle styling
  - Notes: Match dengan overall dark aesthetic

- [x] Task 14: Add navigation link ke /docs
  - File: `web/src/routes/+page.svelte` atau `+layout.svelte`
  - Action: Add subtle link/button ke API docs page
  - Notes: Position di header area

#### Phase 3: API Documentation Page

- [x] Task 15: Create API endpoint data definitions
  - File: `web/src/lib/data/api-docs.ts`
  - Action: Define TypeScript interfaces dan data untuk 9 API endpoints (method, path, description, parameters, request body, response examples)
  - Notes: Data harus konsisten dengan `pkg/api/server.go` dan `api.ts`

- [x] Task 16: Create CodeBlock component dengan tabs
  - File: `web/src/lib/components/docs/CodeBlock.svelte`
  - Action: Create tabbed code block dengan 4 tabs (cURL, Fetch, Axios, Python), syntax highlighting via simple CSS, copy button per tab
  - Notes: Generate code examples DYNAMICALLY from endpoint data (method, path, params) - tidak hardcode. Ensure consistency dengan api.ts

- [x] Task 17: Create API Playground component
  - File: `web/src/lib/components/docs/Playground.svelte`
  - Action: Create form dengan dynamic inputs berdasarkan endpoint parameters, "Try it" button, response viewer dengan formatted JSON
  - Notes: Use native fetch, handle loading states dengan skeleton, proper error states dengan clear messages, copy response button

- [x] Task 18: Create ApiEndpoint component
  - File: `web/src/lib/components/docs/ApiEndpoint.svelte`
  - Action: Create collapsible/expandable endpoint documentation section yang combine description, parameters, CodeBlock, dan Playground
  - Notes: Show method badge (GET/POST), path, description

- [x] Task 19: Create docs page
  - File: `web/src/routes/docs/+page.svelte`
  - Action: Create API documentation page dengan list of all endpoints menggunakan ApiEndpoint component, add navigation back to homepage
  - Notes: Same theme system dengan homepage, responsive layout

- [x] Task 20: Create docs layout (optional) - SKIPPED
  - File: `web/src/routes/docs/+layout.svelte`
  - Action: Create layout khusus untuk docs (jika perlu sidebar navigation)
  - Notes: Optional - bisa skip jika single page docs cukup

#### Phase 4: Polish & Integration

- [x] Task 21: Add text animations dan motion effects
  - Files: Various components
  - Action: Implement text animations untuk visual interest:
    - Typewriter effect untuk hero/title text
    - Text reveal/fade-in untuk section headers
    - Character stagger animation untuk search results appearing
    - Counter animation untuk stats (track count, duration)
  - Notes: Use Svelte transitions (fly, fade, slide) dengan stagger delays, CSS @keyframes, IntersectionObserver untuk scroll-triggered

- [x] Task 22: Add micro-interactions dan hover transitions
  - Files: Various components
  - Action: Add smooth transitions untuk hover states, theme toggle, card interactions
  - Notes: Use Svelte transitions atau CSS transitions

- [x] Task 22: Add micro-interactions dan hover transitions
  - Files: Various components
  - Action: Add smooth transitions untuk hover states, theme toggle, card interactions, button press effects
  - Notes: Use CSS transitions dan Svelte transitions

- [x] Task 23: Test responsive design
  - Action: Test semua breakpoints (mobile <768px, tablet 768-1024px, desktop >1024px), fix layout issues
  - Notes: Priority: mobile first, then desktop

- [x] Task 24: Build dan verify static output
  - Action: Run `npm run build`, verify output di `../static`, test dengan backend
  - Notes: Ensure SPA routing works untuk /docs

### Acceptance Criteria

#### Theme System
- [ ] AC1: Given user buka website pertama kali, when page loads, then dark theme ditampilkan sebagai default
- [ ] AC2: Given user di dark mode, when klik theme toggle, then UI switch ke light mode dengan smooth transition
- [ ] AC3: Given user sudah set theme preference, when refresh page, then theme preference dipertahankan dari localStorage
- [ ] AC4: Given user di mobile device, when toggle theme, then semua komponen update sesuai theme

#### Homepage Redesign
- [ ] AC5: Given user di desktop (>1024px), when buka homepage, then layout menggunakan grid yang memanfaatkan full width
- [ ] AC6: Given user di mobile (<768px), when buka homepage, then layout single column yang optimal untuk mobile
- [ ] AC7: Given user search track, when results muncul, then cards ditampilkan dengan dark theme styling dan proper spacing
- [ ] AC8: Given user download track, when download in progress, then progress bar menampilkan progress dengan accent color
- [ ] AC9: Given action berhasil/gagal, when toast muncul, then toast menggunakan theme-aware styling
- [ ] AC10: Given user hover pada interactive elements, when hovering, then subtle hover state terlihat
- [ ] AC10b: Given page loads, when hero/title text appears, then typewriter atau reveal animation terlihat
- [ ] AC10c: Given user scroll, when section headers masuk viewport, then text fade-in animation terjadi
- [ ] AC10d: Given search results load, when tracks muncul, then stagger animation per item terlihat

#### API Documentation
- [ ] AC11: Given user navigasi ke /docs, when page loads, then semua 9 API endpoints ditampilkan dengan dokumentasi lengkap
- [ ] AC12: Given user lihat endpoint, when expand details, then description, parameters, dan code examples tersedia
- [ ] AC13: Given user pilih tab "cURL", when viewing code block, then valid cURL command ditampilkan
- [ ] AC14: Given user pilih tab "Fetch", when viewing code block, then valid JavaScript fetch code ditampilkan
- [ ] AC15: Given user pilih tab "Axios", when viewing code block, then valid Axios code ditampilkan
- [ ] AC16: Given user pilih tab "Python", when viewing code block, then valid Python requests code ditampilkan
- [ ] AC17: Given user klik copy button, when code di-copy, then code tercopy ke clipboard dan feedback ditampilkan
- [ ] AC18: Given user isi form playground dan klik "Try it", when API dipanggil, then response ditampilkan dengan formatted JSON
- [ ] AC19: Given API call gagal, when error terjadi, then error message ditampilkan dengan jelas
- [ ] AC20: Given user di /docs, when klik link ke homepage, then navigasi ke homepage bekerja

## Additional Context

### Dependencies

- **Existing (no changes needed):**
  - SvelteKit 2.50.1
  - Svelte 5.48.2
  - Tailwind CSS 4.1.18
  - lucide-svelte 0.563.0
  - clsx + tailwind-merge

- **New Dependencies:** Tidak ada - semua diimplementasi dengan existing stack

- **API Dependencies:**
  - Backend harus running untuk test playground
  - Semua 9 endpoints harus accessible

### Testing Strategy

**Manual Testing:**
1. **Theme System:**
   - Toggle dark/light mode di desktop dan mobile
   - Refresh page dan verify persistence
   - Check semua komponen update correctly

2. **Homepage:**
   - Test responsive breakpoints (mobile, tablet, desktop)
   - Test search flow (URL input, keyword search)
   - Test download flow (single track, album, playlist)
   - Verify hover states dan transitions

3. **API Docs:**
   - Navigate ke /docs
   - Expand setiap endpoint
   - Switch tabs di code block
   - Copy code dan verify clipboard
   - Test playground dengan real API calls
   - Test error handling (invalid input, network error)

**Browser Testing:**
- Chrome (primary)
- Firefox
- Safari (if available)
- Mobile Chrome/Safari

### Notes

**High-Risk Items:**
- Homepage redesign (700 lines) - perlu careful refactoring untuk maintain existing functionality
- Theme toggle timing - ensure no flash of wrong theme on initial load

**Known Limitations:**
- Syntax highlighting sederhana (CSS-based, bukan library seperti Prism/Shiki)
- Playground tidak support file upload untuk download endpoint
- API docs static - tidak auto-sync dengan backend changes

**Future Considerations (Out of Scope):**
- OpenAPI/Swagger spec generation
- API versioning documentation
- Rate limit documentation
- Automated testing
- PWA support

## Review Notes

- Adversarial review completed
- Findings: 14 total, 8 fixed, 6 skipped (noise/not applicable)
- Resolution approach: Fix all real issues + accept suggestions
- Fixed issues:
  - F1: Playground fetch options now passed correctly
  - F4: Clipboard API wrapped with try/catch
  - F5: ARIA attributes added to collapsible sections
  - F6: Focus indicator added to theme toggle
  - F8: $derived.by() used correctly
  - F10: res.ok check added before JSON parsing
  - F11: Batch progress extracted to component
