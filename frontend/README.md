# rashnu · Infrastructure Cost Manager

Track depreciation, colocation fees and **per-service cost** for owned (on-prem / colo) hardware — the inverse of a cloud billing dashboard. Built as a single-page React app.

This is a real, buildable project: **React 18 + TypeScript + Vite + Tailwind CSS v4**. It was converted from a single-file in-browser prototype into proper ES modules with full type coverage.

---

## Quick start

```bash
npm install
npm run dev      # start the dev server (http://localhost:5173)
npm run build    # type-check (tsc) + production build to dist/
npm run preview  # serve the production build locally
```

Requires Node 18+.

---

## What it does

- **Overview** — monthly/annual run-rate, cost composition (depreciation vs colo), spend by data center, top services, fleet utilization.
- **Hardware** — server / switch / storage inventory with straight-line amortization and colo allocated by power draw.
- **Data Centers** — facilities, colocation fees, power utilization, per-asset colo split.
- **Services** — Kubernetes + VM service catalog with Prometheus-style usage metrics and per-service cost attribution.
- **Cost Reports** — per-service cost broken down by team / platform / environment / DC, with CSV export.
- **Users** — role-based access (admin = read/write, viewer = read-only).

All data is **seeded sample data** held in React state and persisted to `localStorage`. There is no backend — edits live in the browser. "Reset sample data" (in the Tweaks panel) restores the original dataset.

### Demo logins
| Role | Email | Password |
|------|-------|----------|
| Admin | `arman@rashnnu.io` | `admin` |
| Viewer | `mehdi@rashnnu.io` | `viewer` |

---

## Project structure

```
src/
  main.tsx               # React entry
  App.tsx                # shell: routing, auth, top bar, nav, Tweaks panel
  index.css              # Tailwind v4 import + design-system component layer + theme tokens
  lib/
    types.ts             # all domain + context TypeScript types
    data.ts              # seed data + cost engine (depreciation, colo, allocation) — pure TS
    tweaks.ts            # useTweaks() — localStorage-backed settings hook
  state/
    useAppState.ts       # app state: data CRUD, auth, theme, derived costs (useMemo)
  components/
    icons.tsx            # <Icon> + icon path set
    ui.tsx               # Panel, Stat, Badge, Modal, Field, tables, sorting, toasts…
    charts.tsx           # dependency-free SVG charts (area, bar, donut, h-bars, stacked)
    TweaksPanel.tsx      # floating settings panel + control primitives
  pages/
    Login.tsx  Dashboard.tsx  Hardware.tsx  DataCenters.tsx
    Services.tsx  Reports.tsx  Users.tsx
```

The **cost engine** in `lib/data.ts` is plain TypeScript with no React dependency, so it can be unit-tested or reused on a server.

---

## The Tweaks panel

The floating ⚙ button (bottom-left) opens a settings panel that drives the live cost model and presentation:

- **Allocation method** — split each host's cost by CPU/RAM *usage*, reserved *requests*, *even* split, or *manual weight*.
- **CPU ⟷ RAM weight** — how much CPU vs RAM matters in usage/request allocation.
- **Idle headroom** — bill 100% of host cost to tenants, or hold back unused capacity as "idle".
- **Layout / viz / report view**, **currency** (Rial / Toman), **accent color**, **UI font**, **dark mode**.

These persist to `localStorage`.

---

## Styling

The project uses **Tailwind CSS v4** (via `@tailwindcss/vite`). Tailwind utilities are available throughout, and the app's design tokens are exposed to Tailwind through an `@theme inline` block in `index.css`, so utilities like `text-accent` or `bg-panel` resolve to the live themed CSS variables.

The mature, repeated components (buttons, badges, tables, panels, charts) are kept as a hand-authored component layer in `index.css` rather than long inline utility chains — the idiomatic approach for an established design system. Theme + accent + font are applied as CSS variables on `<html>` at runtime.

---

## Notes

- No router dependency — navigation is simple `useState` page switching (persisted), which keeps the bundle small. Swap in `react-router` if you add deep-linkable routes.
- Money values are large IRR figures, abbreviated (K/M/B/T) in the UI; full values appear in forms and CSV export.
- Tooling is **Vite** (the actively-maintained successor to Create React App, which the React team has deprecated).
