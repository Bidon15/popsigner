# Agent Task: Rebrand Web App - Dashboard & Auth

> **Parallel Execution:** ✅ Can run independently
> **Dependencies:** None
> **Estimated Time:** 1-2 hours

---

## Objective

Update dashboard, auth, and layout templates with POPSigner branding and **Bloomberg Terminal / HFT aesthetic**.

---

## Design Aesthetic: 1980s CRT Terminal

**CRITICAL:** Match the landing page 80s terminal aesthetic. Stranger Things S5 vibes.

### Visual Direction
- **CRT black background** (`#000000`)
- **Amber phosphor** (`#FFB000`) - primary text, headers
- **Green phosphor** (`#33FF00`) - data, success states
- **Terminal red** (`#FF3333`) - errors only
- **Dark mode ONLY** - CRTs were black

### Color Palette
```
#FFB000 - Amber (headers, highlights)
#FFCC00 - Bright amber (hover)
#33FF00 - Green phosphor (data)
#228B22 - Dark green (dimmed)
#1A4D1A - Very dark green (borders)
#FF3333 - Red (errors)
#333300 - Dark amber (borders)
```

### Dashboard Specific
- ALL CAPS for headers
- Monospace ONLY (IBM Plex Mono, VT323)
- Phosphor glow on important data
- Data tables like trading terminals
- Status indicators: `ACTIVE`, `OFFLINE`, `EXIT_OK`

### Typography
- **MONOSPACE ONLY** - This is a terminal
- Add glow: `text-shadow: 0 0 8px currentColor`

---

## Scope

### Files to Modify

| File | Changes |
|------|---------|
| `control-plane/templates/layouts/base.templ` | Title, meta tags |
| `control-plane/templates/layouts/dashboard.templ` | Branding |
| `control-plane/templates/layouts/auth.templ` | Logo, branding |
| `control-plane/templates/layouts/landing.templ` | Title |
| `control-plane/templates/components/sidebar.templ` | Logo |
| `control-plane/templates/pages/dashboard.templ` | Branding |
| `control-plane/templates/pages/login.templ` | Branding, copy |
| `control-plane/templates/pages/signup.templ` | Branding, copy |
| `control-plane/templates/pages/onboarding.templ` | Branding, copy |
| `control-plane/templates/pages/forgot_password.templ` | Branding |
| `control-plane/templates/pages/keys_list.templ` | Add export visibility |
| `control-plane/templates/pages/keys_detail.templ` | Add export action |

---

## Implementation

### layouts/base.templ

```go
// Before
<title>BanhBaoRing - { pageTitle }</title>
<meta name="description" content="BanhBaoRing - Secure key management">

// After
<title>POPSigner - { pageTitle }</title>
<meta name="description" content="POPSigner - Point-of-Presence signing infrastructure">
<meta name="keywords" content="signing, infrastructure, celestia, cosmos, keys">
```

### layouts/auth.templ

```go
// Before
<div class="logo">
    <span class="text-3xl">🔔</span>
    <span>BanhBaoRing</span>
</div>

// After - 80s CRT Auth Layout
<div class="min-h-screen bg-black flex items-center justify-center font-mono">
  <!-- Optional scanlines -->
  <div class="absolute inset-0 pointer-events-none opacity-5
              bg-[repeating-linear-gradient(0deg,transparent,transparent_1px,rgba(0,0,0,0.3)_1px,rgba(0,0,0,0.3)_2px)]">
  </div>
  
  <div class="max-w-md w-full relative z-10">
    <div class="text-center mb-8">
      <span class="text-[#FFB000] text-3xl">◇</span>
      <span class="text-[#FFB000] text-2xl ml-2 uppercase tracking-wider
                   text-shadow-[0_0_15px_#FFB000]">
        POPSIGNER
      </span>
    </div>
    { children... }
  </div>
</div>
```

### components/sidebar.templ

```go
// Before
<a href="/" class="logo">
    <span class="text-2xl">🔔</span>
    <span>BanhBaoRing</span>
</a>

// After - 80s CRT Sidebar
<aside class="w-64 bg-black border-r border-[#333300] min-h-screen font-mono">
  <div class="p-4 border-b border-[#333300]">
    <a href="/" class="flex items-center gap-2 group">
      <span class="text-[#FFB000] text-xl">◇</span>
      <span class="text-[#FFB000] font-bold uppercase tracking-wider
                   group-hover:text-shadow-[0_0_10px_#FFB000]">
        POPSIGNER
      </span>
    </a>
  </div>
  
  <!-- Terminal status bar -->
  <div class="px-4 py-2 text-[#666600] text-xs border-b border-[#1A1A1A]">
    > SYSTEM READY_
  </div>
  
  <!-- Nav items - terminal style -->
  <nav class="p-4 space-y-1">
    <a href="/dashboard" class="block px-3 py-2 text-[#33FF00] uppercase text-sm
                                hover:bg-[#0D1A0D] hover:text-shadow-[0_0_8px_#33FF00]">
      > DASHBOARD
    </a>
    <a href="/keys" class="block px-3 py-2 text-[#33FF00] uppercase text-sm
                          hover:bg-[#0D1A0D] hover:text-shadow-[0_0_8px_#33FF00]">
      > KEYS
    </a>
    <a href="/audit" class="block px-3 py-2 text-[#33FF00] uppercase text-sm
                           hover:bg-[#0D1A0D] hover:text-shadow-[0_0_8px_#33FF00]">
      > AUDIT_LOG
    </a>
    <a href="/settings" class="block px-3 py-2 text-[#228B22] uppercase text-sm
                              hover:bg-[#0D1A0D] hover:text-[#33FF00]">
      > SETTINGS
    </a>
  </nav>
</aside>
```

### pages/login.templ

```go
// Before
<h1>Sign in to BanhBaoRing</h1>
<p>Secure key management for your rollup</p>

// After - 80s CRT Login
<div class="bg-black border border-[#333300] p-8 font-mono">
  <h1 class="text-xl text-[#FFB000] mb-2 uppercase text-shadow-[0_0_10px_#FFB000]">
    > LOGIN_
  </h1>
  <p class="text-[#666600] text-sm mb-6">POINT-OF-PRESENCE SIGNING INFRASTRUCTURE</p>
  
  <form class="space-y-4">
    <div>
      <label class="text-sm text-[#228B22] uppercase">EMAIL:</label>
      <input type="email" 
             class="w-full bg-black border border-[#1A4D1A] text-[#33FF00] p-2 mt-1
                    focus:border-[#33FF00] focus:shadow-[0_0_10px_rgba(51,255,0,0.3)]
                    placeholder-[#336633]"
             placeholder="user@example.com">
    </div>
    <div>
      <label class="text-sm text-[#228B22] uppercase">PASSWORD:</label>
      <input type="password" 
             class="w-full bg-black border border-[#1A4D1A] text-[#33FF00] p-2 mt-1
                    focus:border-[#33FF00] focus:shadow-[0_0_10px_rgba(51,255,0,0.3)]">
    </div>
    <button type="submit" 
            class="w-full bg-[#FFB000] text-black font-bold py-3 uppercase
                   hover:bg-[#FFCC00] hover:shadow-[0_0_20px_#FFB000]">
      [ AUTHENTICATE ]
    </button>
  </form>
</div>
```

### pages/signup.templ

```go
// Before
<h1>Create your BanhBaoRing account</h1>
<p>Get started with secure key management</p>

// After - 80s CRT Signup
<div class="bg-black border border-[#333300] p-8 font-mono">
  <h1 class="text-xl text-[#FFB000] mb-2 uppercase text-shadow-[0_0_10px_#FFB000]">
    > DEPLOY_
  </h1>
  <p class="text-[#666600] text-sm mb-6">CREATE YOUR POPSIGNER ACCOUNT</p>
  
  <form class="space-y-4">
    <div>
      <label class="text-sm text-[#228B22] uppercase">EMAIL:</label>
      <input type="email" 
             class="w-full bg-black border border-[#1A4D1A] text-[#33FF00] p-2 mt-1
                    focus:border-[#33FF00] placeholder-[#336633]">
    </div>
    <div>
      <label class="font-mono text-sm text-neutral-400">password</label>
      <input type="password" class="w-full bg-black border border-neutral-700 text-white p-2 mt-1 font-mono focus:border-amber-600">
    </div>
    <button type="submit" class="w-full bg-amber-600 text-black font-semibold py-2 hover:bg-amber-500">
      Create Account →
    </button>
  </form>
</div>
```

### pages/onboarding.templ

```go
// Before
<h1>Welcome to BanhBaoRing!</h1>
<p>Let's set up your first key</p>

// After
<h1>Welcome to POPSigner</h1>
<p>Let's configure your signing infrastructure</p>
```

### pages/keys_list.templ

Terminal-style keys table:

```go
// Terminal aesthetic keys list
<div class="bg-black min-h-screen">
  <header class="border-b border-neutral-800 p-6">
    <h1 class="font-mono text-xl text-white">
      <span class="text-amber-500">_</span>keys
    </h1>
  </header>
  
  // Data table - trading terminal style
  <table class="w-full">
    <thead class="border-b border-neutral-800">
      <tr class="text-left font-mono text-xs text-neutral-500 uppercase">
        <th class="px-6 py-3">name</th>
        <th class="px-6 py-3">address</th>
        <th class="px-6 py-3">algorithm</th>
        <th class="px-6 py-3">created</th>
        <th class="px-6 py-3">exportable</th>
        <th class="px-6 py-3">actions</th>
      </tr>
    </thead>
    <tbody>
      for _, key := range keys {
        <tr class="border-b border-neutral-900 hover:bg-neutral-950">
          <td class="px-6 py-4 font-mono text-white">{ key.Name }</td>
          <td class="px-6 py-4 font-mono text-neutral-400 text-sm">{ key.Address }</td>
          <td class="px-6 py-4 font-mono text-cyan-400 text-sm">{ key.Algorithm }</td>
          <td class="px-6 py-4 font-mono text-neutral-500 text-sm">{ key.CreatedAt }</td>
          <td class="px-6 py-4">
            if key.Exportable {
              <span class="font-mono text-green-500 text-sm">EXIT_OK</span>
            } else {
              <span class="font-mono text-neutral-600 text-sm">LOCKED</span>
            }
          </td>
          <td class="px-6 py-4">
            <a href={ "/keys/" + key.ID } class="text-amber-500 hover:text-amber-400 text-sm">
              view →
            </a>
          </td>
        </tr>
      }
    </tbody>
  </table>
</div>
```

### pages/keys_detail.templ

Terminal-style key detail:

```go
// Terminal aesthetic key detail
<div class="bg-black min-h-screen p-6">
  <header class="mb-8">
    <h1 class="font-mono text-xl text-white">
      <span class="text-amber-500">_</span>key<span class="text-neutral-600">/</span>{ key.Name }
    </h1>
  </header>
  
  // Key info card
  <div class="bg-neutral-950 border border-neutral-800 p-6 mb-6">
    <div class="grid grid-cols-2 gap-4 font-mono text-sm">
      <div>
        <span class="text-neutral-500">address</span>
        <div class="text-white mt-1">{ key.Address }</div>
      </div>
      <div>
        <span class="text-neutral-500">algorithm</span>
        <div class="text-cyan-400 mt-1">{ key.Algorithm }</div>
      </div>
      <div>
        <span class="text-neutral-500">created</span>
        <div class="text-neutral-400 mt-1">{ key.CreatedAt }</div>
      </div>
      <div>
        <span class="text-neutral-500">exit_status</span>
        <div class="mt-1">
          if key.Exportable {
            <span class="text-green-500">EXIT_GUARANTEED</span>
          } else {
            <span class="text-red-500">LOCKED</span>
          }
        </div>
      </div>
    </div>
  </div>
  
  // Exit guarantee section (only if exportable)
  if key.Exportable {
    <div class="bg-neutral-950 border border-amber-900 p-6">
      <h3 class="font-mono text-amber-500 mb-2">exit_guarantee</h3>
      <p class="text-neutral-400 text-sm mb-4">
        Export this key for use in local keyrings. 
        Your sovereignty is non-negotiable.
      </p>
      <button hx-post={ "/keys/" + key.ID + "/export" }
              class="bg-amber-600 text-black font-mono text-sm px-4 py-2 hover:bg-amber-500">
        export_key →
      </button>
    </div>
  }
</div>
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

# Run locally
go run ./cmd/server

# Test pages:
# - /login
# - /signup
# - /dashboard
# - /keys
# - /keys/{id}

# Check for remaining references
grep -r "banhbao" ./templates/ --include="*.templ"
grep -r "BanhBao" ./templates/ --include="*.templ"
grep -r "🔔" ./templates/ --include="*.templ"
```

---

## Checklist

```
□ layouts/base.templ - title, meta tags
□ layouts/dashboard.templ - any branding
□ layouts/auth.templ - logo
□ layouts/landing.templ - title
□ components/sidebar.templ - logo
□ pages/dashboard.templ - branding
□ pages/login.templ - branding, copy
□ pages/signup.templ - branding, copy
□ pages/onboarding.templ - branding, copy
□ pages/forgot_password.templ - branding
□ pages/keys_list.templ - export visibility
□ pages/keys_detail.templ - export action
□ templ generate passes
□ go build passes
□ No remaining "banhbao", "BanhBao", or 🔔 references
```

---

## Output

After completion, the dashboard and auth pages reflect POPSigner branding with exit guarantee visibility.

