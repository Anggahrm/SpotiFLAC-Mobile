export type ThemeMode = "light" | "dark"

const storageKey = "zflac-theme"

export function useTheme() {
  const theme = useState<ThemeMode>("zflac-theme", () => "dark")
  const initialized = useState<boolean>("zflac-theme-init", () => false)

  const applyTheme = (nextTheme: ThemeMode) => {
    theme.value = nextTheme

    if (import.meta.client) {
      document.documentElement.dataset.theme = nextTheme
      window.localStorage.setItem(storageKey, nextTheme)
    }
  }

  if (import.meta.client && !initialized.value) {
    const savedTheme = window.localStorage.getItem(storageKey)
    const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
    applyTheme(savedTheme === "light" || savedTheme === "dark" ? savedTheme : systemTheme)
    initialized.value = true
  }

  const toggleTheme = () => {
    applyTheme(theme.value === "dark" ? "light" : "dark")
  }

  return { theme, applyTheme, toggleTheme }
}
