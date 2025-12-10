# Implementation: Dashboard Auth Pages

## Agent: 12B - Auth Pages

> **Phase 8.1** - Can start after Agent 12A completes.

---

## 1. Overview

Implement authentication pages: login, signup, OAuth callbacks, and onboarding flow.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| Login page | ✅ |
| Signup page | ✅ |
| OAuth buttons (GitHub, Google, Discord) | ✅ |
| Email/password form | ✅ |
| Onboarding wizard | ✅ |
| Password reset | ✅ |
| Session management | ✅ |

---

## 3. Pages

### 3.1 Login Page

**File:** `templates/pages/login.templ`

```go
package pages

import (
    "github.com/Bidon15/banhbaoring/control-plane/templates/layouts"
    "github.com/Bidon15/banhbaoring/control-plane/templates/components"
)

templ LoginPage(error string) {
    @layouts.Auth("Login") {
        <div class="min-h-screen flex items-center justify-center p-4">
            <div class="w-full max-w-md">
                <!-- Logo -->
                <div class="text-center mb-8">
                    <span class="text-5xl">🔔</span>
                    <h1 class="mt-4 text-3xl font-bold bg-gradient-to-r from-purple-400 to-orange-400 bg-clip-text text-transparent">
                        BanhBaoRing
                    </h1>
                    <p class="mt-2 text-gray-400">Secure key management for Celestia</p>
                </div>
                
                @components.Card("") {
                    if error != "" {
                        <div class="mb-4 p-3 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400 text-sm">
                            { error }
                        </div>
                    }
                    
                    <!-- OAuth Buttons -->
                    <div class="space-y-3">
                        <a href="/auth/github" 
                           class="flex items-center justify-center gap-3 w-full px-4 py-3 bg-[#24292e] hover:bg-[#2f363d] rounded-lg transition-colors">
                            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
                            </svg>
                            <span class="font-medium">Continue with GitHub</span>
                        </a>
                        
                        <a href="/auth/google"
                           class="flex items-center justify-center gap-3 w-full px-4 py-3 bg-white text-gray-900 hover:bg-gray-100 rounded-lg transition-colors">
                            <svg class="w-5 h-5" viewBox="0 0 24 24">
                                <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                                <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                                <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                                <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
                            </svg>
                            <span class="font-medium">Continue with Google</span>
                        </a>
                        
                        <a href="/auth/discord"
                           class="flex items-center justify-center gap-3 w-full px-4 py-3 bg-[#5865F2] hover:bg-[#4752C4] rounded-lg transition-colors">
                            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/>
                            </svg>
                            <span class="font-medium">Continue with Discord</span>
                        </a>
                    </div>
                    
                    <!-- Divider -->
                    <div class="relative my-6">
                        <div class="absolute inset-0 flex items-center">
                            <div class="w-full border-t border-[#4a3f5c]"></div>
                        </div>
                        <div class="relative flex justify-center text-sm">
                            <span class="px-4 bg-[#1a1625] text-gray-400">or</span>
                        </div>
                    </div>
                    
                    <!-- Email Form -->
                    <form action="/login" method="POST" class="space-y-4">
                        <div>
                            <label for="email" class="block text-sm font-medium text-gray-300 mb-1">Email</label>
                            <input type="email" id="email" name="email" required
                                   class="w-full px-4 py-3 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg text-white placeholder-gray-500 focus:border-purple-500 focus:ring-1 focus:ring-purple-500 focus:outline-none"
                                   placeholder="you@example.com"/>
                        </div>
                        
                        <div>
                            <label for="password" class="block text-sm font-medium text-gray-300 mb-1">Password</label>
                            <input type="password" id="password" name="password" required
                                   class="w-full px-4 py-3 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg text-white placeholder-gray-500 focus:border-purple-500 focus:ring-1 focus:ring-purple-500 focus:outline-none"
                                   placeholder="••••••••"/>
                        </div>
                        
                        <div class="flex items-center justify-between text-sm">
                            <a href="/forgot-password" class="text-purple-400 hover:text-purple-300">Forgot password?</a>
                        </div>
                        
                        @components.Button(components.ButtonPrimary, false) {
                            <span class="w-full text-center">Sign In</span>
                        }
                    </form>
                    
                    <p class="mt-6 text-center text-sm text-gray-400">
                        Don't have an account?
                        <a href="/signup" class="text-purple-400 hover:text-purple-300 font-medium">Sign up free</a>
                    </p>
                }
            </div>
        </div>
    }
}
```

### 3.2 Onboarding Flow

**File:** `templates/pages/onboarding.templ`

```go
package pages

import (
    "github.com/Bidon15/banhbaoring/control-plane/templates/layouts"
    "github.com/Bidon15/banhbaoring/control-plane/templates/components"
)

templ OnboardingStep1(user *models.User) {
    @layouts.Auth("Welcome") {
        <div class="min-h-screen flex items-center justify-center p-4">
            <div class="w-full max-w-md">
                <!-- Progress -->
                <div class="mb-8">
                    <div class="flex justify-between items-center mb-2">
                        <span class="text-sm text-gray-400">Step 1 of 3</span>
                        <span class="text-sm text-purple-400">Organization</span>
                    </div>
                    <div class="h-1 bg-[#4a3f5c] rounded-full">
                        <div class="h-1 bg-purple-500 rounded-full" style="width: 33%"></div>
                    </div>
                </div>
                
                @components.Card("") {
                    <div class="text-center mb-6">
                        <span class="text-4xl">👋</span>
                        <h2 class="mt-4 text-2xl font-bold text-white">Welcome, { user.Name }!</h2>
                        <p class="mt-2 text-gray-400">Let's set up your organization</p>
                    </div>
                    
                    <form hx-post="/onboarding/org" hx-target="#main-content" class="space-y-4">
                        <div>
                            <label for="org-name" class="block text-sm font-medium text-gray-300 mb-1">
                                Organization Name
                            </label>
                            <input type="text" id="org-name" name="name" required
                                   class="w-full px-4 py-3 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg text-white focus:border-purple-500 focus:ring-1 focus:ring-purple-500 focus:outline-none"
                                   placeholder="My Rollup Co"/>
                        </div>
                        
                        @components.Button(components.ButtonPrimary, false) {
                            <span class="w-full text-center">Continue →</span>
                        }
                    </form>
                }
            </div>
        </div>
    }
}

templ OnboardingStep2(org *models.Organization) {
    // Step 2: Create first key
    // Similar structure with progress at 66%
}

templ OnboardingStep3(key *models.Key) {
    // Step 3: Integration guide
    // Show key details and code snippets
}
```

---

## 4. Handlers

**File:** `internal/handler/web/auth.go`

```go
package web

import (
    "net/http"
    
    "github.com/a-h/templ"
    "github.com/go-chi/chi/v5"
    
    "github.com/Bidon15/banhbaoring/control-plane/templates/pages"
)

func (h *WebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
    errorMsg := r.URL.Query().Get("error")
    templ.Handler(pages.LoginPage(errorMsg)).ServeHTTP(w, r)
}

func (h *WebHandler) Login(w http.ResponseWriter, r *http.Request) {
    email := r.FormValue("email")
    password := r.FormValue("password")
    
    user, sessionID, err := h.authService.Login(r.Context(), email, password)
    if err != nil {
        http.Redirect(w, r, "/login?error=Invalid+credentials", http.StatusFound)
        return
    }
    
    // Set session cookie
    session, _ := h.sessionStore.Get(r, "session")
    session.Values["user_id"] = user.ID.String()
    session.Values["session_id"] = sessionID
    session.Save(r, w)
    
    // Redirect based on onboarding status
    orgs, _ := h.orgService.ListUserOrgs(r.Context(), user.ID)
    if len(orgs) == 0 {
        http.Redirect(w, r, "/onboarding", http.StatusFound)
        return
    }
    
    http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (h *WebHandler) OAuthStart(w http.ResponseWriter, r *http.Request) {
    provider := chi.URLParam(r, "provider")
    authURL, _ := h.oauthService.GetAuthURL(provider, generateState())
    
    // Store state in session for CSRF protection
    session, _ := h.sessionStore.Get(r, "oauth")
    session.Values["state"] = state
    session.Save(r, w)
    
    http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *WebHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
    provider := chi.URLParam(r, "provider")
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")
    
    // Verify state
    session, _ := h.sessionStore.Get(r, "oauth")
    if session.Values["state"] != state {
        http.Redirect(w, r, "/login?error=Invalid+state", http.StatusFound)
        return
    }
    
    user, sessionID, err := h.oauthService.HandleCallback(r.Context(), provider, code)
    if err != nil {
        http.Redirect(w, r, "/login?error=OAuth+failed", http.StatusFound)
        return
    }
    
    // Set session
    authSession, _ := h.sessionStore.Get(r, "session")
    authSession.Values["user_id"] = user.ID.String()
    authSession.Values["session_id"] = sessionID
    authSession.Save(r, w)
    
    // Check if new user needs onboarding
    orgs, _ := h.orgService.ListUserOrgs(r.Context(), user.ID)
    if len(orgs) == 0 {
        http.Redirect(w, r, "/onboarding", http.StatusFound)
        return
    }
    
    http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (h *WebHandler) Logout(w http.ResponseWriter, r *http.Request) {
    session, _ := h.sessionStore.Get(r, "session")
    session.Options.MaxAge = -1
    session.Save(r, w)
    
    http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *WebHandler) RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session, _ := h.sessionStore.Get(r, "session")
        userID, ok := session.Values["user_id"].(string)
        if !ok || userID == "" {
            http.Redirect(w, r, "/login", http.StatusFound)
            return
        }
        
        // Add user to context
        ctx := context.WithValue(r.Context(), "user_id", userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 5. Deliverables

| File | Description |
|------|-------------|
| `templates/layouts/auth.templ` | Auth pages layout |
| `templates/pages/login.templ` | Login page |
| `templates/pages/signup.templ` | Signup page |
| `templates/pages/onboarding.templ` | 3-step onboarding |
| `templates/pages/forgot_password.templ` | Password reset |
| `internal/handler/web/auth.go` | Auth handlers |

---

## 6. Success Criteria

- [ ] Login page renders correctly
- [ ] OAuth buttons redirect to providers
- [ ] Email/password login works
- [ ] Session is created and persisted
- [ ] New users go to onboarding
- [ ] Existing users go to dashboard
- [ ] Logout clears session
- [ ] Onboarding flow completes in 3 steps

---

## 7. Agent Prompt

```
You are Agent 12B - Dashboard Auth Pages. Implement authentication UI.

Read: doc/implementation/IMPL_12B_DASHBOARD_AUTH.md

Deliverables:
1. Login page with OAuth + email/password
2. Signup page
3. 3-step onboarding wizard
4. Auth handlers (login, logout, OAuth)
5. Session management with gorilla/sessions
6. Forgot password page

Dependencies: Agent 12A must complete first.

Test: go build ./... && templ generate
```

