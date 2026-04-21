<script setup lang="ts">
import { apiEndpoints } from "~/lib/api-docs"

useSeoMeta({
  title: "API atlas",
  description: "Reference for the Go backend routes that power the brutalist download workspace.",
})
</script>

<template>
  <main class="page page--docs">
    <section class="docs-mast">
      <div class="brutal-panel brutal-panel--paper docs-mast__copy">
        <p class="section-kicker">backend contract</p>
        <h1>API ATLAS.</h1>
        <p>
          The web surface has been reworked, but the Go backend remains the engine. This API atlas keeps every route,
          parameter, and payload visible in a print-heavy signal sheet.
        </p>
      </div>

      <NuxtLink to="/" class="ghost-button ghost-button--frame docs-mast__back">
        Back to workspace
      </NuxtLink>
    </section>

    <section class="docs-wall">
      <article v-for="endpoint in apiEndpoints" :key="endpoint.id" class="brutal-panel brutal-panel--paper doc-card">
        <div class="doc-card__top">
          <span class="method-badge" :data-method="endpoint.method">{{ endpoint.method }}</span>
          <code>{{ endpoint.path }}</code>
        </div>

        <h2>{{ endpoint.title }}</h2>
        <p class="doc-card__summary">{{ endpoint.description }}</p>

        <div v-if="endpoint.parameters?.length" class="doc-card__block">
          <h3>Parameters</h3>
          <ul class="doc-list">
            <li v-for="parameter in endpoint.parameters" :key="parameter.name">
              <div class="doc-list__head">
                <strong>{{ parameter.name }}</strong>
                <span>{{ parameter.type }}</span>
                <em>{{ parameter.required ? "required" : "optional" }}</em>
              </div>
              <p>{{ parameter.description }}</p>
            </li>
          </ul>
        </div>

        <div v-if="endpoint.requestBody" class="doc-card__block">
          <h3>Request body</h3>
          <p>{{ endpoint.requestBody.description }}</p>
          <pre>{{ JSON.stringify(endpoint.requestBody.example, null, 2) }}</pre>
        </div>
      </article>
    </section>
  </main>
</template>
