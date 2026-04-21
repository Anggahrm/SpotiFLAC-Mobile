<script setup lang="ts">
const { theme, toggleTheme } = useTheme()
const nextLabel = computed(() => (theme.value === "dark" ? "Switch to light mode" : "Switch to dark mode"))
</script>

<template>
  <button
    type="button"
    class="theme-toggle"
    :aria-label="nextLabel"
    :title="nextLabel"
    @click="toggleTheme"
  >
    <span class="theme-toggle__label">{{ theme === "dark" ? "Night shift" : "Day shift" }}</span>
    <span class="theme-toggle__track">
      <span class="theme-toggle__thumb" :data-theme="theme" />
    </span>
  </button>
</template>
