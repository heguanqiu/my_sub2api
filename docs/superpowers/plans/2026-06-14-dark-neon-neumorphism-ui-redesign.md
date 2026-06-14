# Dark Neon Neumorphism UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign every frontend page into a permanent dark neon neumorphism interface while preserving all functions listed in `docs/frontend-ui-function-inventory.md`.

**Architecture:** Build a shared dark-neon design system first, then migrate layout chrome, shared controls, and page groups. Keep existing Vue routes, Pinia stores, API calls, form handlers, modals, table interactions, payment flows, and auth callbacks intact; only change presentation and small theme helpers.

**Tech Stack:** Vue 3, Vite, TypeScript, TailwindCSS, Pinia, Vue Router, Vitest, Chart.js, GSAP already present in the frontend.

---

## Scope Check

This is one frontend subsystem, not separate backend/product subsystems. The work is large, so implementation is split by independently verifiable page groups: foundation, chrome, public/auth, user, payment, sales, admin, ops/charts, and final audit.

## File Structure

Create:

- `frontend/src/utils/theme.ts`: permanent dark-theme utility used by app bootstrap, embedded pages, and tests.
- `frontend/src/utils/__tests__/theme.spec.ts`: verifies the app always applies and reports dark theme.
- `frontend/src/components/common/StatusRing.vue`: reusable neon gradient status ring for quota, usage, health, risk, and subscription summaries.
- `frontend/src/components/common/__tests__/StatusRing.spec.ts`: verifies status ring labels, CSS variables, and accessibility text.
- `frontend/scripts/audit-dark-neon.mjs`: source audit that fails when a migrated group still contains hard light-theme surface classes.

Modify:

- `frontend/package.json`: add `audit:neon`.
- `frontend/tailwind.config.js`: add charcoal surfaces, neon accent colors, glow shadows, inset shadows, and gradient utilities.
- `frontend/src/style.css`: rewrite global base, buttons, inputs, cards, tables, badges, dropdowns, dialogs, tabs, progress, switch, skeleton, page, and sidebar classes.
- `frontend/src/main.ts`: use `applyPermanentDarkTheme()` before mount.
- `frontend/src/utils/embedded-url.ts`: make `detectTheme()` and default embedded theme always return dark.
- `frontend/src/utils/__tests__/embedded-url.spec.ts`: update light-theme expectations to dark-only behavior.
- `frontend/index.html`: set dark browser theme color.
- `frontend/src/components/layout/AppLayout.vue`, `AppHeader.vue`, `AppSidebar.vue`, `AuthLayout.vue`, `TablePageLayout.vue`: redesign chrome and remove theme toggle.
- `frontend/src/styles/onboarding.css`: dark-neon onboarding popover.
- `frontend/src/components/common/*.vue`: shared control and feedback components.
- `frontend/src/components/charts/*.vue` and ops chart components: dark-neon chart palettes.
- All routed view files under `frontend/src/views/**/*.vue`: page-level migration by group.
- Domain components under `frontend/src/components/auth`, `user`, `payment`, `admin`, `account`, `channels`, `keys`, and `layout`: support migrated views.

Do not modify backend files for this UI work.

## Design System Class Targets

Implement these global classes in `frontend/src/style.css` and use them throughout page migrations:

```css
.neon-page { @apply relative z-10 space-y-6; }
.neon-grid { @apply grid gap-4 md:gap-5 xl:gap-6; }
.neo-panel { @apply rounded-[1.35rem] border border-neon-cyan/15 bg-surface-900/80 shadow-neo backdrop-blur-xl; }
.neo-panel-hover { @apply transition-all duration-200 hover:-translate-y-0.5 hover:border-neon-cyan/35 hover:shadow-neon-cyan; }
.neo-control { @apply rounded-2xl border border-white/10 bg-surface-950/70 shadow-neo-inset; }
.neo-toolbar { @apply flex flex-wrap items-center gap-3 rounded-[1.35rem] border border-neon-cyan/15 bg-surface-900/80 p-4 shadow-neo backdrop-blur-xl; }
.neo-section-title { @apply text-sm font-semibold text-slate-100; }
.neo-muted { @apply text-sm text-slate-400; }
.neo-danger-zone { @apply rounded-2xl border border-neon-pink/25 bg-neon-pink/10 shadow-neon-pink; }
```

Use 8px to 22px radii by context: dense tables and controls stay tighter, dashboards and hero cards can use larger soft radii.

## Task 1: Permanent Dark Theme Utility

**Files:**
- Create: `frontend/src/utils/theme.ts`
- Create: `frontend/src/utils/__tests__/theme.spec.ts`
- Modify: `frontend/src/main.ts`
- Modify: `frontend/src/utils/embedded-url.ts`
- Modify: `frontend/src/utils/__tests__/embedded-url.spec.ts`

- [ ] **Step 1: Write the failing theme tests**

Create `frontend/src/utils/__tests__/theme.spec.ts`:

```ts
import { afterEach, describe, expect, it } from 'vitest'
import { applyPermanentDarkTheme, detectPermanentTheme } from '../theme'

describe('permanent dark theme', () => {
  afterEach(() => {
    document.documentElement.className = ''
    localStorage.clear()
  })

  it('always applies the dark class and stores dark preference', () => {
    localStorage.setItem('theme', 'light')

    applyPermanentDarkTheme()

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(document.documentElement.style.colorScheme).toBe('dark')
    expect(localStorage.getItem('theme')).toBe('dark')
  })

  it('always reports dark theme', () => {
    document.documentElement.classList.remove('dark')

    expect(detectPermanentTheme()).toBe('dark')
  })
})
```

- [ ] **Step 2: Update embedded URL tests for dark-only behavior**

Change `frontend/src/utils/__tests__/embedded-url.spec.ts` so the second test expects dark:

```ts
it('omits optional params when they are empty', () => {
  const result = buildEmbeddedUrl('https://pay.example.com/checkout', undefined, '', 'dark')

  const url = new URL(result)
  expect(url.searchParams.get('theme')).toBe('dark')
  expect(url.searchParams.get('ui_mode')).toBe('embedded')
  expect(url.searchParams.has('user_id')).toBe(false)
  expect(url.searchParams.has('token')).toBe(false)
  expect(url.searchParams.has('lang')).toBe(false)
})

it('detects permanent dark mode even if document root class is missing', () => {
  document.documentElement.classList.remove('dark')
  expect(detectTheme()).toBe('dark')
})
```

- [ ] **Step 3: Run the focused tests to verify failure**

Run:

```bash
pnpm --dir frontend test:run -- src/utils/__tests__/theme.spec.ts src/utils/__tests__/embedded-url.spec.ts
```

Expected: FAIL because `../theme` does not exist and `detectTheme()` can still return light.

- [ ] **Step 4: Implement permanent theme utility**

Create `frontend/src/utils/theme.ts`:

```ts
export type AppTheme = 'dark'

export function applyPermanentDarkTheme(): AppTheme {
  if (typeof document !== 'undefined') {
    document.documentElement.classList.add('dark')
    document.documentElement.style.colorScheme = 'dark'
  }
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('theme', 'dark')
  }
  return 'dark'
}

export function detectPermanentTheme(): AppTheme {
  return 'dark'
}
```

- [ ] **Step 5: Use the utility during bootstrap**

Replace `initThemeClass()` in `frontend/src/main.ts` with:

```ts
import { applyPermanentDarkTheme } from '@/utils/theme'
```

and call it inside `bootstrap()`:

```ts
async function bootstrap() {
  applyPermanentDarkTheme()

  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)
```

Remove the old `initThemeClass()` function.

- [ ] **Step 6: Make embedded URL theme dark-only**

Update `frontend/src/utils/embedded-url.ts`:

```ts
import { detectPermanentTheme, type AppTheme } from './theme'
```

Change the `buildEmbeddedUrl()` signature:

```ts
export function buildEmbeddedUrl(
  baseUrl: string,
  userId?: number,
  authToken?: string | null,
  theme: AppTheme = detectPermanentTheme(),
  lang?: string,
): string {
```

Change `detectTheme()`:

```ts
export function detectTheme(): AppTheme {
  return detectPermanentTheme()
}
```

- [ ] **Step 7: Run focused tests to verify pass**

Run:

```bash
pnpm --dir frontend test:run -- src/utils/__tests__/theme.spec.ts src/utils/__tests__/embedded-url.spec.ts
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/utils/theme.ts frontend/src/utils/__tests__/theme.spec.ts frontend/src/main.ts frontend/src/utils/embedded-url.ts frontend/src/utils/__tests__/embedded-url.spec.ts
git commit -m "feat(frontend): enforce permanent dark theme"
```

## Task 2: Tailwind Tokens And Global Neon Surfaces

**Files:**
- Modify: `frontend/tailwind.config.js`
- Modify: `frontend/src/style.css`
- Create: `frontend/src/components/layout/__tests__/NeonDesignTokens.spec.ts`

- [ ] **Step 1: Write failing token tests**

Create `frontend/src/components/layout/__tests__/NeonDesignTokens.spec.ts`:

```ts
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../../..')
const tailwindSource = readFileSync(resolve(root, 'tailwind.config.js'), 'utf8')
const styleSource = readFileSync(resolve(root, 'src/style.css'), 'utf8')

describe('dark neon design tokens', () => {
  it('defines charcoal surfaces and neon accents in Tailwind', () => {
    expect(tailwindSource).toContain('surface:')
    expect(tailwindSource).toContain('neon:')
    expect(tailwindSource).toContain("'neo-inset'")
    expect(tailwindSource).toContain("'neon-cyan'")
  })

  it('defines global dark neon CSS variables and reusable surfaces', () => {
    expect(styleSource).toContain('--color-surface-950')
    expect(styleSource).toContain('--color-neon-cyan')
    expect(styleSource).toContain('.neon-page')
    expect(styleSource).toContain('.neo-panel')
    expect(styleSource).toContain('.neo-control')
  })
})
```

- [ ] **Step 2: Run token tests to verify failure**

Run:

```bash
pnpm --dir frontend test:run -- src/components/layout/__tests__/NeonDesignTokens.spec.ts
```

Expected: FAIL because the new token names are not defined.

- [ ] **Step 3: Add Tailwind tokens**

In `frontend/tailwind.config.js`, extend `colors`, `boxShadow`, and `backgroundImage` with:

```js
surface: {
  50: '#f7f8fb',
  100: '#e8ebf2',
  200: '#cfd5e2',
  300: '#a8b3c7',
  400: '#77859d',
  500: '#566176',
  600: '#3a4151',
  700: '#252a35',
  800: '#181b24',
  900: '#10121a',
  950: '#07080d'
},
neon: {
  cyan: '#20e6ff',
  blue: '#4d8dff',
  pink: '#ff4ecd',
  lime: '#9cff64',
  orange: '#ffb45e',
  purple: '#b278ff'
}
```

Add shadows:

```js
'neo': '10px 10px 24px rgba(0,0,0,0.52), -8px -8px 20px rgba(105,130,180,0.08), inset 0 1px 0 rgba(255,255,255,0.06)',
'neo-sm': '6px 6px 16px rgba(0,0,0,0.44), -5px -5px 14px rgba(105,130,180,0.06)',
'neo-inset': 'inset 6px 6px 14px rgba(0,0,0,0.48), inset -4px -4px 10px rgba(255,255,255,0.04)',
'neon-cyan': '0 0 26px rgba(32,230,255,0.28)',
'neon-pink': '0 0 26px rgba(255,78,205,0.25)',
'neon-lime': '0 0 24px rgba(156,255,100,0.22)',
'neon-orange': '0 0 24px rgba(255,180,94,0.22)',
'neon-purple': '0 0 26px rgba(178,120,255,0.24)'
```

Add background images:

```js
'neon-grid':
  'linear-gradient(rgba(32,230,255,0.035) 1px, transparent 1px), linear-gradient(90deg, rgba(255,78,205,0.03) 1px, transparent 1px)',
'status-cold': 'conic-gradient(from 180deg, #20e6ff, #4d8dff, rgba(255,255,255,0.08))',
'status-warm': 'conic-gradient(from 180deg, #ffb45e, #ff4ecd, rgba(255,255,255,0.08))',
'status-healthy': 'conic-gradient(from 180deg, #9cff64, #20e6ff, rgba(255,255,255,0.08))',
'status-risk': 'conic-gradient(from 180deg, #ff4ecd, #ffb45e, rgba(255,255,255,0.08))'
```

- [ ] **Step 4: Rewrite global base and component classes**

In `frontend/src/style.css`, add CSS variables under `@layer base`:

```css
:root {
  color-scheme: dark;
  --color-surface-950: #07080d;
  --color-surface-900: #10121a;
  --color-surface-800: #181b24;
  --color-surface-700: #252a35;
  --color-neon-cyan: #20e6ff;
  --color-neon-pink: #ff4ecd;
  --color-neon-lime: #9cff64;
  --color-neon-orange: #ffb45e;
  --color-neon-purple: #b278ff;
}

body {
  @apply min-h-screen bg-surface-950 text-slate-100;
  background:
    radial-gradient(circle at 14% 8%, rgba(32, 230, 255, 0.16), transparent 28rem),
    radial-gradient(circle at 84% 12%, rgba(255, 78, 205, 0.13), transparent 30rem),
    linear-gradient(135deg, #07080d 0%, #10121a 52%, #151722 100%);
}
```

Replace the existing `.btn`, `.input`, `.glass`, `.card`, `.table`, `.badge`, `.dropdown`, `.modal-*`, `.dialog-*`, `.tabs`, `.progress`, `.switch`, `.sidebar-*`, `.page-*`, `.empty-state`, and `.skeleton` blocks with dark-only styles using the class targets in this plan.

- [ ] **Step 5: Run token tests and Tailwind build check**

Run:

```bash
pnpm --dir frontend test:run -- src/components/layout/__tests__/NeonDesignTokens.spec.ts
pnpm --dir frontend typecheck
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/tailwind.config.js frontend/src/style.css frontend/src/components/layout/__tests__/NeonDesignTokens.spec.ts
git commit -m "feat(frontend): add dark neon design system tokens"
```

## Task 3: Status Ring And Shared Control Components

**Files:**
- Create: `frontend/src/components/common/StatusRing.vue`
- Create: `frontend/src/components/common/__tests__/StatusRing.spec.ts`
- Modify: `frontend/src/components/common/index.ts`
- Modify: `frontend/src/components/common/Input.vue`
- Modify: `frontend/src/components/common/SearchInput.vue`
- Modify: `frontend/src/components/common/TextArea.vue`
- Modify: `frontend/src/components/common/Toggle.vue`
- Modify: `frontend/src/components/common/Select.vue`
- Modify: `frontend/src/components/common/DateRangePicker.vue`
- Modify: `frontend/src/components/common/Pagination.vue`
- Modify: `frontend/src/components/common/DataTable.vue`
- Modify: `frontend/src/components/common/StatusBadge.vue`
- Modify: `frontend/src/components/common/LoadingSpinner.vue`
- Modify: `frontend/src/components/common/Skeleton.vue`
- Modify: `frontend/src/components/common/BaseDialog.vue`
- Modify: `frontend/src/components/common/Toast.vue`
- Modify: `frontend/src/components/common/ConfirmDialog.vue`
- Modify: `frontend/src/components/common/ExportProgressDialog.vue`
- Modify: `frontend/src/components/common/SubscriptionProgressMini.vue`
- Modify: `frontend/src/components/common/NavigationProgress.vue`
- Modify: `frontend/src/components/common/AnnouncementBell.vue`

- [ ] **Step 1: Write failing StatusRing tests**

Create `frontend/src/components/common/__tests__/StatusRing.spec.ts`:

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StatusRing from '../StatusRing.vue'

describe('StatusRing', () => {
  it('renders a clamped percentage and accessible label', () => {
    const wrapper = mount(StatusRing, {
      props: {
        value: 128,
        label: 'Quota used',
        tone: 'warm'
      }
    })

    expect(wrapper.text()).toContain('100%')
    expect(wrapper.attributes('aria-label')).toBe('Quota used: 100%')
  })

  it('exposes tone and progress through CSS variables', () => {
    const wrapper = mount(StatusRing, {
      props: {
        value: 42,
        label: 'Healthy usage',
        tone: 'healthy'
      }
    })

    expect(wrapper.classes()).toContain('status-ring-healthy')
    expect(wrapper.attributes('style')).toContain('--status-ring-progress: 42%')
  })
})
```

- [ ] **Step 2: Run StatusRing tests to verify failure**

Run:

```bash
pnpm --dir frontend test:run -- src/components/common/__tests__/StatusRing.spec.ts
```

Expected: FAIL because `StatusRing.vue` does not exist.

- [ ] **Step 3: Implement StatusRing**

Create `frontend/src/components/common/StatusRing.vue`:

```vue
<template>
  <div
    class="status-ring"
    :class="toneClass"
    :style="ringStyle"
    role="img"
    :aria-label="`${label}: ${clampedValue}%`"
  >
    <div class="status-ring-core">
      <span class="status-ring-value">{{ clampedValue }}%</span>
      <span v-if="caption" class="status-ring-caption">{{ caption }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type StatusRingTone = 'cold' | 'warm' | 'healthy' | 'risk'

const props = withDefaults(
  defineProps<{
    value: number
    label: string
    caption?: string
    tone?: StatusRingTone
  }>(),
  {
    caption: '',
    tone: 'cold'
  }
)

const clampedValue = computed(() => Math.min(100, Math.max(0, Math.round(props.value))))
const toneClass = computed(() => `status-ring-${props.tone}`)
const ringStyle = computed(() => ({
  '--status-ring-progress': `${clampedValue.value}%`
}))
</script>
```

Add to `frontend/src/components/common/index.ts`:

```ts
export { default as StatusRing } from './StatusRing.vue'
```

- [ ] **Step 4: Add StatusRing CSS**

Add to `frontend/src/style.css`:

```css
.status-ring {
  @apply grid h-16 w-16 place-items-center rounded-full p-1.5;
  background: conic-gradient(var(--status-ring-start) 0 var(--status-ring-progress), rgba(255, 255, 255, 0.08) var(--status-ring-progress) 100%);
  box-shadow: 0 0 24px color-mix(in srgb, var(--status-ring-start) 35%, transparent);
}

.status-ring-core {
  @apply grid h-full w-full place-items-center rounded-full bg-surface-950 text-center shadow-neo-inset;
}

.status-ring-value {
  @apply text-xs font-black leading-none text-white;
}

.status-ring-caption {
  @apply mt-0.5 block text-[10px] leading-none text-slate-400;
}

.status-ring-cold { --status-ring-start: var(--color-neon-cyan); }
.status-ring-warm { --status-ring-start: var(--color-neon-pink); }
.status-ring-healthy { --status-ring-start: var(--color-neon-lime); }
.status-ring-risk { --status-ring-start: var(--color-neon-orange); }
```

- [ ] **Step 5: Convert common controls to dark-neon classes**

Use these replacements:

```vue
<!-- Input.vue input class core -->
'input w-full transition-all duration-200',
disabled ? 'cursor-not-allowed opacity-50' : ''
```

```vue
<!-- Toggle.vue button class core -->
class="neo-control relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-neon-cyan/50"
:class="[modelValue ? 'bg-neon-cyan/30 shadow-neon-cyan' : 'bg-surface-950/70']"
```

```vue
<!-- Toggle.vue thumb class core -->
class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-slate-100 shadow-neo-sm transition duration-200 ease-in-out"
```

For `Select.vue`, `DateRangePicker.vue`, `Pagination.vue`, `DataTable.vue`, `BaseDialog.vue`, and `Toast.vue`, replace hard light classes with `.neo-panel`, `.neo-control`, `.dropdown`, `.table`, `.modal-content`, `.dialog-container`, and `.toast` global classes.

- [ ] **Step 6: Run common component tests**

Run:

```bash
pnpm --dir frontend test:run -- src/components/common src/components/layout/__tests__/NeonDesignTokens.spec.ts
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/common frontend/src/style.css
git commit -m "feat(frontend): restyle common controls for neon dark ui"
```

## Task 4: App Chrome And Auth Layout

**Files:**
- Modify: `frontend/src/components/layout/AppLayout.vue`
- Modify: `frontend/src/components/layout/AppHeader.vue`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/components/layout/AuthLayout.vue`
- Modify: `frontend/src/components/layout/TablePageLayout.vue`
- Modify: `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`
- Modify: `frontend/src/styles/onboarding.css`
- Modify: `frontend/index.html`

- [ ] **Step 1: Extend AppSidebar tests**

Add to `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`:

```ts
describe('AppSidebar permanent dark theme chrome', () => {
  it('does not expose theme toggle code', () => {
    expect(componentSource).not.toContain('toggleTheme')
    expect(componentSource).not.toContain("localStorage.setItem('theme'")
    expect(componentSource).not.toContain('MoonIcon')
    expect(componentSource).not.toContain('SunIcon')
  })

  it('keeps support and collapse tools available', () => {
    expect(componentSource).toContain('@click="openContactDialog"')
    expect(componentSource).toContain('@click="toggleSidebar"')
  })
})
```

- [ ] **Step 2: Run AppSidebar tests to verify failure**

Run:

```bash
pnpm --dir frontend test:run -- src/components/layout/__tests__/AppSidebar.spec.ts
```

Expected: FAIL because the sidebar still contains theme toggle code.

- [ ] **Step 3: Update AppLayout background**

In `frontend/src/components/layout/AppLayout.vue`, change the root shell to:

```vue
<div class="app-shell min-h-screen bg-surface-950 text-slate-100">
  <div class="app-mesh pointer-events-none fixed inset-0 bg-neon-grid bg-[size:64px_64px] opacity-60"></div>
  <div class="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_15%_12%,rgba(32,230,255,0.15),transparent_32rem),radial-gradient(circle_at_84%_8%,rgba(255,78,205,0.12),transparent_34rem)]"></div>
```

Keep the existing sidebar, header, slot, padding prop, onboarding callback, and computed content class.

- [ ] **Step 4: Update AppHeader**

Change header classes to:

```vue
<header class="glass sticky top-0 z-30 border-b border-white/10 shadow-neo-sm">
  <div class="flex h-16 items-center justify-between px-4 md:px-6">
```

Change docs and balance elements to `.neo-control` or `.tool-pill` style classes while keeping `AnnouncementBell`, `LocaleSwitcher`, `SubscriptionProgressMini`, `docUrl`, `pageTitle`, and `pageDescription`.

- [ ] **Step 5: Remove theme toggle from AppSidebar**

Remove the theme button block from the sidebar tool row. The resulting row keeps contact and collapse:

```vue
<div class="sidebar-tool-row" :class="{ 'sidebar-tool-row-collapsed': sidebarCollapsed }">
  <button
    type="button"
    class="sidebar-tool-button"
    :title="t('common.contactSupport')"
    :aria-label="t('common.contactSupport')"
    @click="openContactDialog"
  >
    <Icon name="chat" size="md" />
  </button>

  <button
    type="button"
    class="sidebar-tool-button"
    :title="sidebarCollapsed ? t('nav.expand') : t('nav.collapse')"
    :aria-label="sidebarCollapsed ? t('nav.expand') : t('nav.collapse')"
    @click="toggleSidebar"
  >
    <ChevronDoubleLeftIcon v-if="!sidebarCollapsed" class="h-5 w-5 flex-shrink-0" />
    <ChevronDoubleRightIcon v-else class="h-5 w-5 flex-shrink-0" />
  </button>
</div>
```

Remove `isDark`, `toggleTheme()`, `SunIcon`, `MoonIcon`, and the theme initialization block from the script.

- [ ] **Step 6: Redesign AuthLayout without decorative orbs**

In `frontend/src/components/layout/AuthLayout.vue`, replace the background decoration with a non-orb grid and scanline treatment:

```vue
<div class="auth-background absolute inset-0 bg-surface-950"></div>
<div class="auth-decoration pointer-events-none absolute inset-0 overflow-hidden">
  <div class="absolute inset-0 bg-neon-grid bg-[size:56px_56px] opacity-60"></div>
  <div class="absolute inset-0 bg-[linear-gradient(180deg,rgba(255,255,255,0.045)_0,transparent_1px)] bg-[size:100%_7px] opacity-25"></div>
  <div class="absolute inset-0 bg-[radial-gradient(circle_at_20%_15%,rgba(32,230,255,0.18),transparent_28rem),radial-gradient(circle_at_80%_20%,rgba(255,78,205,0.14),transparent_30rem)]"></div>
</div>
```

Use `.neo-panel` for `auth-card` and keep slots, logo, site name, subtitle, and footer.

- [ ] **Step 7: Update onboarding and browser chrome colors**

In `frontend/index.html`, set:

```html
<meta name="theme-color" content="#07080d" />
```

In `frontend/src/styles/onboarding.css`, update `.driver-popover.theme-tour-popover` to dark glass, neon borders, and high-contrast text. Keep existing driver selectors and button behavior.

- [ ] **Step 8: Run layout tests**

Run:

```bash
pnpm --dir frontend test:run -- src/components/layout
pnpm --dir frontend typecheck
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add frontend/src/components/layout frontend/src/styles/onboarding.css frontend/index.html
git commit -m "feat(frontend): redesign app chrome for dark neon ui"
```

## Task 5: Dark-Neon Source Audit Script

**Files:**
- Create: `frontend/scripts/audit-dark-neon.mjs`
- Modify: `frontend/package.json`

- [ ] **Step 1: Add the audit script**

Create `frontend/scripts/audit-dark-neon.mjs`:

```js
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'

const root = process.cwd()

const groups = {
  publicAuth: [
    'src/views/HomeView.vue',
    'src/views/KeyUsageView.vue',
    'src/views/NotFoundView.vue',
    'src/views/setup',
    'src/views/auth',
    'src/views/public',
    'src/components/auth'
  ],
  user: [
    'src/views/user',
    'src/components/user',
    'src/components/keys',
    'src/components/charts'
  ],
  payment: [
    'src/components/payment',
    'src/components/admin/payment',
    'src/views/admin/orders'
  ],
  sales: [
    'src/views/sales'
  ],
  admin: [
    'src/views/admin',
    'src/components/admin',
    'src/components/account',
    'src/components/channels'
  ],
  ops: [
    'src/views/admin/ops'
  ]
}

const excludedByGroup = {
  admin: ['src/views/admin/ops']
}

const forbidden = [
  { name: 'white background', pattern: /\bbg-white\b/ },
  { name: 'light gray background', pattern: /\bbg-gray-(50|100|200)\b/ },
  { name: 'light gradient stop', pattern: /\b(from|via|to)-gray-(50|100|200)\b/ },
  { name: 'light border', pattern: /\bborder-gray-(100|200|300)\b/ },
  { name: 'dark text for light surface', pattern: /\btext-gray-(700|800|900)\b/ },
  { name: 'legacy blue primary glow', pattern: /\bshadow-primary-500\b/ }
]

function collect(entry) {
  const fullPath = join(root, entry)
  if (!existsSync(fullPath)) return []
  const stats = statSync(fullPath)
  if (stats.isFile()) return [fullPath]
  return readdirSync(fullPath).flatMap((name) => {
    const child = join(entry, name)
    const childFullPath = join(root, child)
    const childStats = statSync(childFullPath)
    if (childStats.isDirectory()) return collect(child)
    return /\.(vue|ts)$/.test(name) ? [childFullPath] : []
  })
}

const groupName = process.argv.includes('--group')
  ? process.argv[process.argv.indexOf('--group') + 1]
  : 'all'

const selectedEntries = groupName === 'all'
  ? Object.values(groups).flat()
  : groups[groupName]

if (!selectedEntries) {
  console.error(`Unknown group: ${groupName}`)
  console.error(`Known groups: ${Object.keys(groups).join(', ')}, all`)
  process.exit(2)
}

const violations = []

const exclusions = excludedByGroup[groupName] ?? []

for (const file of [...new Set(selectedEntries.flatMap(collect))]) {
  const relativeFile = relative(root, file).replaceAll('\\', '/')
  if (exclusions.some((entry) => relativeFile.startsWith(entry))) continue
  const source = readFileSync(file, 'utf8')
  const lines = source.split('\n')
  lines.forEach((line, index) => {
    forbidden.forEach((rule) => {
      if (rule.pattern.test(line)) {
        violations.push(`${relativeFile}:${index + 1} ${rule.name}: ${line.trim()}`)
      }
    })
  })
}

if (violations.length > 0) {
  console.error(`Dark neon audit failed for group "${groupName}" with ${violations.length} violation(s).`)
  console.error(violations.join('\n'))
  process.exit(1)
}

console.log(`Dark neon audit passed for group "${groupName}".`)
```

- [ ] **Step 2: Add package script**

In `frontend/package.json`, add:

```json
"audit:neon": "node scripts/audit-dark-neon.mjs"
```

- [ ] **Step 3: Run audit once to verify it detects existing public/auth violations**

Run:

```bash
pnpm --dir frontend audit:neon -- --group publicAuth
```

Expected: FAIL with one or more legacy light class violations.

- [ ] **Step 4: Commit**

```bash
git add frontend/scripts/audit-dark-neon.mjs frontend/package.json
git commit -m "test(frontend): add dark neon source audit"
```

## Task 6: Public, Setup, Auth, Key Usage, Legal, And 404 Pages

**Files:**
- Modify: `frontend/src/views/HomeView.vue`
- Modify: `frontend/src/views/KeyUsageView.vue`
- Modify: `frontend/src/views/NotFoundView.vue`
- Modify: `frontend/src/views/setup/SetupWizardView.vue`
- Modify: `frontend/src/views/auth/LoginView.vue`
- Modify: `frontend/src/views/auth/RegisterView.vue`
- Modify: `frontend/src/views/auth/EmailVerifyView.vue`
- Modify: `frontend/src/views/auth/ForgotPasswordView.vue`
- Modify: `frontend/src/views/auth/ResetPasswordView.vue`
- Modify: `frontend/src/views/auth/OAuthCallbackView.vue`
- Modify: `frontend/src/views/auth/LinuxDoCallbackView.vue`
- Modify: `frontend/src/views/auth/WechatCallbackView.vue`
- Modify: `frontend/src/views/auth/WechatPaymentCallbackView.vue`
- Modify: `frontend/src/views/auth/DingTalkCallbackView.vue`
- Modify: `frontend/src/views/auth/DingTalkEmailCompletionView.vue`
- Modify: `frontend/src/views/auth/OidcCallbackView.vue`
- Modify: `frontend/src/views/public/LegalDocumentView.vue`
- Modify: `frontend/src/components/auth/*.vue`

- [ ] **Step 1: Run public/auth audit to verify failure**

Run:

```bash
pnpm --dir frontend audit:neon -- --group publicAuth
```

Expected: FAIL with legacy light classes in public/auth pages.

- [ ] **Step 2: Remove standalone theme toggles from KeyUsageView**

In `frontend/src/views/KeyUsageView.vue`, remove the header theme toggle button and delete:

```ts
const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
```

Replace ring track logic with permanent dark:

```ts
const ringTrackColor = computed(() => '#11131c')
```

Remove the mounted block that reads `localStorage.getItem('theme')`.

- [ ] **Step 3: Redesign public page roots**

For each routed public/auth/setup page, change the outermost content wrapper to use dark-neon classes:

```vue
<div class="neon-page">
```

For cards and form panels, use:

```vue
class="neo-panel p-5 md:p-6"
```

For filter/search/input groups, use:

```vue
class="neo-control p-3"
```

For primary actions, use:

```vue
class="btn btn-primary"
```

For secondary links and back buttons, use:

```vue
class="btn btn-secondary"
```

- [ ] **Step 4: Preserve public/auth functions from inventory**

Manually verify these functions remain wired in the templates:

```text
/home: docs/login/register links, mobile menu, Base URL copy, custom home HTML/iframe
/login: email/password, password visibility, Turnstile, agreement, forgot password, OAuth, TOTP
/register: invite code, promo code, referral code, Turnstile, agreement, email verification, OAuth
/email-verify: code input, resend, Turnstile resend, pending OAuth account creation
/forgot-password and /reset-password: Turnstile, token reset, password confirmation
/key-usage: API key visibility, query, date range, usage rings, language/docs
/legal/:documentId: Markdown/HTML render and home return
```

- [ ] **Step 5: Run focused behavior tests**

Run:

```bash
pnpm --dir frontend test:run -- src/views/auth src/views/__tests__/KeyUsageView.spec.ts src/components/auth
```

Expected: PASS.

- [ ] **Step 6: Run public/auth audit to verify pass**

Run:

```bash
pnpm --dir frontend audit:neon -- --group publicAuth
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/HomeView.vue frontend/src/views/KeyUsageView.vue frontend/src/views/NotFoundView.vue frontend/src/views/setup frontend/src/views/auth frontend/src/views/public frontend/src/components/auth
git commit -m "feat(frontend): redesign public and auth pages"
```

## Task 7: User Dashboard, Keys, Usage, Profile, Models, Monitor, Redeem, Affiliate, Subscriptions, And Custom Pages

**Files:**
- Modify: `frontend/src/views/user/DashboardView.vue`
- Modify: `frontend/src/components/user/dashboard/UserDashboardStats.vue`
- Modify: `frontend/src/components/user/dashboard/UserDashboardCharts.vue`
- Modify: `frontend/src/components/user/dashboard/UserDashboardRecentUsage.vue`
- Modify: `frontend/src/components/user/dashboard/UserDashboardQuickActions.vue`
- Modify: `frontend/src/views/user/KeysView.vue`
- Modify: `frontend/src/components/keys/UseKeyModal.vue`
- Modify: `frontend/src/components/keys/EndpointPopover.vue`
- Modify: `frontend/src/views/user/UsageView.vue`
- Modify: `frontend/src/components/user/UserErrorRequestsTable.vue`
- Modify: `frontend/src/components/user/UserErrorDetailModal.vue`
- Modify: `frontend/src/views/user/ProfileView.vue`
- Modify: `frontend/src/components/user/profile/*.vue`
- Modify: `frontend/src/views/user/AvailableChannelsView.vue`
- Modify: `frontend/src/components/channels/AvailableChannelsTable.vue`
- Modify: `frontend/src/components/channels/PricingRow.vue`
- Modify: `frontend/src/components/channels/PricingEntryCard.vue`
- Modify: `frontend/src/views/user/ChannelStatusView.vue`
- Modify: `frontend/src/components/user/monitor/*.vue`
- Modify: `frontend/src/views/user/RedeemView.vue`
- Modify: `frontend/src/views/user/AffiliateView.vue`
- Modify: `frontend/src/views/user/SubscriptionsView.vue`
- Modify: `frontend/src/views/user/PlaygroundView.vue`
- Modify: `frontend/src/views/user/CustomPageView.vue`
- Modify: `frontend/src/components/charts/*.vue`

- [ ] **Step 1: Run user audit to verify failure**

Run:

```bash
pnpm --dir frontend audit:neon -- --group user
```

Expected: FAIL with legacy user surface classes.

- [ ] **Step 2: Redesign user dashboard modules**

Use `StatusRing` in dashboard stat/quota summaries:

```vue
<StatusRing
  :value="quotaPercent"
  :label="quotaLabel"
  :tone="quotaPercent >= 85 ? 'warm' : 'healthy'"
/>
```

Keep `loadStats()`, `loadCharts()`, `loadRecent()`, `loadPlatformQuotas()`, `refreshAll()`, date range state, and API calls unchanged.

- [ ] **Step 3: Redesign API key workflows**

Keep create/edit/delete/copy/reset/enable/disable/IP-list/quota/rate-limit/expiry/group/import-client functions. Apply:

```vue
class="neo-toolbar"
```

to search/filter/action bars, and:

```vue
class="neo-panel p-4"
```

to key cards, modal panels, and usage guide blocks.

- [ ] **Step 4: Redesign user usage and errors**

Preserve key/date filters, stats cards, usage table, error tab, pagination, CSV export, and error detail modal. Use `.table-container` and `.table` for desktop tables, and `.neo-panel` for mobile data cards.

- [ ] **Step 5: Redesign profile and subscription pages**

For profile cards, use `.neo-panel`; for sensitive actions such as password, TOTP disable, and unlinking identity providers, use `.neo-danger-zone`. Keep avatar, password, third-party binding, notification email, TOTP setup/disable, and support info.

- [ ] **Step 6: Redesign models and monitor pages**

Keep model search, platform/group filters, model details, copy model/API examples, monitor preloading, detail dialogs, availability rows, latency, and timeline. Use neon per-platform color accents from `frontend/src/utils/platformColors.ts`.

- [ ] **Step 7: Redesign redeem, affiliate, playground, subscriptions, and custom pages**

Keep redeem history, balance/subscription refresh, referral link copy, rebate transfer, iframe session creation, embedded chat iframe, subscription progress, custom page iframe/HTML/Markdown rendering, table-of-contents toggle, and code copy.

- [ ] **Step 8: Run focused user tests**

Run:

```bash
pnpm --dir frontend test:run -- src/views/user src/components/user src/components/keys src/components/channels src/components/charts
```

Expected: PASS.

- [ ] **Step 9: Run user audit to verify pass**

Run:

```bash
pnpm --dir frontend audit:neon -- --group user
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add frontend/src/views/user frontend/src/components/user frontend/src/components/keys frontend/src/components/channels frontend/src/components/charts
git commit -m "feat(frontend): redesign user pages for neon dark ui"
```

## Task 8: Payment And Orders

**Files:**
- Modify: `frontend/src/views/user/PaymentView.vue`
- Modify: `frontend/src/views/user/PaymentQRCodeView.vue`
- Modify: `frontend/src/views/user/PaymentResultView.vue`
- Modify: `frontend/src/views/user/StripePaymentView.vue`
- Modify: `frontend/src/views/user/StripePopupView.vue`
- Modify: `frontend/src/views/user/AirwallexPaymentView.vue`
- Modify: `frontend/src/views/user/UserOrdersView.vue`
- Modify: `frontend/src/components/payment/*.vue`
- Modify: `frontend/src/views/admin/orders/AdminPaymentDashboardView.vue`
- Modify: `frontend/src/views/admin/orders/AdminOrdersView.vue`
- Modify: `frontend/src/views/admin/orders/AdminPaymentPlansView.vue`
- Modify: `frontend/src/views/admin/orders/PlanEditDialog.vue`
- Modify: `frontend/src/components/admin/payment/*.vue`

- [ ] **Step 1: Run payment audit to verify failure**

Run:

```bash
pnpm --dir frontend audit:neon -- --group payment
```

Expected: FAIL with legacy payment/admin order surface classes.

- [ ] **Step 2: Force Stripe and Airwallex dark presentation**

In `frontend/src/views/user/StripePaymentView.vue` and `frontend/src/components/payment/StripePaymentInline.vue`, keep existing Stripe options but force:

```ts
appearance: {
  theme: 'night',
  variables: {
    borderRadius: '12px',
    colorPrimary: '#20e6ff',
    colorBackground: '#10121a',
    colorText: '#edf7ff'
  }
}
```

Keep payment intent creation, result handling, and popup/redirect logic.

- [ ] **Step 3: Redesign user purchase and payment state panels**

Keep balance recharge, subscription plan purchase, amount limits, method selection, fees, order creation, QR polling, cancellation, result verification, WeChat resume, Stripe, Airwallex, and help preview. Use `.neo-panel` for plan cards, `.neo-control` for amount/method selectors, and warm orange/pink accents for pending/error states.

- [ ] **Step 4: Redesign order tables and admin payment dashboards**

Keep filters, pagination, cancel pending orders, refund request/admin refund, retry, detail dialogs, plan create/edit/delete, and listed toggle. Use `.table-container`, `.table`, `.badge-*`, `.modal-content`, and neon chart containers.

- [ ] **Step 5: Run focused payment tests**

Run:

```bash
pnpm --dir frontend test:run -- src/views/user/__tests__/PaymentView.spec.ts src/views/user/__tests__/PaymentResultView.spec.ts src/views/user/__tests__/StripePaymentView.spec.ts src/views/user/__tests__/AirwallexPaymentView.spec.ts src/components/payment
```

Expected: PASS.

- [ ] **Step 6: Run payment audit to verify pass**

Run:

```bash
pnpm --dir frontend audit:neon -- --group payment
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/user/PaymentView.vue frontend/src/views/user/PaymentQRCodeView.vue frontend/src/views/user/PaymentResultView.vue frontend/src/views/user/StripePaymentView.vue frontend/src/views/user/StripePopupView.vue frontend/src/views/user/AirwallexPaymentView.vue frontend/src/views/user/UserOrdersView.vue frontend/src/components/payment frontend/src/views/admin/orders frontend/src/components/admin/payment
git commit -m "feat(frontend): redesign payment and order flows"
```

## Task 9: Sales Pages

**Files:**
- Modify: `frontend/src/views/sales/SalesDashboardView.vue`
- Modify: `frontend/src/views/sales/SalesCustomersView.vue`
- Modify: `frontend/src/views/sales/SalesCustomerOrdersView.vue`
- Modify: `frontend/src/views/sales/SalesOrdersView.vue`
- Modify: `frontend/src/views/sales/SalesReferralView.vue`

- [ ] **Step 1: Run sales audit to verify failure**

Run:

```bash
pnpm --dir frontend audit:neon -- --group sales
```

Expected: FAIL with legacy sales classes.

- [ ] **Step 2: Redesign sales dashboard and tables**

Keep today/7-day/30-day customers, orders, revenue stats, customer search, customer order links, order filters, pagination, referral link fetch/refresh/copy/open/disable/revoke, and inviter search. Use cyan for customer data, green for paid/order success, orange for pending orders, and pink for revoked/disabled referral states.

- [ ] **Step 3: Run sales audit and typecheck**

Run:

```bash
pnpm --dir frontend audit:neon -- --group sales
pnpm --dir frontend typecheck
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/sales
git commit -m "feat(frontend): redesign sales pages"
```

## Task 10: Admin Core Pages

**Files:**
- Modify: `frontend/src/views/admin/DashboardView.vue`
- Modify: `frontend/src/views/admin/UsersView.vue`
- Modify: `frontend/src/components/admin/user/*.vue`
- Modify: `frontend/src/views/admin/GroupsView.vue`
- Modify: `frontend/src/components/admin/group/*.vue`
- Modify: `frontend/src/views/admin/ChannelsView.vue`
- Modify: `frontend/src/components/admin/channel/*.vue`
- Modify: `frontend/src/views/admin/ChannelMonitorView.vue`
- Modify: `frontend/src/components/admin/monitor/*.vue`
- Modify: `frontend/src/views/admin/SubscriptionsView.vue`
- Modify: `frontend/src/views/admin/AccountsView.vue`
- Modify: `frontend/src/components/account/*.vue`
- Modify: `frontend/src/components/admin/account/*.vue`
- Modify: `frontend/src/views/admin/AnnouncementsView.vue`
- Modify: `frontend/src/components/admin/announcements/*.vue`
- Modify: `frontend/src/views/admin/ProxiesView.vue`
- Modify: `frontend/src/views/admin/RedeemView.vue`
- Modify: `frontend/src/views/admin/PromoCodesView.vue`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Modify: `frontend/src/views/admin/BackupView.vue`
- Modify: `frontend/src/views/admin/settings/EmailTemplateEditor.vue`
- Modify: `frontend/src/views/admin/RiskControlView.vue`
- Modify: `frontend/src/views/admin/UsageView.vue`
- Modify: `frontend/src/components/admin/usage/*.vue`
- Modify: `frontend/src/views/admin/affiliates/*.vue`

- [ ] **Step 1: Run admin audit to verify failure**

Run:

```bash
pnpm --dir frontend audit:neon -- --group admin
```

Expected: FAIL with legacy admin classes.

- [ ] **Step 2: Redesign admin dashboard**

Keep API key/account/request/user/cost/RPM/TPM/response-time stats, trends, model distribution, and user ranking. Replace stat cards with `.neo-panel` cards using data-specific neon accents and use `StatusRing` for high-level health/usage summaries.

- [ ] **Step 3: Redesign large data management pages**

For `UsersView.vue`, `GroupsView.vue`, `ChannelsView.vue`, `AccountsView.vue`, `SubscriptionsView.vue`, `ProxiesView.vue`, `RedeemView.vue`, `PromoCodesView.vue`, and `UsageView.vue`, use this structure:

```vue
<AppLayout>
  <div class="neon-page">
    <section class="neo-toolbar">
      <!-- existing search, filters, refresh, create, import, export, bulk actions -->
    </section>
    <section class="neo-panel overflow-hidden">
      <!-- existing DataTable/table/card list -->
    </section>
    <!-- existing modals and dialogs -->
  </div>
</AppLayout>
```

Keep all existing event handlers and API calls. Do not rename emitted events from modal components.

- [ ] **Step 4: Redesign settings, backup, risk, announcements, affiliates, and monitor**

Keep settings tabs, custom menu, legal docs, feature switches, auth/OAuth/Turnstile, user defaults, gateway policy, payment providers, SMTP/templates, Admin API key, backup settings, S3/R2 connection test, scheduled/manual backup, download/restore/delete, risk control tests, logs, unblock/delete hash actions, announcement target/read status, affiliate record tabs, and channel monitor template/apply/run result flows.

- [ ] **Step 5: Run focused admin tests**

Run:

```bash
pnpm --dir frontend test:run -- src/views/admin src/components/admin src/components/account
```

Expected: PASS.

- [ ] **Step 6: Run admin audit to verify pass**

Run:

```bash
pnpm --dir frontend audit:neon -- --group admin
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/admin frontend/src/components/admin frontend/src/components/account
git commit -m "feat(frontend): redesign admin core pages"
```

## Task 11: Admin Ops And Charts

**Files:**
- Modify: `frontend/src/views/admin/ops/OpsDashboard.vue`
- Modify: `frontend/src/views/admin/ops/components/*.vue`
- Modify: `frontend/src/views/admin/ops/utils/*.ts`
- Modify: `frontend/src/components/charts/TokenUsageTrend.vue`
- Modify: `frontend/src/components/charts/ModelDistributionChart.vue`
- Modify: `frontend/src/components/charts/GroupDistributionChart.vue`
- Modify: `frontend/src/components/charts/EndpointDistributionChart.vue`
- Modify: `frontend/src/components/admin/account/AccountStatsModal.vue`
- Modify: `frontend/src/components/account/AccountStatsModal.vue`

- [ ] **Step 1: Run ops audit to verify failure**

Run:

```bash
pnpm --dir frontend audit:neon -- --group ops
```

Expected: FAIL with legacy ops classes.

- [ ] **Step 2: Redesign ops dashboard as cyber monitoring console**

Keep time/platform/group/query-mode filters, fullscreen, throughput, switch rate, latency, error trend, error distribution, request details, error details, alert rules, alert events, runtime settings, email notification, system logs, concurrency, and OpenAI token stats.

Use this page shell:

```vue
<AppLayout :padded="false">
  <div class="neon-page p-4 md:p-6 lg:p-8">
    <!-- existing OpsDashboardHeader -->
    <div class="neon-grid grid-cols-1 xl:grid-cols-12">
      <!-- existing chart and table modules with neo-panel classes -->
    </div>
  </div>
</AppLayout>
```

- [ ] **Step 3: Force chart palettes to permanent dark**

For Chart.js components, replace conditional light/dark palettes with fixed colors:

```ts
const chartTheme = {
  text: '#cbd5e1',
  mutedText: '#94a3b8',
  grid: 'rgba(148, 163, 184, 0.14)',
  tooltipBackground: '#10121a',
  tooltipTitle: '#f8fafc',
  tooltipBody: '#cbd5e1',
  cyan: '#20e6ff',
  pink: '#ff4ecd',
  lime: '#9cff64',
  orange: '#ffb45e',
  purple: '#b278ff'
}
```

Keep all dataset construction, click handlers, and modal-opening behavior.

- [ ] **Step 4: Run focused ops/chart tests**

Run:

```bash
pnpm --dir frontend test:run -- src/views/admin/ops src/components/charts src/components/admin/account src/components/account
```

Expected: PASS.

- [ ] **Step 5: Run ops audit to verify pass**

Run:

```bash
pnpm --dir frontend audit:neon -- --group ops
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/admin/ops frontend/src/components/charts frontend/src/components/admin/account/AccountStatsModal.vue frontend/src/components/account/AccountStatsModal.vue
git commit -m "feat(frontend): redesign ops monitoring and charts"
```

## Task 12: Full Function Inventory Pass

**Files:**
- Read: `docs/frontend-ui-function-inventory.md`
- Modify migrated frontend files only when the inventory check finds a missing UI control or broken layout.

- [ ] **Step 1: Check every inventory row against migrated UI**

Use this command to keep the inventory open while checking routes:

```bash
sed -n '1,220p' docs/frontend-ui-function-inventory.md
```

For every row, verify the listed controls still exist in the migrated page or component source. If a control was visually hidden by mistake, restore it with the same click handler or route link it used before migration.

- [ ] **Step 2: Run all source audits**

Run:

```bash
pnpm --dir frontend audit:neon -- --group all
```

Expected: PASS.

- [ ] **Step 3: Run full automated verification**

Run:

```bash
pnpm --dir frontend typecheck
pnpm --dir frontend test:run
pnpm --dir frontend build
```

Expected: PASS for all three commands.

- [ ] **Step 4: Commit inventory fixes**

If Step 1 required changes, commit them:

```bash
git add frontend
git commit -m "fix(frontend): restore controls after neon ui migration"
```

If Step 1 required no changes, do not create an empty commit.

## Task 13: Browser Verification

**Files:**
- No planned file changes.

- [ ] **Step 1: Start a local frontend server**

Run:

```bash
pnpm --dir frontend dev -- --host 0.0.0.0
```

Expected: Vite prints a local URL such as `http://localhost:5173/`.

- [ ] **Step 2: Desktop route check**

Open these routes in a desktop viewport and verify no text overlap, no unreadable controls, no clipped dialog, and no missing main actions:

```text
/home
/login
/register
/key-usage
/dashboard
/keys
/usage
/purchase
/orders
/payment/qrcode
/payment/result
/admin/dashboard
/admin/ops
/admin/users
/admin/accounts
/admin/settings
/admin/usage
```

- [ ] **Step 3: Mobile route check**

Use a mobile viewport and verify sidebar drawer, header actions, tables, card lists, forms, and modals:

```text
/login
/dashboard
/keys
/usage
/purchase
/admin/dashboard
/admin/users
/admin/accounts
/admin/settings
```

- [ ] **Step 4: Stop the dev server**

Stop the Vite process with `Ctrl-C`. Confirm there is no running terminal session needed for the task.

## Task 14: Final Commit And Summary

**Files:**
- Modify only if browser verification finds issues.

- [ ] **Step 1: Inspect final diff**

Run:

```bash
git status --short
git diff --stat
```

Expected: frontend UI files, tests, and audit script only. Existing unrelated Docker/deploy changes may still be present and must remain uncommitted unless the user separately requests them.

- [ ] **Step 2: Commit final verification fixes**

If browser verification found and fixed issues:

```bash
git add frontend
git commit -m "fix(frontend): polish neon ui responsive states"
```

If no fixes were needed, do not create an empty commit.

- [ ] **Step 3: Report verification evidence**

Final response must include:

```text
Implemented full dark neon neumorphism UI redesign.
Verified with:
- pnpm --dir frontend typecheck
- pnpm --dir frontend test:run
- pnpm --dir frontend build
- pnpm --dir frontend audit:neon -- --group all
Manual browser checks covered desktop and mobile key routes.
```

If any command fails, report the exact failing command and the first actionable error instead of claiming completion.
