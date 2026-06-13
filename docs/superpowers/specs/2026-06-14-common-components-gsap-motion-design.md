# Common Components GSAP Motion Design

## Context

The frontend is a Vue 3 application using TailwindCSS and a shared component library under `frontend/src/components/common`. Current component motion is mostly CSS transitions and keyframes. The goal is to improve perceived quality with GSAP while keeping the first change scoped to the common component library.

Existing unrelated workspace changes in Docker/deploy files are outside this scope and must not be modified by this work.

## Chosen Approach

Use a targeted GSAP motion layer for common components.

Add a shared composable such as `frontend/src/composables/useGsapMotion.ts` to centralize motion presets and Vue Transition hook helpers. Components should call this module instead of embedding ad hoc `gsap.to()` calls throughout the library.

Add `gsap` as a frontend dependency. Use GSAP core only for this phase. Do not add ScrollTrigger, Flip, or other plugins.

## Scope

Initial component coverage:

- `BaseDialog.vue`; `ConfirmDialog.vue` benefits through `BaseDialog`.
- `Toast.vue`.
- `Select.vue` dropdown.
- `EmptyState.vue`.
- `StatCard.vue`.
- `DataTable.vue` only for lightweight mobile-card or empty-state entrance behavior.

Do not animate `DataTable` virtualized desktop rows. The table has careful non-zero height handling around `@tanstack/vue-virtual`, so the motion layer must not interfere with row measurement, scroll rect updates, or virtual padding.

## Motion Direction

The selected tone is expressive, but bounded for a data-heavy admin interface.

- Dialogs: fade in the overlay; enter the panel from `y: 18` and `scale: 0.96` with a subtle `back.out` ease. Exit should be faster and less elastic.
- Toasts: enter from the right with a slight scale change; stagger multiple active toasts; exit quickly to the right with fade.
- Select dropdowns: expand subtly from the trigger area; options may use a short stagger; close quickly.
- Empty states and stat cards: use a small upward entrance and short child-content stagger. Hover lift may be used only with transforms and must not change layout dimensions.
- Data table: avoid row-by-row desktop animation. Use only brief entrance behavior for mobile cards or empty-state transitions.

Use transform aliases (`x`, `y`, `scale`) and `autoAlpha` where elements should become non-interactive while hidden. Avoid animating layout-heavy properties such as width, height, top, or left unless there is no practical alternative.

## Reduced Motion

Respect `prefers-reduced-motion: reduce`.

When reduced motion is enabled, GSAP hooks should use `duration: 0` or a near-instant fade-only path. This behavior should live in the shared motion module so component code remains simple and consistent.

## Testing

Tests should focus on behavior and integration, not pixel-perfect animation output.

Required coverage:

- Dialog open/close still works, Escape behavior remains intact, and focus management is not broken.
- Toast rendering and removal still work.
- Select can open, select an option, close, and position its teleported dropdown.
- Reduced-motion code paths can be exercised without waiting for real animation timing.
- GSAP can be mocked in unit tests to avoid flaky timeline behavior.

Expected verification commands:

```bash
pnpm --dir frontend typecheck
pnpm --dir frontend test:run -- src/components/common
```

After implementation, perform a quick browser check using the running frontend service or a non-conflicting local port. Verify dialogs, toasts, select dropdowns, stat cards, empty states, and table states do not visibly overlap, jump, or break dark mode.

## Non-Goals

- No page-level route transitions in this phase.
- No layout/sidebar/header animation pass in this phase.
- No GSAP plugins in this phase.
- No broad redesign of the common component visual language.
- No changes to backend behavior.
