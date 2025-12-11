# Implementation: Web Dashboard (HTMX + templ)

## Agent: 11 - Dashboard

> **Depends on:** Agent 07 (Control Plane Foundation)  
> **Design Reference:** [`doc/design/DESIGN_SYSTEM.md`](../design/DESIGN_SYSTEM.md)

---

## 1. Overview

This agent implements the web dashboard using HTMX, templ, and Tailwind CSS. The dashboard is served from the same Go binary as the Control Plane API.

---

## 2. Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| **Templates** | templ | Latest |
| **Styling** | Tailwind CSS | 3.x |
| **Components** | DaisyUI | 4.x |
| **Interactivity** | HTMX | 2.0 |
| **Reactivity** | Alpine.js | 3.x |
| **Charts** | Chart.js | 4.x |
| **Syntax** | Highlight.js | 11.x |
| **OAuth** | markbates/goth | Latest |
| **Sessions** | gorilla/sessions | Latest |

---

## 3. Project Structure Addition

```
control-plane/
├── cmd/server/main.go              # Add web routes
├── internal/
│   └── handler/
│       └── web/                    # NEW: HTML handlers
│           ├── landing_handler.go
│           ├── auth_handler.go
│           ├── dashboard_handler.go
│           ├── keys_handler.go
│           ├── settings_handler.go
│           └── helpers.go
├── templates/                      # NEW: templ files
│   ├── layouts/
│   │   ├── base.templ
│   │   ├── landing.templ
│   │   ├── auth.templ
│   │   └── dashboard.templ
│   ├── pages/
│   │   ├── landing/
│   │   │   ├── index.templ
│   │   │   ├── features.templ
│   │   │   └── pricing.templ
│   │   ├── auth/
│   │   │   ├── login.templ
│   │   │   └── signup.templ
│   │   └── dashboard/
│   │       ├── overview.templ
│   │       ├── keys_list.templ
│   │       ├── key_detail.templ
│   │       └── settings.templ
│   ├── partials/
│   │   ├── keys_table.templ
│   │   ├── activity_feed.templ
│   │   └── toast.templ
│   └── components/
│       ├── button.templ
│       ├── card.templ
│       ├── modal.templ
│       ├── nav.templ
│       └── sidebar.templ
├── static/                         # NEW: Static assets
│   ├── css/
│   │   ├── input.css
│   │   └── output.css
│   ├── js/
│   │   └── app.js
│   └── img/
│       └── logo.svg
├── tailwind.config.js              # NEW
└── Makefile                        # Add templ/css targets
```

---

## 4. Sub-Agent Breakdown

### 4.1 Agent 11A: Foundation & Landing Page

**Files:**
- `templates/layouts/base.templ`
- `templates/layouts/landing.templ`
- `templates/pages/landing/index.templ`
- `templates/pages/landing/pricing.templ`
- `templates/components/nav.templ`
- `templates/components/button.templ`
- `internal/handler/web/landing_handler.go`
- `static/css/input.css`
- `tailwind.config.js`

**Deliverables:**
- [ ] templ setup with `templ generate`
- [ ] Tailwind configuration with custom colors
- [ ] Base layout with CDN scripts (HTMX, Alpine, Chart.js)
- [ ] Landing page with all sections (Hero, Problem, Solution, Features, Pricing, CTA)
- [ ] Responsive navigation
- [ ] Footer component

**Key Code:**

```go
// templates/layouts/base.templ
package layouts

templ Base(title string, showNav bool) {
    <!DOCTYPE html>
    <html lang="en" class="dark">
    <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>{ title } | BanhBaoRing</title>
        <link rel="stylesheet" href="/static/css/output.css"/>
        <link rel="preconnect" href="https://fonts.googleapis.com"/>
        <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500&family=Outfit:wght@400;500;600;700&family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet"/>
    </head>
    <body class="bg-[#0c0a14] text-white font-body antialiased">
        if showNav {
            @Nav()
        }
        { children... }
        <script src="https://unpkg.com/htmx.org@2.0.4"></script>
        <script defer src="https://unpkg.com/alpinejs@3.14.8/dist/cdn.min.js"></script>
        <script src="/static/js/app.js"></script>
    </body>
    </html>
}
```

```go
// internal/handler/web/landing_handler.go
package web

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/Bidon15/banhbaoring/control-plane/templates/pages/landing"
)

type LandingHandler struct{}

func NewLandingHandler() *LandingHandler {
    return &LandingHandler{}
}

func (h *LandingHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/", h.Index)
    r.Get("/pricing", h.Pricing)
    r.Get("/features", h.Features)
    return r
}

func (h *LandingHandler) Index(w http.ResponseWriter, r *http.Request) {
    landing.Index().Render(r.Context(), w)
}
```

---

### 4.2 Agent 11B: Auth Pages

**Files:**
- `templates/layouts/auth.templ`
- `templates/pages/auth/login.templ`
- `templates/pages/auth/signup.templ`
- `templates/pages/auth/onboarding.templ`
- `internal/handler/web/auth_handler.go`

**Deliverables:**
- [ ] Auth layout (centered card)
- [ ] Login page with OAuth buttons (GitHub, Google)
- [ ] Signup page
- [ ] Onboarding wizard (name org → create first key)
- [ ] OAuth callback handling
- [ ] Session management with cookies

**Key Code:**

```go
// templates/pages/auth/login.templ
package auth

import "github.com/Bidon15/banhbaoring/control-plane/templates/layouts"
import "github.com/Bidon15/banhbaoring/control-plane/templates/components"

templ Login(csrfToken string, errorMsg string) {
    @layouts.Auth("Log in") {
        <div class="text-center mb-8">
            <h1 class="text-3xl font-bold mb-2">Welcome back</h1>
            <p class="text-gray-400">Sign in to your BanhBaoRing account</p>
        </div>
        
        if errorMsg != "" {
            <div class="bg-red-500/10 border border-red-500/50 text-red-400 px-4 py-3 rounded-lg mb-6">
                { errorMsg }
            </div>
        }
        
        <!-- OAuth Buttons -->
        <div class="space-y-3 mb-6">
            <a href="/auth/github" class="flex items-center justify-center gap-3 w-full py-3 px-4 bg-[#24292e] hover:bg-[#2f363d] text-white rounded-lg transition-colors">
                <svg class="w-5 h-5">...</svg>
                Continue with GitHub
            </a>
            <a href="/auth/google" class="flex items-center justify-center gap-3 w-full py-3 px-4 bg-white hover:bg-gray-100 text-gray-900 rounded-lg transition-colors">
                <svg class="w-5 h-5">...</svg>
                Continue with Google
            </a>
        </div>
        
        <div class="relative mb-6">
            <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-[#4a3f5c]"></div>
            </div>
            <div class="relative flex justify-center text-sm">
                <span class="px-2 bg-[#1a1625] text-gray-400">or continue with email</span>
            </div>
        </div>
        
        <!-- Email Form -->
        <form hx-post="/auth/login" hx-target="#auth-result">
            <input type="hidden" name="csrf_token" value={ csrfToken }/>
            @components.Input("email", "Email", "you@company.com", "email", true)
            @components.Input("password", "Password", "••••••••", "password", true)
            <button type="submit" class="w-full mt-4 py-3 bg-gradient-to-r from-purple-500 to-orange-500 text-white font-semibold rounded-lg hover:shadow-lg hover:shadow-purple-500/25 transition-all">
                Sign in
            </button>
        </form>
        
        <p class="text-center text-gray-400 mt-6">
            Don't have an account? 
            <a href="/signup" class="text-purple-400 hover:text-purple-300">Sign up free</a>
        </p>
        
        <div id="auth-result"></div>
    }
}
```

---

### 4.3 Agent 11C: Dashboard Overview & Keys

**Files:**
- `templates/layouts/dashboard.templ`
- `templates/pages/dashboard/overview.templ`
- `templates/pages/dashboard/keys_list.templ`
- `templates/pages/dashboard/key_detail.templ`
- `templates/pages/dashboard/keys_new.templ`
- `templates/partials/keys_table.templ`
- `templates/partials/activity_feed.templ`
- `templates/components/sidebar.templ`
- `templates/components/card.templ`
- `templates/components/key_card.templ`
- `internal/handler/web/dashboard_handler.go`
- `internal/handler/web/keys_handler.go`

**Deliverables:**
- [ ] Dashboard layout with sidebar
- [ ] Overview page with stats cards
- [ ] Recent activity feed (HTMX polling)
- [ ] Keys list with search/filter (HTMX)
- [ ] Key detail page with signature chart
- [ ] Create key modal
- [ ] Delete key confirmation
- [ ] Sign test functionality

**Key Code:**

```go
// templates/layouts/dashboard.templ
package layouts

templ Dashboard(title string, user *models.User, org *models.Organization) {
    @Base(title, false) {
        <div class="flex h-screen">
            <!-- Sidebar -->
            @Sidebar(org)
            
            <!-- Main content -->
            <main class="flex-1 overflow-y-auto">
                <!-- Top bar -->
                <header class="sticky top-0 z-10 bg-[#0c0a14]/80 backdrop-blur-lg border-b border-[#4a3f5c]/50 px-6 py-4">
                    <div class="flex items-center justify-between">
                        <h1 class="text-xl font-semibold">{ title }</h1>
                        @UserMenu(user)
                    </div>
                </header>
                
                <!-- Page content -->
                <div class="p-6">
                    { children... }
                </div>
            </main>
        </div>
        
        <!-- Modal container -->
        <div id="modal"></div>
        
        <!-- Toast container -->
        @Toast()
    }
}
```

```go
// templates/pages/dashboard/keys_list.templ
package dashboard

templ KeysList(keys []*models.Key, namespaces []*models.Namespace, search string) {
    @layouts.Dashboard("Keys", user, org) {
        <!-- Header -->
        <div class="flex justify-between items-center mb-6">
            <div>
                <h2 class="text-2xl font-bold">Keys</h2>
                <p class="text-gray-400">Manage your signing keys</p>
            </div>
            <button 
                hx-get="/keys/new"
                hx-target="#modal"
                class="bg-gradient-to-r from-purple-500 to-orange-500 text-white font-semibold px-4 py-2 rounded-lg"
            >
                + Create Key
            </button>
        </div>
        
        <!-- Filters -->
        <div class="flex gap-4 mb-6">
            <input 
                type="search" 
                name="q" 
                placeholder="Search keys..."
                value={ search }
                hx-get="/keys"
                hx-trigger="keyup changed delay:300ms, search"
                hx-target="#keys-list"
                hx-push-url="true"
                class="flex-1 px-4 py-2 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg text-white placeholder:text-gray-500 focus:border-purple-500"
            />
            <select 
                name="namespace"
                hx-get="/keys"
                hx-trigger="change"
                hx-target="#keys-list"
                hx-include="[name='q']"
                class="px-4 py-2 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg text-white"
            >
                <option value="">All Namespaces</option>
                for _, ns := range namespaces {
                    <option value={ ns.ID.String() }>{ ns.Name }</option>
                }
            </select>
        </div>
        
        <!-- Keys List -->
        <div id="keys-list">
            @KeysTable(keys)
        </div>
    }
}
```

---

### 4.4 Agent 11D: Settings & Billing

**Files:**
- `templates/pages/dashboard/settings/profile.templ`
- `templates/pages/dashboard/settings/team.templ`
- `templates/pages/dashboard/settings/api_keys.templ`
- `templates/pages/dashboard/settings/billing.templ`
- `templates/pages/dashboard/audit.templ`
- `templates/pages/dashboard/usage.templ`
- `internal/handler/web/settings_handler.go`

**Deliverables:**
- [ ] Profile settings page
- [ ] Team management (invite, remove, roles)
- [ ] API keys management
- [ ] Billing page with plan display
- [ ] Audit log with filters
- [ ] Usage charts (Chart.js)

---

## 5. Route Registration

**File:** `cmd/server/main.go`

```go
// Web routes (HTML)
r.Route("/", func(r chi.Router) {
    // Public pages
    r.Mount("/", webHandlers.Landing.Routes())
    
    // Auth pages
    r.Mount("/auth", webHandlers.Auth.Routes())
    r.Get("/login", webHandlers.Auth.Login)
    r.Get("/signup", webHandlers.Auth.Signup)
    
    // Static files
    r.Handle("/static/*", http.StripPrefix("/static/", 
        http.FileServer(http.Dir("static"))))
    
    // Protected pages
    r.Group(func(r chi.Router) {
        r.Use(middleware.WebAuth) // Cookie-based auth
        
        r.Get("/dashboard", webHandlers.Dashboard.Overview)
        r.Mount("/keys", webHandlers.Keys.Routes())
        r.Mount("/audit", webHandlers.Audit.Routes())
        r.Mount("/settings", webHandlers.Settings.Routes())
    })
})

// API routes (JSON) - existing
r.Route("/v1", func(r chi.Router) {
    // ... existing API routes
})
```

---

## 6. Makefile Additions

```makefile
# templ generation
.PHONY: templ templ-watch

templ:
	templ generate

templ-watch:
	templ generate --watch

# Tailwind CSS
.PHONY: css css-watch

css:
	./tailwindcss -i static/css/input.css -o static/css/output.css --minify

css-watch:
	./tailwindcss -i static/css/input.css -o static/css/output.css --watch

# Development (run all watchers)
.PHONY: dev

dev:
	@make -j3 templ-watch css-watch air
```

---

## 7. tailwind.config.js

```javascript
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.templ",
    "./static/js/**/*.js",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#fdf4ff',
          100: '#fae8ff',
          200: '#f5d0fe',
          300: '#f0abfc',
          400: '#e879f9',
          500: '#d946ef',
          600: '#a855f7',
          700: '#7e22ce',
          800: '#6b21a8',
          900: '#581c87',
        },
        accent: {
          400: '#fb923c',
          500: '#f97316',
          600: '#ea580c',
        },
      },
      fontFamily: {
        display: ['Outfit', 'sans-serif'],
        body: ['Plus Jakarta Sans', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      backgroundColor: {
        'bg-primary': '#0c0a14',
        'bg-secondary': '#1a1625',
        'bg-tertiary': '#2d2640',
      },
    },
  },
  plugins: [
    require('daisyui'),
  ],
  daisyui: {
    themes: ['dark'],
  },
}
```

---

## 8. Success Criteria

### 8.1 Functional

- [ ] Landing page renders with all sections
- [ ] OAuth login/signup works (GitHub, Google)
- [ ] Dashboard shows user's keys
- [ ] HTMX partial updates work (search, filter)
- [ ] Create key modal works
- [ ] Sign test returns signature
- [ ] Audit log displays with pagination
- [ ] Settings pages save correctly

### 8.2 Performance

- [ ] LCP < 1.5s
- [ ] Total JS < 50KB gzipped
- [ ] Lighthouse score > 90

### 8.3 Accessibility

- [ ] Keyboard navigation works
- [ ] Focus indicators visible
- [ ] Color contrast ≥ 4.5:1

---

## 9. Timeline

| Phase | Deliverables | Duration |
|-------|--------------|----------|
| **11A** | Foundation, Landing Page | 3 days |
| **11B** | Auth Pages, OAuth | 2 days |
| **11C** | Dashboard, Keys | 4 days |
| **11D** | Settings, Billing, Audit | 3 days |
| **Polish** | Mobile, Animations, Testing | 2 days |

**Total: ~14 days (2 weeks)**

---

## 10. Dependencies

- Agent 07 (CP Foundation) - Database, services
- Agent 08A-C (Auth) - Auth services
- Agent 09A-B (Orgs, Keys) - Key management API

---

## 11. Agent Prompt

```
You are Agent 11 - Dashboard. Your task is to implement the web dashboard using HTMX, templ, and Tailwind CSS.

Read the specs:
- doc/implementation/IMPL_11_DASHBOARD.md (this file)
- doc/design/DESIGN_SYSTEM.md (visual design)
- doc/product/PRD_DASHBOARD.md (requirements)

Start with Phase 11A (Foundation + Landing Page):
1. Install templ: go install github.com/a-h/templ/cmd/templ@latest
2. Download Tailwind standalone binary
3. Create templates/ directory structure
4. Implement base layout with CDN scripts
5. Create landing page with Hero, Problem, Solution sections
6. Add responsive navigation
7. Set up static file serving

Tech stack: templ, Tailwind CSS 3, HTMX 2.0, Alpine.js 3, Chart.js 4

Test: templ generate && make css && go run ./cmd/server
Visit: http://localhost:8080
```

