# Agent Task: Rebrand Web App - Landing Page

> **Parallel Execution:** ✅ Can run independently
> **Dependencies:** None
> **Estimated Time:** 2-3 hours

---

## Objective

Update all landing page templates with POPSigner branding, new copy, and **Bloomberg Terminal / HFT aesthetic**.

---

## Design Aesthetic: 1980s CRT Terminal

**IMPORTANT:** Authentic 80s Bloomberg terminal. Stranger Things S5 vibes. CRT phosphor glow.

### Visual Direction
- **CRT phosphor colors** - amber and green glow on black
- **True black background** (`#000000`) - like a CRT monitor
- **Amber phosphor** (`#FFB000`) - primary text/headlines
- **Green phosphor** (`#33FF00`) - data, secondary text
- **Red terminal** (`#FF3333`) - errors/alerts only
- **Dark mode ONLY** - CRTs were always black

### Color Palette
```
#FFB000 - Amber phosphor (primary)
#FFCC00 - Bright amber (highlights)
#33FF00 - Green phosphor (data)
#228B22 - Dark green (dimmed)
#1A4D1A - Very dark green (borders)
#FF3333 - Terminal red (alerts)
#333300 - Dark amber (borders)
#000000 - CRT black (background)
```

### Typography
- **MONOSPACE ONLY** - This is a terminal
- **IBM Plex Mono** or **VT323** for retro feel
- Phosphor glow: `text-shadow: 0 0 8px currentColor`

### UI Elements
- ALL CAPS for headers
- Square brackets for buttons: `[ DEPLOY ]`
- Phosphor glow on hover
- Borders in dark amber/green
- Optional scanline overlay

---

## Scope

### Files to Modify

| File | Changes |
|------|---------|
| `control-plane/templates/components/landing/nav.templ` | Logo, brand name |
| `control-plane/templates/components/landing/hero.templ` | Headline, copy, CTAs |
| `control-plane/templates/components/landing/problems.templ` | Reframe or remove |
| `control-plane/templates/components/landing/solution.templ` | New positioning |
| `control-plane/templates/components/landing/how_it_works.templ` | Remove time claims |
| `control-plane/templates/components/landing/features.templ` | Update features |
| `control-plane/templates/components/landing/pricing.templ` | New tiers €49/€499/€19,999 |
| `control-plane/templates/components/landing/cta.templ` | New CTA |
| `control-plane/templates/components/landing/footer.templ` | Brand, links |

---

## Copy Reference

Refer to `doc/design/DESIGN_SYSTEM.md` for approved copy.

### Forbidden Words (NEVER use)

- low-latency, fast, faster, high-performance
- speed, throughput, milliseconds, ms
- zero hops, zero network hops
- "Ring ring!", bell emoji (🔔)

### Approved Replacements

| Instead of | Use |
|------------|-----|
| speed | proximity, inline, on the execution path |
| edge | Point-of-Presence, where systems already run |
| performance | deterministic, predictable, non-blocking |
| scale | parallel, worker-native, burst-ready |

---

## Implementation

### nav.templ

```go
// Before
<span class="text-2xl">🔔</span>
<span class="...">BanhBaoRing</span>

// After - 80s CRT Terminal Nav
<nav class="bg-black border-b border-[#333300] sticky top-0 z-50 font-mono">
  <div class="max-w-6xl mx-auto px-6 py-4 flex justify-between items-center">
    <a href="/" class="flex items-center gap-2 group">
      <span class="text-[#FFB000] text-xl">◇</span>
      <span class="text-[#FFB000] font-bold uppercase tracking-wider
                   group-hover:text-shadow-[0_0_10px_#FFB000]">
        POPSIGNER
      </span>
    </a>
    
    <div class="flex items-center gap-8 text-sm uppercase">
      <a href="/docs" class="text-[#33FF00] hover:text-shadow-[0_0_8px_#33FF00]">DOCS</a>
      <a href="/pricing" class="text-[#33FF00] hover:text-shadow-[0_0_8px_#33FF00]">PRICING</a>
      <a href="https://github.com/..." class="text-[#33FF00] hover:text-shadow-[0_0_8px_#33FF00]">GITHUB</a>
      <a href="/login" class="text-[#666600] hover:text-[#FFB000]">LOGIN</a>
      <a href="/signup" 
         class="bg-[#FFB000] text-black font-bold px-4 py-2 
                hover:bg-[#FFCC00] hover:shadow-[0_0_15px_#FFB000]">
        [ DEPLOY ]
      </a>
    </div>
  </div>
</nav>
```

### hero.templ

```go
// Before
<div class="text-6xl mb-4 animate-bounce">🔔</div>
<h1>Ring ring!<br/>Sign where your infra lives.</h1>
<p>📍 Point of Presence key management for sovereign rollups.</p>
<p>Deploy next to your nodes. Same region. Same datacenter.</p>

// After - 80s CRT Terminal Hero
<section class="bg-black min-h-screen flex items-center font-mono">
  <!-- Optional: CRT scanline overlay -->
  <div class="absolute inset-0 pointer-events-none opacity-10
              bg-[repeating-linear-gradient(0deg,transparent,transparent_1px,rgba(0,0,0,0.3)_1px,rgba(0,0,0,0.3)_2px)]">
  </div>
  
  <div class="max-w-6xl mx-auto px-6 relative z-10">
    <!-- Blinking cursor effect -->
    <div class="text-[#33FF00] text-sm mb-4 opacity-70">
      > INITIALIZING POPSIGNER..._
    </div>
    
    <!-- Headline - Amber phosphor with glow -->
    <h1 class="text-5xl md:text-6xl font-bold uppercase tracking-wider
               text-[#FFB000] text-shadow-[0_0_20px_#FFB000]">
      POINT-OF-PRESENCE<br/>
      SIGNING INFRASTRUCTURE
    </h1>
    
    <!-- Subhead - Green phosphor -->
    <p class="text-xl text-[#33FF00] mt-8 max-w-2xl opacity-90">
      A DISTRIBUTED SIGNING LAYER DESIGNED TO LIVE INLINE WITH 
      EXECUTION—NOT BEHIND AN API QUEUE.
    </p>
    
    <!-- Secondary - Dimmed green -->
    <p class="text-lg text-[#228B22] mt-4">
      DEPLOY NEXT TO YOUR SYSTEMS. KEYS REMAIN REMOTE. YOU REMAIN SOVEREIGN.
    </p>
    
    <!-- CTAs - Terminal button style -->
    <div class="mt-12 flex gap-6">
      <a href="/signup" 
         class="bg-[#FFB000] text-black font-bold px-8 py-4 uppercase
                hover:bg-[#FFCC00] hover:shadow-[0_0_25px_#FFB000]
                transition-all duration-150">
        [ DEPLOY POPSIGNER ]
      </a>
      <a href="/docs" 
         class="border-2 border-[#33FF00] text-[#33FF00] px-8 py-4 uppercase
                hover:bg-[#33FF00] hover:text-black
                hover:shadow-[0_0_20px_#33FF00]
                transition-all duration-150">
        [ DOCUMENTATION ]
      </a>
    </div>
    
    <!-- Version/status bar -->
    <div class="mt-16 text-[#666600] text-xs">
      STATUS: OPERATIONAL | VERSION: 1.0.0 | (FORMERLY BANHBAORING)
    </div>
  </div>
</section>
```

### pricing.templ

```go
// Before
PricingTier{Name: "Free", Price: "$0", ...}
PricingTier{Name: "Pro", Price: "$49", ...}
PricingTier{Name: "Enterprise", Price: "Custom", ...}

// After - 80s CRT Terminal Pricing
<section class="bg-black py-24 font-mono">
  <div class="max-w-6xl mx-auto px-6">
    <h2 class="text-3xl text-[#FFB000] mb-4 uppercase text-shadow-[0_0_10px_#FFB000]">
      > PRICING_
    </h2>
    <p class="text-[#666600] mb-12">WE SELL PLACEMENT, NOT TRANSACTIONS.</p>
    
    <div class="grid md:grid-cols-3 gap-6">
      // Tier 1 - Shared (Green border)
      <div class="bg-black border border-[#1A4D1A] p-8 hover:border-[#33FF00]">
        <h3 class="text-lg text-[#228B22] mb-4 uppercase">SHARED</h3>
        <div class="text-4xl text-[#33FF00] mb-2 text-shadow-[0_0_8px_#33FF00]">
          €49<span class="text-lg text-[#228B22]">/MO</span>
        </div>
        <p class="text-[#336633] text-sm mb-6">SHARED POP INFRASTRUCTURE</p>
        <ul class="text-[#33FF00] text-sm space-y-2 mb-8 opacity-80">
          <li>> SHARED POINT-OF-PRESENCE</li>
          <li>> NO SLA</li>
          <li>> PLUGINS INCLUDED</li>
          <li>> EXIT GUARANTEE</li>
        </ul>
        <a href="/signup?plan=shared" 
           class="block text-center border border-[#228B22] py-2 text-[#33FF00] 
                  hover:bg-[#33FF00] hover:text-black uppercase">
          [ START SHARED ]
        </a>
      </div>
      
      // Tier 2 - Priority (Amber highlighted)
      <div class="bg-black border-2 border-[#FFB000] p-8 relative
                  shadow-[0_0_20px_rgba(255,176,0,0.3)]">
        <div class="absolute -top-3 left-6 bg-[#FFB000] text-black text-xs px-2 py-1 uppercase font-bold">
          RECOMMENDED
        </div>
        <h3 class="text-lg text-[#FFB000] mb-4 uppercase">PRIORITY</h3>
        <div class="text-4xl text-[#FFB000] mb-2 text-shadow-[0_0_15px_#FFB000]">
          €499<span class="text-lg text-[#CC8800]">/MO</span>
        </div>
        <p class="text-[#CC8800] text-sm mb-6">PRODUCTION WORKLOADS</p>
        <ul class="text-[#FFB000] text-sm space-y-2 mb-8 opacity-90">
          <li>> PRIORITY POP LANES</li>
          <li>> REGION SELECTION</li>
          <li>> 99.9% SLA</li>
          <li>> SELF-SERVE SCALING</li>
        </ul>
        <a href="/signup?plan=priority" 
           class="block text-center bg-[#FFB000] text-black py-2 font-bold uppercase
                  hover:bg-[#FFCC00] hover:shadow-[0_0_20px_#FFB000]">
          [ DEPLOY PRIORITY ]
        </a>
      </div>
      
      // Tier 3 - Dedicated (Green border)
      <div class="bg-black border border-[#1A4D1A] p-8 hover:border-[#33FF00]">
        <h3 class="text-lg text-[#228B22] mb-4 uppercase">DEDICATED</h3>
        <div class="text-4xl text-[#33FF00] mb-2 text-shadow-[0_0_8px_#33FF00]">
          €19,999<span class="text-lg text-[#228B22]">/MO</span>
        </div>
        <p class="text-[#336633] text-sm mb-6">DEDICATED INFRASTRUCTURE</p>
        <ul class="text-[#33FF00] text-sm space-y-2 mb-8 opacity-80">
          <li>> REGION-PINNED POP</li>
          <li>> CPU ISOLATION</li>
          <li>> 99.99% SLA</li>
          <li>> MANUAL ONBOARDING</li>
        </ul>
        <a href="/contact" 
           class="block text-center border border-neutral-700 py-2 text-neutral-300 hover:border-amber-600">
          Contact Us
        </a>
      </div>
    </div>
  </div>
</section>
```

### features.templ

Update feature cards with terminal aesthetic:

```go
// Before
@FeatureCard("⚡", "Parallel Workers", "Create multiple signing workers...")
@FeatureCard("📊", "Real-time Analytics", "Monitor signing operations...")

// After - Terminal style cards, no emojis in production
// Use simple geometric icons or text prefixes
<section class="bg-black py-24">
  <div class="max-w-6xl mx-auto px-6">
    <h2 class="font-mono text-3xl text-white mb-12">
      <span class="text-amber-500">_</span>capabilities
    </h2>
    
    <div class="grid md:grid-cols-3 gap-6">
      // Card template - terminal style
      <div class="bg-neutral-950 border border-neutral-800 p-6 hover:border-amber-600">
        <h3 class="font-mono text-amber-500 text-lg mb-2">inline_signing</h3>
        <p class="text-neutral-400">On the execution path, not behind a queue.</p>
      </div>
      
      <div class="bg-neutral-950 border border-neutral-800 p-6 hover:border-amber-600">
        <h3 class="font-mono text-amber-500 text-lg mb-2">exit_guarantee</h3>
        <p class="text-neutral-400">Export keys anytime. Sovereignty by default.</p>
      </div>
      
      <div class="bg-neutral-950 border border-neutral-800 p-6 hover:border-amber-600">
        <h3 class="font-mono text-amber-500 text-lg mb-2">plugin_architecture</h3>
        <p class="text-neutral-400">secp256k1 built-in. Bring your own algorithms.</p>
      </div>
      
      <div class="bg-neutral-950 border border-neutral-800 p-6 hover:border-amber-600">
        <h3 class="font-mono text-amber-500 text-lg mb-2">audit_trail</h3>
        <p class="text-neutral-400">Every signature logged. Compliance ready.</p>
      </div>
      
      <div class="bg-neutral-950 border border-neutral-800 p-6 hover:border-amber-600">
        <h3 class="font-mono text-amber-500 text-lg mb-2">kubernetes_native</h3>
        <p class="text-neutral-400">Helm charts, CRDs, GitOps ready.</p>
      </div>
      
      <div class="bg-neutral-950 border border-neutral-800 p-6 hover:border-amber-600">
        <h3 class="font-mono text-amber-500 text-lg mb-2">open_source</h3>
        <p class="text-neutral-400">Apache 2.0. Self-host forever.</p>
      </div>
    </div>
  </div>
</section>
```

### cta.templ

```go
// Before
<div class="text-7xl mb-6 animate-bounce">🔔</div>
<h2>Ready to secure your keys?</h2>
<p>Sign up free. First signature in 5 minutes.</p>

// After - Terminal aesthetic
<section class="bg-neutral-950 py-24 border-t border-neutral-800">
  <div class="max-w-4xl mx-auto px-6 text-center">
    <h2 class="font-mono text-3xl text-white mb-6">
      <span class="text-amber-500">$</span> deploy signing infrastructure
    </h2>
    <p class="text-neutral-400 mb-10">
      Keys remote. Signing inline. You sovereign.
    </p>
    <div class="flex justify-center gap-4">
      <a href="/signup" 
         class="bg-amber-600 text-black font-semibold px-8 py-3 hover:bg-amber-500">
        Deploy POPSigner →
      </a>
      <a href="/docs" 
         class="border border-neutral-700 text-neutral-300 px-8 py-3 hover:border-amber-600">
        Documentation
      </a>
    </div>
  </div>
</section>
```

### footer.templ

```go
// Before
<span>🔔</span><span>BanhBaoRing</span>

// After - Terminal minimal
<footer class="bg-black border-t border-neutral-900 py-12">
  <div class="max-w-6xl mx-auto px-6">
    <div class="flex justify-between items-center">
      <div>
        <span class="font-mono text-lg text-white">
          <span class="text-amber-500">◇</span> POPSigner
        </span>
        <p class="text-sm text-neutral-600 mt-1">
          Point-of-Presence signing infrastructure
        </p>
      </div>
      <div class="flex gap-8 text-sm text-neutral-500">
        <a href="/docs" class="hover:text-amber-500">Docs</a>
        <a href="https://github.com/..." class="hover:text-amber-500">GitHub</a>
        <a href="/contact" class="hover:text-amber-500">Contact</a>
      </div>
    </div>
    <div class="mt-8 pt-8 border-t border-neutral-900 text-xs text-neutral-700">
      © 2025 POPSigner. Apache 2.0.
    </div>
  </div>
</footer>
```

---

## After Editing Templates

Regenerate the Go files:

```bash
cd control-plane
templ generate
```

---

## Verification

```bash
cd control-plane

# Generate templates
templ generate

# Build
go build ./...

# Run locally and check visually
go run ./cmd/server

# Open http://localhost:8080 and verify:
# - No bell emoji
# - No "Ring ring!"
# - No time-based claims
# - Correct pricing (€49/€499/€19,999)
# - POPSigner branding throughout

# Check for remaining references
grep -r "banhbao" ./templates/ --include="*.templ"
grep -r "Ring ring" ./templates/ --include="*.templ"
grep -r "🔔" ./templates/ --include="*.templ"
```

---

## Checklist

```
□ nav.templ - logo, brand name
□ hero.templ - headline, copy, CTAs (no time claims)
□ problems.templ - reframe or convert to "what_it_is"
□ solution.templ - new positioning
□ how_it_works.templ - remove time badges
□ features.templ - new feature list
□ pricing.templ - €49/€499/€19,999 tiers
□ cta.templ - new CTA (no bell)
□ footer.templ - brand, copyright
□ templ generate passes
□ go build passes
□ Visual verification - no forbidden elements
□ No remaining "banhbao", "Ring ring", or 🔔 references
```

---

## Output

After completion, the landing page reflects POPSigner branding with infrastructure-focused copy.

