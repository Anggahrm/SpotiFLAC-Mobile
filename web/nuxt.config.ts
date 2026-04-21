export default defineNuxtConfig({
  compatibilityDate: "2025-07-15",
  srcDir: ".",
  css: ["~/assets/css/main.css"],
  nitro: {
    prerender: {
      routes: ["/", "/docs"],
    },
  },
  app: {
    head: {
      title: "zFlac Atelier",
      meta: [
        {
          name: "description",
          content: "Fresh Nuxt interface for high-quality music downloads from Spotify and Deezer links.",
        },
        {
          name: "theme-color",
          content: "#171326",
        },
      ],
    },
  },
  devtools: { enabled: false },
})
