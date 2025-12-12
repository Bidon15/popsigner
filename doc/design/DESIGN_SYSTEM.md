# POPSigner Design System

> **POPSigner** — Point-of-Presence signing infrastructure.

---

## 1. Brand Identity

### 1.1 Product Name & Positioning

**POPSigner** — Point-of-Presence signing infrastructure. A distributed signing layer designed to live inline with execution, not behind an API queue.

POPSigner (formerly BanhBaoRing) reflects a clearer articulation of what the system is. The rename signals maturation from playful internal naming to category-defining infrastructure positioning.

### 1.2 Logo Concept

```
   ┌─────────────────────────┐
   │                         │
   │      ◇ POPSigner        │
   │                         │
   └─────────────────────────┘
```

- **Icon:** Geometric mark (diamond/node)—no emoji
- **Wordmark:** "POPSigner" in IBM Plex Sans or similar
- **Avoid:** Bell emoji, playful elements, crypto aesthetics

### 1.3 Taglines

| Context | Tagline |
|---------|---------|
| **Hero** | Point-of-Presence Signing Infrastructure |
| **Sub-hero** | Deploy inline with execution. Keys remain remote. You remain sovereign. |
| **Technical** | Distributed signing for rollups, bots, and infrastructure teams. |
| **One-liner** | Signing at the point of execution. |
| **Positioning** | We sell placement, not speed. Speed is a consequence. |

---

## 2. Value Proposition

### 2.1 Core Principles

| Principle | Description |
|-----------|-------------|
| **Inline Signing** | Signing happens on the execution path, not behind a queue |
| **Sovereignty by Default** | Keys are remote, but you control them. Export anytime. Exit anytime. |
| **Neutral Anchor** | Recovery data anchored to neutral data availability |

### 2.2 What POPSigner Is

- Point-of-Presence signing infrastructure
- A distributed signing layer
- Designed to live next to execution, not behind an API queue

### 2.3 What POPSigner Is Not

- A wallet
- MPC custody
- A consumer crypto product
- A compliance-first enterprise tool

### 2.4 Target Audience

- Senior backend engineers
- Infrastructure teams
- Rollup teams
- Execution bots / market makers

---

## 3. Language Constraints

### 3.1 Forbidden Words

The following words must **NEVER** appear in marketing copy:

- low-latency
- fast / faster
- high-performance
- speed
- throughput
- milliseconds / ms
- zero hops / zero network hops

### 3.2 Approved Replacements

| Instead of | Use |
|------------|-----|
| speed | proximity, inline, on the execution path |
| edge | Point-of-Presence, where systems already run |
| performance | deterministic, predictable, non-blocking |
| scale | parallel, worker-native, burst-ready |

### 3.3 Tone Guidelines

**Sound like:**
- Cloudflare
- Fastly
- Datadog

**Do not sound like:**
- Wallets
- Custody vendors
- Crypto dashboards
- VC pitch decks

---

## 4. Color Palette

### 4.1 Primary Colors

```css
:root {
  /* === PRIMARY: Professional indigo === */
  --primary-50: #eef2ff;
  --primary-100: #e0e7ff;
  --primary-200: #c7d2fe;
  --primary-300: #a5b4fc;
  --primary-400: #818cf8;
  --primary-500: #6366f1;    /* Main */
  --primary-600: #4f46e5;
  --primary-700: #4338ca;
  --primary-800: #3730a3;
  --primary-900: #312e81;
  
  /* === ACCENT: Subtle warmth === */
  --accent-50: #fff7ed;
  --accent-100: #ffedd5;
  --accent-200: #fed7aa;
  --accent-300: #fdba74;
  --accent-400: #fb923c;
  --accent-500: #f97316;     /* Main */
  --accent-600: #ea580c;
  --accent-700: #c2410c;
}
```

### 4.2 Semantic Colors

```css
:root {
  /* Success */
  --success-400: #4ade80;
  --success-500: #22c55e;
  --success-600: #16a34a;
  
  /* Warning */
  --warning-400: #facc15;
  --warning-500: #eab308;
  --warning-600: #ca8a04;
  
  /* Error */
  --error-400: #f87171;
  --error-500: #ef4444;
  --error-600: #dc2626;
}
```

### 4.3 Dark Theme (Primary)

```css
:root {
  /* Dark mode - default */
  --bg-primary: #0f0f10;     /* Near black */
  --bg-secondary: #18181b;   /* Card backgrounds */
  --bg-tertiary: #27272a;    /* Elevated surfaces */
  --bg-hover: #3f3f46;       /* Hover states */
  
  --text-primary: #fafafa;   /* Main text */
  --text-secondary: #a1a1aa; /* Muted text */
  --text-tertiary: #71717a;  /* Disabled text */
  
  --border: #3f3f46;         /* Borders */
  --border-hover: #52525b;   /* Hover borders */
}
```

### 4.4 Light Theme (Secondary)

```css
[data-theme="light"] {
  --bg-primary: #fafafa;
  --bg-secondary: #ffffff;
  --bg-tertiary: #f4f4f5;
  
  --text-primary: #18181b;
  --text-secondary: #52525b;
  
  --border: #e4e4e7;
}
```

---

## 5. Typography

### 5.1 Font Stack

```css
:root {
  /* Display & Body - Professional, infrastructure-focused */
  --font-display: "IBM Plex Sans", "Inter", system-ui, sans-serif;
  --font-body: "IBM Plex Sans", "Inter", system-ui, sans-serif;
  
  /* Monospace - code, addresses, keys */
  --font-mono: "JetBrains Mono", "Fira Code", "SF Mono", monospace;
}
```

### 5.2 Avoid

- Outfit (overused)
- Space Grotesk (overused in crypto)
- Playful or decorative fonts

### 5.3 Font Sizes (Tailwind scale)

| Name | Size | Line Height | Use Case |
|------|------|-------------|----------|
| `text-xs` | 12px | 16px | Labels, badges |
| `text-sm` | 14px | 20px | Secondary text |
| `text-base` | 16px | 24px | Body text |
| `text-lg` | 18px | 28px | Lead text |
| `text-xl` | 20px | 28px | Section headers |
| `text-2xl` | 24px | 32px | Card titles |
| `text-3xl` | 30px | 36px | Page headers |
| `text-4xl` | 36px | 40px | Hero subtitle |
| `text-5xl` | 48px | 48px | Hero headline |

---

## 6. Landing Page Design

### 6.1 Hero Section

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  ┌─── NAV ───────────────────────────────────────────────────────────────┐  │
│  │  ◇ POPSigner        Docs  Pricing  GitHub         [Sign In] [Deploy]  │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│                                                                             │
│         Point-of-Presence Signing Infrastructure                            │
│                                                                             │
│      A distributed signing layer designed to live inline with               │
│      execution—not behind an API queue.                                     │
│                                                                             │
│      Deploy next to your systems. Keys remain remote.                       │
│      You remain sovereign.                                                  │
│                                                                             │
│          ┌─────────────────────────────────────────────┐                    │
│          │  Deploy POPSigner →                         │                    │
│          └─────────────────────────────────────────────┘                    │
│                                                                             │
│          [Read the Architecture →]                                          │
│                                                                             │
│      (formerly BanhBaoRing)                                                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.2 What It Is Section

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│                         A signing layer, not a service.                     │
│                                                                             │
│   POPSigner is Point-of-Presence signing infrastructure. It deploys        │
│   where your systems already run—the same region, the same rack,            │
│   the same execution path.                                                  │
│                                                                             │
│   This isn't custody. This isn't MPC. This is signing at the               │
│   point of execution.                                                       │
│                                                                             │
│   ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐  │
│   │                     │ │                     │ │                     │  │
│   │   Inline Signing    │ │   Sovereignty       │ │   Neutral Anchor    │  │
│   │                     │ │                     │ │                     │  │
│   │   On the execution  │ │   Export anytime.   │ │   Recovery data     │  │
│   │   path, not behind  │ │   Exit anytime.     │ │   anchored to       │  │
│   │   a queue.          │ │   No lock-in.       │ │   neutral DA.       │  │
│   │                     │ │                     │ │                     │  │
│   └─────────────────────┘ └─────────────────────┘ └─────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.3 Pricing Section

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│      Three deployment models. Choose your isolation level.                  │
│                                                                             │
│   ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐  │
│   │                     │ │                     │ │                     │  │
│   │  SHARED POPSIGNER   │ │ PRIORITY POPSIGNER  │ │ DEDICATED POPSIGNER │  │
│   │                     │ │  ★ MOST POPULAR     │ │                     │  │
│   │      €49/month      │ │     €499/month      │ │   €19,999/month     │  │
│   │                     │ │                     │ │                     │  │
│   │   • Shared POP      │ │   • Priority lanes  │ │   • Region-pinned   │  │
│   │   • No SLA          │ │   • Region select   │ │   • CPU isolation   │  │
│   │   • Plugins         │ │   • 99.9% SLA       │ │   • 99.99% SLA      │  │
│   │   • Escape hatch    │ │   • Self-serve      │ │   • Manual onboard  │  │
│   │                     │ │                     │ │                     │  │
│   │  [Start with Shared]│ │  [Deploy Priority]  │ │  [Contact Us]       │  │
│   │                     │ │                     │ │                     │  │
│   └─────────────────────┘ └─────────────────────┘ └─────────────────────┘  │
│                                                                             │
│   Self-host option is always free. 100% open source.                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.4 CTA Section

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│       Deploy signing infrastructure that lives where you do.                │
│                                                                             │
│                  ┌───────────────────────────────┐                          │
│                  │     Deploy POPSigner →        │                          │
│                  └───────────────────────────────┘                          │
│                                                                             │
│                  [Read Documentation →]                                     │
│                                                                             │
│   ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐  │
│   │   Open Source       │ │   Built on OpenBao  │ │   Exit by Default   │  │
│   └─────────────────────┘ └─────────────────────┘ └─────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.5 Footer

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  ◇ POPSigner              Product        Developers       Company          │
│                           --------       -----------      --------         │
│  Point-of-Presence        Pricing        Documentation    About            │
│  Signing Infrastructure   Docs           SDK (Go)         Contact          │
│                           GitHub         SDK (Rust)                        │
│                           Status         API Reference                     │
│                                                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  © 2025 POPSigner. Open source under Apache 2.0.      [GitHub] [Discord]   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Component Library

### 7.1 Buttons

```html
<!-- Primary -->
<button class="
  bg-primary-600 hover:bg-primary-700
  text-white font-medium 
  px-6 py-3 rounded-lg
  transition-colors duration-200
">
  Deploy POPSigner →
</button>

<!-- Secondary - outline -->
<button class="
  border border-zinc-600 
  text-zinc-300
  px-5 py-2.5 rounded-lg
  hover:bg-zinc-800 hover:border-zinc-500
  transition-colors duration-200
">
  Read Documentation
</button>

<!-- Ghost -->
<button class="
  text-zinc-400 
  px-4 py-2 rounded-lg
  hover:text-white hover:bg-zinc-800
  transition-colors duration-200
">
  Cancel
</button>
```

### 7.2 Cards

```html
<!-- Feature card -->
<div class="
  bg-zinc-900
  border border-zinc-800 rounded-xl p-6
  hover:border-zinc-700
  transition-colors duration-200
">
  <h3 class="text-lg font-medium text-white mb-2">Inline Signing</h3>
  <p class="text-zinc-400 text-sm">On the execution path, not behind a queue.</p>
</div>
```

### 7.3 Code Blocks

```html
<!-- Code block -->
<div class="relative">
  <div class="absolute top-3 right-3 flex items-center gap-2">
    <span class="text-xs text-zinc-500 uppercase font-mono">Go</span>
    <button class="text-zinc-400 hover:text-white p-1.5 rounded">📋</button>
  </div>
  <pre class="bg-zinc-950 border border-zinc-800 rounded-lg p-6 overflow-x-auto">
    <code class="text-sm text-zinc-300">
client := popsigner.NewClient("psk_xxx")
sig, _ := client.Sign.Sign(ctx, keyID, txBytes, false)
    </code>
  </pre>
</div>
```

---

## 8. Page Layouts

### 8.1 Landing Page Layout

```
┌───────────────────────────────────────────────────────────────┐
│ Nav (fixed, minimal)                                          │
├───────────────────────────────────────────────────────────────┤
│ Hero (centered, text-focused)                                 │
├───────────────────────────────────────────────────────────────┤
│ What It Is (principles)                                       │
├───────────────────────────────────────────────────────────────┤
│ Architecture (diagram + code)                                 │
├───────────────────────────────────────────────────────────────┤
│ Exit Guarantee                                                │
├───────────────────────────────────────────────────────────────┤
│ Features (streamlined)                                        │
├───────────────────────────────────────────────────────────────┤
│ Pricing (3 tiers)                                             │
├───────────────────────────────────────────────────────────────┤
│ Final CTA                                                     │
├───────────────────────────────────────────────────────────────┤
│ Footer                                                        │
└───────────────────────────────────────────────────────────────┘
```

### 8.2 Dashboard Layout

```
┌───────────────────────────────────────────────────────────────┐
│ Top Bar (logo, search, user menu)                             │
├───────────────┬───────────────────────────────────────────────┤
│               │                                               │
│   Sidebar     │   Main Content                                │
│               │                                               │
│   Overview    │   ┌─────────────────────────────────────────┐ │
│   Keys        │   │  Page Content                           │ │
│   Usage       │   │                                         │ │
│   Audit       │   │                                         │ │
│   Settings    │   │                                         │ │
│               │   └─────────────────────────────────────────┘ │
│               │                                               │
└───────────────┴───────────────────────────────────────────────┘
```

---

## 9. Animation & Motion

### 9.1 Transition Defaults

```css
/* Keep animations subtle and professional */
.transition-fast { transition-duration: 150ms; }
.transition-normal { transition-duration: 200ms; }

/* Easing */
.ease-smooth { transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1); }
```

### 9.2 Hover Effects

```css
/* Button hover - subtle */
.btn:hover {
  background-color: var(--primary-700);
}

/* Card hover - border only */
.card:hover {
  border-color: var(--border-hover);
}
```

---

## 10. Accessibility

### 10.1 Requirements

- WCAG 2.1 AA compliance
- Color contrast ratio ≥ 4.5:1
- Full keyboard navigation
- Focus indicators on all interactive elements
- Screen reader support (ARIA labels)
- Reduced motion support (`prefers-reduced-motion`)

### 10.2 Focus Styles

```css
*:focus-visible {
  outline: 2px solid var(--primary-500);
  outline-offset: 2px;
}
```

---

## 11. Implementation Checklist

### Phase 1: Foundation
- [ ] Update branding from BanhBaoRing to POPSigner
- [ ] Remove bell emoji from all components
- [ ] Update color scheme to professional palette
- [ ] Update typography to IBM Plex Sans

### Phase 2: Landing Page
- [ ] Update hero copy (remove time claims)
- [ ] Add "What It Is" section
- [ ] Add "Exit Guarantee" section
- [ ] Update pricing to €49/€499/€19,999
- [ ] Update footer

### Phase 3: Dashboard
- [ ] Update branding throughout
- [ ] Add "Export Key" functionality visibility
- [ ] Update billing page with new tiers

### Phase 4: Documentation
- [ ] Update all docs with POPSigner naming
- [ ] Remove forbidden language throughout
- [ ] Update code examples with new API prefix (psk_)

---

## 12. References

- [Tailwind CSS](https://tailwindcss.com/docs)
- [HTMX](https://htmx.org/docs/)
- [Alpine.js](https://alpinejs.dev/start-here)
- [templ](https://templ.guide/)
