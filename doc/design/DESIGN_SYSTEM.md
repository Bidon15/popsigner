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

> **Aesthetic:** 1980s Bloomberg Terminal / CRT Phosphor
> 
> Think: Stranger Things S5 vibes. Retro CRT monitors. Amber and green phosphor glow on black.
> This is the authentic Bloomberg terminal from the 80s—not the modern orange.

### 4.1 Primary Colors (CRT Phosphor)

```css
:root {
  /* === PRIMARY: Amber Phosphor (Yellow-Orange CRT glow) === */
  --phosphor-amber: #FFB000;      /* Classic amber phosphor */
  --phosphor-amber-bright: #FFCC00; /* Highlighted amber */
  --phosphor-amber-dim: #CC8800;  /* Dimmed amber */
  
  /* === SECONDARY: Green Phosphor (the classic "green screen") === */
  --phosphor-green: #33FF00;      /* Bright phosphor green */
  --phosphor-green-dim: #228B22;  /* Forest/dark green */
  --phosphor-green-dark: #1A4D1A; /* Very dark green for backgrounds */
  
  /* === ACCENT: Terminal Red (function keys) === */
  --terminal-red: #FF3333;        /* Alert/stop red */
  --terminal-red-dim: #CC2222;    /* Dimmed red */
}
```

### 4.2 CRT Color Scale (Tailwind-compatible)

```css
:root {
  /* Amber scale (primary) */
  --amber-50: #FFFDF0;
  --amber-100: #FFF8CC;
  --amber-200: #FFEC99;
  --amber-300: #FFDD66;
  --amber-400: #FFCC00;   /* Bright highlight */
  --amber-500: #FFB000;   /* Main phosphor amber */
  --amber-600: #CC8800;   /* Dimmed */
  --amber-700: #996600;
  --amber-800: #664400;
  --amber-900: #332200;
  
  /* Green scale (secondary) */
  --green-50: #F0FFF0;
  --green-100: #CCFFCC;
  --green-200: #99FF99;
  --green-300: #66FF66;
  --green-400: #33FF00;   /* Bright phosphor green */
  --green-500: #22CC00;   /* Main green */
  --green-600: #228B22;   /* Forest green */
  --green-700: #1A6B1A;
  --green-800: #1A4D1A;   /* Dark green */
  --green-900: #0D260D;   /* Very dark */
}
```

### 4.3 Semantic Colors (80s Terminal)

```css
:root {
  /* Success - Phosphor Green */
  --success: #33FF00;
  --success-dim: #228B22;
  
  /* Warning - Phosphor Amber */
  --warning: #FFB000;
  --warning-dim: #CC8800;
  
  /* Error - Terminal Red */
  --error: #FF3333;
  --error-dim: #CC2222;
  
  /* Data highlights */
  --data-up: #33FF00;      /* Green - positive */
  --data-down: #FF3333;    /* Red - negative */
  --data-neutral: #FFB000; /* Amber - highlight */
}
```

### 4.4 Dark Theme (CRT Black - Default)

```css
:root {
  /* CRT Monitor Black */
  --bg-primary: #000000;     /* True CRT black */
  --bg-secondary: #0A0A0A;   /* Slightly elevated */
  --bg-tertiary: #111111;    /* Card backgrounds */
  --bg-hover: #1A1A1A;       /* Hover states */
  --bg-glow: #0D1A0D;        /* Subtle green tint (CRT bleed) */
  
  /* Text - Phosphor colors */
  --text-primary: #FFB000;   /* Amber phosphor (main text) */
  --text-secondary: #33FF00; /* Green phosphor (data) */
  --text-muted: #666600;     /* Dimmed amber */
  --text-dim: #336633;       /* Dimmed green */
  
  /* Borders - Subtle phosphor glow */
  --border: #333300;         /* Dark amber border */
  --border-green: #1A4D1A;   /* Dark green border */
  --border-hover: #FFB000;   /* Amber glow on hover */
}
```

### 4.5 Light Theme

```css
/* NO LIGHT THEME.
   CRT terminals were black. Period.
   The phosphor glows on darkness. */
```

### 4.6 CRT Effects (Optional)

```css
/* Scanline overlay */
.crt-scanlines {
  background: repeating-linear-gradient(
    0deg,
    rgba(0, 0, 0, 0.15),
    rgba(0, 0, 0, 0.15) 1px,
    transparent 1px,
    transparent 2px
  );
}

/* Phosphor glow effect */
.phosphor-glow {
  text-shadow: 0 0 5px currentColor, 0 0 10px currentColor;
}

/* CRT screen curve (subtle) */
.crt-curve {
  border-radius: 10px / 20px;
  box-shadow: inset 0 0 50px rgba(0, 0, 0, 0.5);
}
```

---

## 5. Typography

> **Aesthetic:** 80s Terminal. Monospace EVERYTHING. CRT vibes.

### 5.1 Font Stack

```css
:root {
  /* ALL TEXT should feel like a terminal */
  
  /* Primary - Classic terminal monospace */
  --font-terminal: "IBM Plex Mono", "Fira Code", "Courier New", monospace;
  
  /* Display - For large headlines (still mono, but can vary) */
  --font-display: "VT323", "IBM Plex Mono", "Press Start 2P", monospace;
  
  /* Body - Readable mono */
  --font-body: "IBM Plex Mono", "JetBrains Mono", monospace;
}
```

### 5.2 Typography Rules

- **EVERYTHING is monospace** - This is a terminal
- **Headlines:** Larger monospace, all caps optional
- **Body:** Standard monospace
- **Data/Keys/Addresses:** Monospace (obviously)
- **Numbers:** Tabular, fixed-width
- **Phosphor glow:** Add text-shadow for emphasis

### 5.3 Text Effects

```css
/* Phosphor glow for important text */
.text-glow {
  text-shadow: 0 0 8px currentColor;
}

/* Flickering effect (subtle) */
@keyframes flicker {
  0%, 100% { opacity: 1; }
  92% { opacity: 0.95; }
  94% { opacity: 0.9; }
  96% { opacity: 0.95; }
}

.text-flicker {
  animation: flicker 3s infinite;
}
```

### 5.4 Avoid

- Sans-serif fonts (not terminal)
- Rounded fonts
- Modern "clean" typography
- Variable fonts with personality

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

### 6.1 Hero Section (Terminal)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ bg-black                                                                    │
│  ┌─── NAV (border-b border-neutral-900) ─────────────────────────────────┐  │
│  │  ◇ POPSigner       Docs  Pricing  GitHub       [Login] ███Deploy███  │  │
│  │  (amber-500)       (neutral-400)               (text)  (amber-600)   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│                                                                             │
│         Point-of-Presence                                                   │
│         Signing Infrastructure  <-- font-mono, text-5xl                     │
│         ^^^^^^^^^^^^^^^^^^^^^^                                              │
│         (text-amber-500)                                                    │
│                                                                             │
│      A distributed signing layer designed to live inline with               │
│      execution—not behind an API queue. <-- text-neutral-400                │
│                                                                             │
│      Deploy next to your systems. Keys remain remote.                       │
│      You remain sovereign. <-- text-neutral-500                             │
│                                                                             │
│          ┌──────────────────────────────────────────┐                       │
│          │  Deploy POPSigner →                      │ bg-amber-600          │
│          └──────────────────────────────────────────┘ text-black            │
│                                                                             │
│          [Documentation] <-- border border-neutral-700                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.2 Capabilities Section (Terminal)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ bg-black                                                                    │
│                                                                             │
│   _capabilities  <-- font-mono, text-3xl, text-white                        │
│   ^                                                                         │
│   (amber-500)                                                               │
│                                                                             │
│   ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐  │
│   │ bg-neutral-950      │ │ border-neutral-800  │ │ hover:border-amber  │  │
│   │                     │ │                     │ │                     │  │
│   │ inline_signing      │ │ exit_guarantee      │ │ neutral_anchor      │  │
│   │ (font-mono amber)   │ │ (font-mono amber)   │ │ (font-mono amber)   │  │
│   │                     │ │                     │ │                     │  │
│   │ On the execution    │ │ Export anytime.     │ │ Recovery data       │  │
│   │ path, not behind    │ │ Exit anytime.       │ │ anchored to         │  │
│   │ a queue.            │ │ No lock-in.         │ │ neutral DA.         │  │
│   │ (text-neutral-400)  │ │ (text-neutral-400)  │ │ (text-neutral-400)  │  │
│   │                     │ │                     │ │                     │  │
│   └─────────────────────┘ └─────────────────────┘ └─────────────────────┘  │
│                                                                             │
│   ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐  │
│   │ plugin_arch         │ │ audit_trail         │ │ open_source         │  │
│   └─────────────────────┘ └─────────────────────┘ └─────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.3 Pricing Section (Terminal)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ bg-black                                                                    │
│                                                                             │
│   _pricing  <-- font-mono, text-3xl, text-white                             │
│   ^                                                                         │
│   (amber-500)                                                               │
│                                                                             │
│   We sell placement, not transactions.  <-- text-neutral-500                │
│                                                                             │
│   ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐  │
│   │ bg-neutral-950      │ │ border-2 amber-600  │ │ border-neutral-800  │  │
│   │ border-neutral-800  │ │ ┌ RECOMMENDED ┐     │ │                     │  │
│   │                     │ │ └─────────────┘     │ │                     │  │
│   │ shared (mono, gray) │ │ priority (amber)    │ │ dedicated (gray)    │  │
│   │                     │ │                     │ │                     │  │
│   │ €49                 │ │ €499                │ │ €19,999             │  │
│   │ /mo (font-mono)     │ │ /mo (font-mono)     │ │ /mo (font-mono)     │  │
│   │                     │ │                     │ │                     │  │
│   │ ▸ Shared POP        │ │ ▸ Priority lanes    │ │ ▸ Region-pinned     │  │
│   │ ▸ No SLA            │ │ ▸ Region select     │ │ ▸ CPU isolation     │  │
│   │ ▸ Plugins           │ │ ▸ 99.9% SLA         │ │ ▸ 99.99% SLA        │  │
│   │ ▸ Exit guarantee    │ │ ▸ Self-serve        │ │ ▸ Manual onboard    │  │
│   │                     │ │                     │ │                     │  │
│   │ [Start with Shared] │ │ ███Deploy Priority██│ │ [Contact Us]        │  │
│   │ (border outline)    │ │ (bg-amber-600)      │ │ (border outline)    │  │
│   │                     │ │                     │ │                     │  │
│   └─────────────────────┘ └─────────────────────┘ └─────────────────────┘  │
│                                                                             │
│   Self-host forever. Apache 2.0.  <-- text-neutral-600                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.4 CTA Section (Terminal)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ bg-neutral-950, border-t border-neutral-800                                 │
│                                                                             │
│   $ deploy signing infrastructure  <-- font-mono, text-3xl                  │
│   ^                                                                         │
│   (amber-500)                                                               │
│                                                                             │
│   Keys remote. Signing inline. You sovereign.  <-- text-neutral-400         │
│                                                                             │
│                  ┌───────────────────────────────┐                          │
│                  │     Deploy POPSigner →        │  bg-amber-600            │
│                  └───────────────────────────────┘  text-black              │
│                                                                             │
│                  [Documentation]  <-- border border-neutral-700             │
│                                                                             │
│   ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐  │
│   │   Open Source       │ │   Built on OpenBao  │ │   Exit by Default   │  │
│   └─────────────────────┘ └─────────────────────┘ └─────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.5 Footer (Terminal)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ bg-black, border-t border-neutral-900                                       │
│                                                                             │
│  ◇ POPSigner  <-- font-mono, amber-500 for diamond, white for text          │
│  Point-of-Presence signing infrastructure  <-- text-neutral-600, text-sm    │
│                                                                             │
│                            Docs    Pricing    GitHub    Contact             │
│                            ----    -------    ------    -------             │
│                            text-neutral-500, hover:text-amber-500           │
│                                                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│  border-t border-neutral-900                                                │
│                                                                             │
│  © 2025 POPSigner. Apache 2.0.  <-- text-neutral-700, text-xs               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Component Library

> **Aesthetic:** 1980s CRT Terminal. Phosphor glow. Amber + Green on black.

### 7.1 Buttons

```html
<!-- Primary - Amber Phosphor -->
<button class="
  bg-[#FFB000] hover:bg-[#FFCC00]
  text-black font-mono font-bold
  px-6 py-3 uppercase
  shadow-[0_0_10px_#FFB000]
  hover:shadow-[0_0_20px_#FFCC00]
">
  [ DEPLOY POPSIGNER ]
</button>

<!-- Secondary - Green Outline -->
<button class="
  border-2 border-[#33FF00]
  text-[#33FF00] font-mono
  px-5 py-2.5 uppercase
  hover:bg-[#33FF00] hover:text-black
  hover:shadow-[0_0_15px_#33FF00]
">
  [ DOCUMENTATION ]
</button>

<!-- Danger - Red -->
<button class="
  border-2 border-[#FF3333]
  text-[#FF3333] font-mono
  px-4 py-2 uppercase
  hover:bg-[#FF3333] hover:text-black
">
  [ CANCEL ]
</button>
```

### 7.2 Cards

```html
<!-- CRT Feature Card -->
<div class="
  bg-black
  border border-[#333300] p-6
  hover:border-[#FFB000]
  hover:shadow-[0_0_10px_rgba(255,176,0,0.3)]
">
  <h3 class="font-mono text-[#FFB000] text-lg mb-2 uppercase
             text-shadow-[0_0_8px_#FFB000]">
    INLINE_SIGNING
  </h3>
  <p class="font-mono text-[#33FF00] text-sm opacity-80">
    On the execution path, not behind a queue.
  </p>
</div>

<!-- Data Card (CRT Display) -->
<div class="
  bg-black
  border border-[#1A4D1A] p-4
">
  <div class="font-mono text-xs text-[#666600] mb-1">KEYS_ACTIVE</div>
  <div class="font-mono text-3xl text-[#33FF00] text-shadow-[0_0_10px_#33FF00]">247</div>
  <div class="font-mono text-xs text-[#33FF00] mt-1">▲ +12 TODAY</div>
</div>
```

### 7.3 Code Blocks

```html
<!-- CRT Code Block -->
<div class="relative bg-black border border-[#333300]">
  <!-- Header bar -->
  <div class="flex items-center justify-between border-b border-[#333300] px-4 py-2 bg-[#0A0A0A]">
    <span class="font-mono text-xs text-[#666600]">main.go</span>
    <button class="font-mono text-xs text-[#666600] hover:text-[#FFB000]">[COPY]</button>
  </div>
  <!-- Code with phosphor colors -->
  <pre class="p-4 overflow-x-auto font-mono text-sm">
<span class="text-[#FFB000]">client</span> := popsigner.<span class="text-[#FFCC00]">NewClient</span>(<span class="text-[#33FF00]">"psk_xxx"</span>)
<span class="text-[#FFB000]">sig</span>, _ := client.Sign.<span class="text-[#FFCC00]">Sign</span>(ctx, keyID, txBytes, <span class="text-[#33FF00]">false</span>)
  </pre>
</div>
```

### 7.4 Data Tables (CRT Terminal)

```html
<!-- CRT Data Table -->
<table class="w-full font-mono text-sm">
  <thead class="border-b border-[#333300]">
    <tr class="text-left text-xs text-[#FFB000] uppercase">
      <th class="px-4 py-3">NAME</th>
      <th class="px-4 py-3">ADDRESS</th>
      <th class="px-4 py-3">STATUS</th>
    </tr>
  </thead>
  <tbody class="text-[#33FF00]">
    <tr class="border-b border-[#1A1A1A] hover:bg-[#0D1A0D]">
      <td class="px-4 py-3">validator_1</td>
      <td class="px-4 py-3 opacity-70">celestia1abc...</td>
      <td class="px-4 py-3 text-[#33FF00] text-shadow-[0_0_5px_#33FF00]">ACTIVE</td>
    </tr>
    <tr class="border-b border-[#1A1A1A] hover:bg-[#0D1A0D]">
      <td class="px-4 py-3">validator_2</td>
      <td class="px-4 py-3 opacity-70">celestia1def...</td>
      <td class="px-4 py-3 text-[#FF3333]">OFFLINE</td>
    </tr>
  </tbody>
</table>
```

### 7.5 Status Indicators

```html
<!-- CRT Status Badges -->
<span class="font-mono text-xs px-2 py-1 text-[#33FF00] border border-[#33FF00] 
             shadow-[0_0_5px_#33FF00]">ACTIVE</span>
             
<span class="font-mono text-xs px-2 py-1 text-[#FF3333] border border-[#FF3333]
             shadow-[0_0_5px_#FF3333]">ERROR</span>
             
<span class="font-mono text-xs px-2 py-1 text-[#FFB000] border border-[#FFB000]
             shadow-[0_0_5px_#FFB000]">PENDING</span>

<span class="font-mono text-xs px-2 py-1 text-[#33FF00] border border-[#228B22]">
  EXIT_OK
</span>
```

### 7.6 CRT Screen Container

```html
<!-- Wrap content in CRT monitor frame -->
<div class="
  bg-black
  border-4 border-[#2A2A2A]
  rounded-lg
  p-1
  shadow-[inset_0_0_50px_rgba(0,0,0,0.5)]
">
  <!-- Optional scanlines overlay -->
  <div class="
    relative
    bg-[repeating-linear-gradient(0deg,rgba(0,0,0,0.1),rgba(0,0,0,0.1)_1px,transparent_1px,transparent_2px)]
  ">
    <!-- Content here -->
  </div>
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

### 8.2 Dashboard Layout (Terminal)

```
┌───────────────────────────────────────────────────────────────┐
│ ◇ POPSigner                          user@org ▾              │
├───────────────┬───────────────────────────────────────────────┤
│               │                                               │
│   _dashboard  │   $ _keys                                     │
│   _keys       │   ───────────────────────────────────────     │
│   _audit      │                                               │
│   _settings   │   name         address          status        │
│               │   ────         ───────          ──────        │
│               │   validator_1  celestia1abc...  ACTIVE        │
│               │   validator_2  celestia1def...  EXIT_OK       │
│               │                                               │
│   _export     │   [create_key →]                              │
│               │                                               │
└───────────────┴───────────────────────────────────────────────┘

Notes:
- Sidebar: bg-neutral-950, border-r border-neutral-800
- Nav items: font-mono, prefixed with underscore
- Active state: text-amber-500, bg-neutral-900
- Main content: bg-black
- Data tables: monospace, trading terminal style
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
