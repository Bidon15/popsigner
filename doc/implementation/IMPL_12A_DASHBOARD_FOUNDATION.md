# Implementation: Dashboard Foundation

## Agent: 12A - Dashboard Foundation (BLOCKING)

> **Phase 8.0** - Must complete before other dashboard agents can start.

---

## 1. Overview

Set up the web dashboard project structure using Go + templ + HTMX + Tailwind CSS.

> **Tech Stack:** Go templates (templ), HTMX for interactivity, Alpine.js for reactivity, Tailwind CSS for styling. NO React, NO Node.js.

---

## 2. Tech Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| Language | Go 1.22+ | Same as control-plane |
| Templates | templ | Type-safe Go templates |
| Interactivity | HTMX 2.0 | HTML-over-the-wire |
| Reactivity | Alpine.js 3.x | Dropdowns, modals |
| Styling | Tailwind CSS | Utility-first |
| Router | Chi | Already in use |
| Sessions | gorilla/sessions | Redis-backed |

---

## 3. Project Structure

Add to existing `control-plane/` directory:

```
control-plane/
├── internal/
│   └── handler/
│       └── web/                    # NEW: Web handlers
│           ├── routes.go           # Web routes setup
│           ├── auth.go             # Login/signup/OAuth
│           ├── dashboard.go        # Dashboard page
│           ├── keys.go             # Keys pages
│           ├── billing.go          # Billing pages
│           ├── settings.go         # Settings pages
│           └── audit.go            # Audit pages
├── templates/                      # NEW: templ files
│   ├── layouts/
│   │   ├── base.templ              # HTML head, scripts
│   │   ├── auth.templ              # Auth layout (no sidebar)
│   │   └── dashboard.templ         # Dashboard layout (sidebar)
│   ├── pages/
│   │   ├── login.templ
│   │   ├── signup.templ
│   │   └── ... (other pages)
│   ├── partials/                   # HTMX partial responses
│   │   └── ...
│   └── components/
│       ├── button.templ
│       ├── card.templ
│       ├── sidebar.templ
│       ├── nav.templ
│       ├── modal.templ
│       ├── toast.templ
│       └── code_block.templ
├── static/
│   ├── css/
│   │   ├── input.css               # Tailwind input
│   │   └── output.css              # Compiled CSS
│   ├── js/
│   │   └── app.js                  # Alpine init, utils
│   └── img/
│       └── logo.svg
├── tailwind.config.js
└── Makefile                        # Updated with templ/css commands
```

---

## 4. Dependencies

**Update `go.mod`:**

```go
require (
    github.com/a-h/templ v0.2.793
    github.com/gorilla/sessions v1.2.2
    // ... existing deps
)
```

**Install templ CLI:**
```bash
go install github.com/a-h/templ/cmd/templ@latest
```

---

## 5. Base Layout

**File:** `templates/layouts/base.templ`

```go
package layouts

templ Base(title string, showCharts, showCode bool) {
    <!DOCTYPE html>
    <html lang="en" class="dark">
    <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>{ title } | BanhBaoRing</title>
        
        <!-- Fonts -->
        <link rel="preconnect" href="https://fonts.bunny.net"/>
        <link href="https://fonts.bunny.net/css?family=outfit:400,500,600,700|jetbrains-mono:400" rel="stylesheet"/>
        
        <!-- Tailwind CSS -->
        <link rel="stylesheet" href="/static/css/output.css"/>
        
        <!-- Favicon -->
        <link rel="icon" href="/static/img/favicon.svg" type="image/svg+xml"/>
    </head>
    <body class="bg-[#0c0a14] text-[#faf5ff] font-sans antialiased">
        { children... }
        
        <!-- HTMX -->
        <script src="https://unpkg.com/htmx.org@2.0.4" 
                integrity="sha384-HGfztofotfshcF7+8n44JQL2oJmowVChPTg48S+jvZoztPfvwD79OC/LTtG6dMp+" 
                crossorigin="anonymous"></script>
        
        <!-- Alpine.js -->
        <script defer src="https://unpkg.com/alpinejs@3.14.8/dist/cdn.min.js"></script>
        
        if showCharts {
            <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
        }
        
        if showCode {
            <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/styles/github-dark.min.css"/>
            <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/lib/core.min.js"></script>
            <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/lib/languages/go.min.js"></script>
            <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/lib/languages/rust.min.js"></script>
            <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/lib/languages/bash.min.js"></script>
            <script>hljs.highlightAll();</script>
        }
        
        <!-- App JS -->
        <script src="/static/js/app.js"></script>
    </body>
    </html>
}
```

---

## 6. Dashboard Layout

**File:** `templates/layouts/dashboard.templ`

```go
package layouts

import "github.com/Bidon15/banhbaoring/control-plane/templates/components"

templ Dashboard(title string, user *models.User, org *models.Organization) {
    @Base(title, false, false) {
        <div class="min-h-screen flex" x-data="{ sidebarOpen: false }">
            <!-- Desktop Sidebar -->
            <aside class="hidden lg:flex lg:w-64 lg:flex-col lg:fixed lg:inset-y-0">
                @components.Sidebar(user, org)
            </aside>
            
            <!-- Mobile Sidebar (slide-out) -->
            <div x-show="sidebarOpen" 
                 x-transition:enter="transition ease-out duration-200"
                 x-transition:enter-start="opacity-0"
                 x-transition:enter-end="opacity-100"
                 x-transition:leave="transition ease-in duration-150"
                 x-transition:leave-start="opacity-100"
                 x-transition:leave-end="opacity-0"
                 class="lg:hidden fixed inset-0 z-40">
                <div class="absolute inset-0 bg-black/50" @click="sidebarOpen = false"></div>
                <aside class="absolute left-0 top-0 h-full w-64 bg-[#0c0a14] border-r border-[#4a3f5c]"
                       x-transition:enter="transition ease-out duration-200"
                       x-transition:enter-start="-translate-x-full"
                       x-transition:enter-end="translate-x-0"
                       x-transition:leave="transition ease-in duration-150"
                       x-transition:leave-start="translate-x-0"
                       x-transition:leave-end="-translate-x-full">
                    @components.Sidebar(user, org)
                </aside>
            </div>
            
            <!-- Main Content -->
            <div class="lg:pl-64 flex flex-col flex-1">
                <!-- Top Nav (mobile) -->
                <header class="lg:hidden flex items-center justify-between p-4 border-b border-[#4a3f5c]">
                    <button @click="sidebarOpen = true" class="text-gray-400 hover:text-white">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
                        </svg>
                    </button>
                    <span class="text-xl font-bold">🔔 BanhBaoRing</span>
                    <div class="w-6"></div>
                </header>
                
                <!-- Page Content -->
                <main id="main-content" class="flex-1 p-6 lg:p-8">
                    { children... }
                </main>
            </div>
            
            <!-- Toast Container -->
            <div id="toast-container" class="fixed bottom-4 right-4 z-50"></div>
            
            <!-- Modal Container -->
            <div id="modal" 
                 class="fixed inset-0 z-50 hidden items-center justify-center"
                 x-data="{ open: false }"
                 x-show="open"
                 @modal-open.window="open = true; $el.classList.remove('hidden'); $el.classList.add('flex')"
                 @modal-close.window="open = false; setTimeout(() => $el.classList.add('hidden'), 150)"
                 @keydown.escape.window="$dispatch('modal-close')">
                <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="$dispatch('modal-close')"></div>
                <div id="modal-content" 
                     class="relative bg-[#1a1625] border border-[#4a3f5c] rounded-xl max-w-lg w-full mx-4 p-6 shadow-2xl"
                     x-transition:enter="transition ease-out duration-200"
                     x-transition:enter-start="opacity-0 scale-95"
                     x-transition:enter-end="opacity-100 scale-100"
                     x-transition:leave="transition ease-in duration-150"
                     x-transition:leave-start="opacity-100 scale-100"
                     x-transition:leave-end="opacity-0 scale-95">
                </div>
            </div>
        </div>
    }
}
```

---

## 7. Components

**File:** `templates/components/sidebar.templ`

```go
package components

import "github.com/Bidon15/banhbaoring/control-plane/internal/models"

templ Sidebar(user *models.User, org *models.Organization) {
    <div class="flex flex-col h-full bg-[#0c0a14] border-r border-[#4a3f5c]">
        <!-- Logo -->
        <div class="flex items-center h-16 px-6 border-b border-[#4a3f5c]">
            <span class="text-2xl">🔔</span>
            <span class="ml-2 text-xl font-bold bg-gradient-to-r from-purple-400 to-orange-400 bg-clip-text text-transparent">
                BanhBaoRing
            </span>
        </div>
        
        <!-- Org Selector -->
        <div class="px-4 py-3 border-b border-[#4a3f5c]">
            <div class="flex items-center gap-2 px-3 py-2 bg-[#1a1625] rounded-lg">
                <span class="text-sm font-medium text-white truncate">{ org.Name }</span>
                <span class="ml-auto text-xs text-purple-400 bg-purple-500/10 px-2 py-0.5 rounded">
                    { org.Plan }
                </span>
            </div>
        </div>
        
        <!-- Navigation -->
        <nav class="flex-1 px-4 py-4 space-y-1 overflow-y-auto">
            @SidebarLink("/dashboard", "🏠", "Overview", true)
            @SidebarLink("/keys", "🔑", "Keys", false)
            @SidebarLink("/usage", "📊", "Usage", false)
            @SidebarLink("/audit", "📜", "Audit Log", false)
            
            <div class="pt-4 mt-4 border-t border-[#4a3f5c]">
                <p class="px-3 mb-2 text-xs font-semibold text-gray-500 uppercase">Settings</p>
                @SidebarLink("/settings/team", "👥", "Team", false)
                @SidebarLink("/settings/api-keys", "🔗", "API Keys", false)
                @SidebarLink("/settings/billing", "💳", "Billing", false)
            </div>
        </nav>
        
        <!-- User Menu -->
        <div class="p-4 border-t border-[#4a3f5c]">
            <div class="flex items-center gap-3">
                if user.AvatarURL != "" {
                    <img src={ user.AvatarURL } alt="Avatar" class="w-8 h-8 rounded-full"/>
                } else {
                    <div class="w-8 h-8 rounded-full bg-purple-500 flex items-center justify-center text-white text-sm font-bold">
                        { string(user.Name[0]) }
                    </div>
                }
                <div class="flex-1 min-w-0">
                    <p class="text-sm font-medium text-white truncate">{ user.Name }</p>
                    <p class="text-xs text-gray-400 truncate">{ user.Email }</p>
                </div>
                <form action="/logout" method="POST">
                    <button type="submit" class="text-gray-400 hover:text-white" title="Logout">
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                                  d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                        </svg>
                    </button>
                </form>
            </div>
        </div>
    </div>
}

templ SidebarLink(href, icon, label string, active bool) {
    <a href={ templ.SafeURL(href) }
       hx-get={ href }
       hx-target="#main-content"
       hx-push-url="true"
       class={ "flex items-center gap-3 px-3 py-2 rounded-lg transition-colors",
               templ.KV("bg-purple-500/10 text-purple-300", active),
               templ.KV("text-gray-400 hover:text-white hover:bg-white/5", !active) }>
        <span>{ icon }</span>
        <span>{ label }</span>
    </a>
}
```

**File:** `templates/components/button.templ`

```go
package components

type ButtonVariant string

const (
    ButtonPrimary   ButtonVariant = "primary"
    ButtonSecondary ButtonVariant = "secondary"
    ButtonDanger    ButtonVariant = "danger"
    ButtonGhost     ButtonVariant = "ghost"
)

templ Button(variant ButtonVariant, disabled bool) {
    <button 
        disabled?={ disabled }
        class={ buttonClasses(variant, disabled) }>
        { children... }
    </button>
}

func buttonClasses(variant ButtonVariant, disabled bool) string {
    base := "inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg font-medium transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-[#0c0a14]"
    
    if disabled {
        return base + " opacity-50 cursor-not-allowed"
    }
    
    switch variant {
    case ButtonPrimary:
        return base + " bg-gradient-to-r from-purple-500 to-orange-500 text-white shadow-lg shadow-purple-500/25 hover:shadow-purple-500/40 focus:ring-purple-500"
    case ButtonSecondary:
        return base + " border border-purple-500/50 text-purple-300 hover:bg-purple-500/10 hover:border-purple-500 focus:ring-purple-500"
    case ButtonDanger:
        return base + " bg-red-500/10 text-red-400 border border-red-500/50 hover:bg-red-500/20 focus:ring-red-500"
    case ButtonGhost:
        return base + " text-gray-400 hover:text-white hover:bg-white/5 focus:ring-gray-500"
    default:
        return base
    }
}
```

**File:** `templates/components/card.templ`

```go
package components

templ Card(title string) {
    <div class="bg-[#1a1625]/80 backdrop-blur-lg border border-[#4a3f5c] rounded-xl p-6 
                hover:border-purple-500/30 hover:shadow-lg hover:shadow-purple-500/5 
                transition-all duration-300">
        if title != "" {
            <h3 class="text-lg font-semibold text-white mb-4">{ title }</h3>
        }
        { children... }
    </div>
}
```

**File:** `templates/components/toast.templ`

```go
package components

type ToastVariant string

const (
    ToastSuccess ToastVariant = "success"
    ToastError   ToastVariant = "error"
    ToastWarning ToastVariant = "warning"
    ToastInfo    ToastVariant = "info"
)

templ Toast(message string, variant ToastVariant) {
    <div id="toast" 
         hx-swap-oob="innerHTML:#toast-container"
         x-data="{ show: true }"
         x-show="show"
         x-init="setTimeout(() => { show = false; $el.remove() }, 5000)"
         x-transition:enter="transition ease-out duration-300"
         x-transition:enter-start="opacity-0 translate-y-2"
         x-transition:enter-end="opacity-100 translate-y-0"
         x-transition:leave="transition ease-in duration-200"
         x-transition:leave-start="opacity-100 translate-y-0"
         x-transition:leave-end="opacity-0 translate-y-2"
         class={ "px-4 py-3 rounded-lg shadow-lg flex items-center gap-3", toastClasses(variant) }>
        <span>{ toastIcon(variant) }</span>
        <p class="text-white font-medium">{ message }</p>
        <button @click="show = false; $el.parentElement.remove()" class="ml-auto text-white/60 hover:text-white">
            ✕
        </button>
    </div>
}

func toastClasses(variant ToastVariant) string {
    switch variant {
    case ToastSuccess:
        return "bg-emerald-500/90"
    case ToastError:
        return "bg-red-500/90"
    case ToastWarning:
        return "bg-amber-500/90"
    default:
        return "bg-purple-500/90"
    }
}

func toastIcon(variant ToastVariant) string {
    switch variant {
    case ToastSuccess:
        return "✓"
    case ToastError:
        return "✕"
    case ToastWarning:
        return "⚠"
    default:
        return "ℹ"
    }
}
```

---

## 8. Tailwind Configuration

**File:** `tailwind.config.js`

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
        // BanhBaoRing brand colors
        'bao': {
          bg: '#0c0a14',
          card: '#1a1625',
          border: '#4a3f5c',
        },
      },
      fontFamily: {
        sans: ['Outfit', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
  plugins: [],
}
```

**File:** `static/css/input.css`

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* Custom scrollbar */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #1a1625;
}

::-webkit-scrollbar-thumb {
  background: #4a3f5c;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #5a4f6c;
}

/* HTMX loading indicator */
.htmx-request .htmx-indicator {
  display: inline-block;
}

.htmx-indicator {
  display: none;
}
```

---

## 9. App JavaScript

**File:** `static/js/app.js`

```javascript
// Copy to clipboard utility
window.copyToClipboard = async (text) => {
  await navigator.clipboard.writeText(text);
};

// HTMX configuration
document.addEventListener('htmx:configRequest', (event) => {
  // Add CSRF token to all requests
  const csrfToken = document.querySelector('meta[name="csrf-token"]')?.content;
  if (csrfToken) {
    event.detail.headers['X-CSRF-Token'] = csrfToken;
  }
});

// Show toast on HTMX errors
document.addEventListener('htmx:responseError', (event) => {
  const container = document.getElementById('toast-container');
  if (container) {
    container.innerHTML = `
      <div class="px-4 py-3 rounded-lg shadow-lg bg-red-500/90 text-white">
        Something went wrong. Please try again.
      </div>
    `;
  }
});
```

---

## 10. Web Routes Setup

**File:** `internal/handler/web/routes.go`

```go
package web

import (
    "github.com/go-chi/chi/v5"
    "github.com/gorilla/sessions"
    
    "github.com/Bidon15/banhbaoring/control-plane/internal/service"
)

type WebHandler struct {
    authService    service.AuthService
    keyService     service.KeyService
    orgService     service.OrgService
    billingService service.BillingService
    auditService   service.AuditService
    sessionStore   sessions.Store
}

func NewWebHandler(
    authService service.AuthService,
    keyService service.KeyService,
    orgService service.OrgService,
    billingService service.BillingService,
    auditService service.AuditService,
    sessionStore sessions.Store,
) *WebHandler {
    return &WebHandler{
        authService:    authService,
        keyService:     keyService,
        orgService:     orgService,
        billingService: billingService,
        auditService:   auditService,
        sessionStore:   sessionStore,
    }
}

func (h *WebHandler) Routes() chi.Router {
    r := chi.NewRouter()
    
    // Static files
    r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    
    // Public routes
    r.Group(func(r chi.Router) {
        r.Get("/", h.LandingPage)
        r.Get("/login", h.LoginPage)
        r.Post("/login", h.Login)
        r.Get("/signup", h.SignupPage)
        r.Post("/signup", h.Signup)
        r.Get("/auth/{provider}", h.OAuthStart)
        r.Get("/auth/{provider}/callback", h.OAuthCallback)
    })
    
    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(h.RequireAuth)
        
        r.Get("/dashboard", h.Dashboard)
        r.Post("/logout", h.Logout)
        
        // Keys
        r.Get("/keys", h.KeysList)
        r.Get("/keys/new", h.KeysNew)
        r.Post("/keys", h.KeysCreate)
        r.Get("/keys/workers/new", h.WorkerKeysNew)
        r.Post("/keys/workers", h.WorkerKeysCreate)
        r.Get("/keys/{id}", h.KeysDetail)
        r.Post("/keys/{id}/sign-test", h.KeysSignTest)
        r.Delete("/keys/{id}", h.KeysDelete)
        
        // Usage
        r.Get("/usage", h.Usage)
        
        // Audit
        r.Get("/audit", h.AuditLog)
        
        // Settings
        r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
            http.Redirect(w, r, "/settings/profile", http.StatusFound)
        })
        r.Get("/settings/profile", h.SettingsProfile)
        r.Get("/settings/team", h.SettingsTeam)
        r.Get("/settings/api-keys", h.SettingsAPIKeys)
        r.Get("/settings/billing", h.SettingsBilling)
    })
    
    return r
}
```

---

## 11. Makefile Updates

```makefile
# Add to existing Makefile

.PHONY: templ templ-watch css css-watch dev-web

templ:
	templ generate

templ-watch:
	templ generate --watch

css:
	./tailwindcss -i static/css/input.css -o static/css/output.css --minify

css-watch:
	./tailwindcss -i static/css/input.css -o static/css/output.css --watch

# Run web development (templ + css + server)
dev-web:
	@make -j3 templ-watch css-watch run

# Production build
build-web:
	templ generate
	./tailwindcss -i static/css/input.css -o static/css/output.css --minify
	go build -o bin/server ./cmd/server
```

---

## 12. Deliverables

| File | Description |
|------|-------------|
| `templates/layouts/base.templ` | Base HTML layout |
| `templates/layouts/auth.templ` | Auth pages layout |
| `templates/layouts/dashboard.templ` | Dashboard layout |
| `templates/components/*.templ` | Reusable components |
| `static/css/input.css` | Tailwind source |
| `static/js/app.js` | Client-side utilities |
| `tailwind.config.js` | Tailwind configuration |
| `internal/handler/web/routes.go` | Web routes |
| `Makefile` | Updated build commands |

---

## 13. Success Criteria

- [ ] `templ generate` runs without errors
- [ ] Tailwind CSS builds successfully
- [ ] Base layout renders correctly
- [ ] Dashboard layout with sidebar works
- [ ] Components (Button, Card, Toast) work
- [ ] Static files served correctly
- [ ] HTMX and Alpine.js loaded
- [ ] Mobile responsive sidebar

---

## 14. Agent Prompt

```
You are Agent 12A - Dashboard Foundation. Set up the web dashboard project structure.

Read: doc/implementation/IMPL_12A_DASHBOARD_FOUNDATION.md

Tech Stack: Go + templ + HTMX + Alpine.js + Tailwind CSS (NO React, NO Node.js)

Deliverables:
1. Install templ, set up templates/ directory
2. Base layout with HTMX, Alpine.js, Tailwind
3. Dashboard layout with sidebar
4. Core components (Button, Card, Toast, Sidebar)
5. Tailwind configuration
6. Static files structure
7. Web routes setup
8. Makefile updates

This extends the existing control-plane/ directory.

Test: templ generate && ./tailwindcss -i static/css/input.css -o static/css/output.css && go build ./...
```

