<template>
  <div class="shell">
    <header class="topbar">
      <NuxtLink to="/" class="brand">
        <span class="brand__mark">FLAC</span>
        <span class="brand__text">
          <strong>zFlac Signal Desk</strong>
          <small>brutalist capture field</small>
        </span>
      </NuxtLink>

      <nav class="topbar__nav">
        <NuxtLink to="/" class="topbar__link">Workspace</NuxtLink>
        <NuxtLink to="/docs" class="topbar__link">API atlas</NuxtLink>
        <ThemeToggle />
      </nav>
    </header>

    <NuxtRouteAnnouncer />
    <NuxtPage />
  </div>
</template>

<script setup lang="ts">
useHead({
  titleTemplate: (titleChunk) => (titleChunk ? `${titleChunk} · zFlac Signal Desk` : "zFlac Signal Desk"),
})
</script>
