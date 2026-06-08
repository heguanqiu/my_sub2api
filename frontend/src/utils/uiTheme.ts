export type UiTheme = 'classic' | 'mecha'

const UI_THEME_STORAGE_KEY = 'uiTheme'

function normalizeUiTheme(value: string | null): UiTheme {
  return value === 'mecha' ? 'mecha' : 'classic'
}

export function getStoredUiTheme(): UiTheme {
  if (typeof window === 'undefined') return 'classic'
  return normalizeUiTheme(window.localStorage.getItem(UI_THEME_STORAGE_KEY))
}

export function applyUiTheme(theme: UiTheme) {
  if (typeof document === 'undefined') return

  if (theme === 'mecha') {
    document.documentElement.dataset.uiTheme = 'mecha'
  } else {
    delete document.documentElement.dataset.uiTheme
  }
}

export function setUiTheme(theme: UiTheme) {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(UI_THEME_STORAGE_KEY, theme)
  }
  applyUiTheme(theme)
}

export function initUiTheme(): UiTheme {
  const theme = getStoredUiTheme()
  applyUiTheme(theme)
  return theme
}
