# Implementation: Dashboard Keys Pages

## Agent: 12C - Keys Management Pages

> **Phase 8.2** - Can run in parallel with 12D after Agent 12B completes.

---

## 1. Overview

Implement key management pages: list, detail, create, worker keys.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| Keys list with search/filter | ✅ |
| Key details page | ✅ |
| Create key modal | ✅ |
| Worker keys wizard | ✅ |
| Sign test | ✅ |
| Delete key | ✅ |

---

## 3. Pages

### 3.1 Keys List

**File:** `templates/pages/keys_list.templ`

```go
package pages

import (
    "github.com/Bidon15/banhbaoring/control-plane/templates/layouts"
    "github.com/Bidon15/banhbaoring/control-plane/templates/components"
    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

templ KeysListPage(user *models.User, org *models.Organization, keys []*models.Key, namespaces []*models.Namespace) {
    @layouts.Dashboard("Keys", user, org) {
        <div class="space-y-6">
            <!-- Header -->
            <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                <div>
                    <h1 class="text-2xl font-bold text-white">Keys</h1>
                    <p class="text-gray-400">Manage your cryptographic keys</p>
                </div>
                <div class="flex gap-3">
                    <button hx-get="/keys/workers/new"
                            hx-target="#modal-content"
                            @click="$dispatch('modal-open')"
                            class="px-4 py-2 border border-purple-500/50 text-purple-300 rounded-lg hover:bg-purple-500/10">
                        ⚡ Create Workers
                    </button>
                    <button hx-get="/keys/new"
                            hx-target="#modal-content"
                            @click="$dispatch('modal-open')"
                            class="px-4 py-2 bg-gradient-to-r from-purple-500 to-orange-500 text-white rounded-lg shadow-lg shadow-purple-500/25">
                        + Create Key
                    </button>
                </div>
            </div>
            
            <!-- Filters -->
            <div class="flex flex-col sm:flex-row gap-4">
                <input type="search"
                       name="q"
                       placeholder="Search keys..."
                       hx-get="/keys"
                       hx-trigger="keyup changed delay:300ms, search"
                       hx-target="#keys-list"
                       hx-push-url="true"
                       class="flex-1 px-4 py-2 bg-[#1a1625] border border-[#4a3f5c] rounded-lg text-white placeholder-gray-500 focus:border-purple-500 focus:outline-none"/>
                
                <select name="namespace"
                        hx-get="/keys"
                        hx-trigger="change"
                        hx-target="#keys-list"
                        hx-include="[name='q']"
                        class="px-4 py-2 bg-[#1a1625] border border-[#4a3f5c] rounded-lg text-white focus:border-purple-500 focus:outline-none">
                    <option value="">All Namespaces</option>
                    for _, ns := range namespaces {
                        <option value={ ns.ID.String() }>{ ns.Name }</option>
                    }
                </select>
            </div>
            
            <!-- Keys List -->
            <div id="keys-list">
                @KeysList(keys)
            </div>
        </div>
    }
}

templ KeysList(keys []*models.Key) {
    if len(keys) == 0 {
        @components.Card("") {
            <div class="text-center py-12">
                <span class="text-6xl">🔑</span>
                <h3 class="mt-4 text-lg font-semibold text-white">No keys yet</h3>
                <p class="mt-2 text-gray-400">Create your first key to get started</p>
                <button hx-get="/keys/new"
                        hx-target="#modal-content"
                        @click="$dispatch('modal-open')"
                        class="mt-4 px-4 py-2 bg-purple-500 text-white rounded-lg hover:bg-purple-600">
                    Create Key
                </button>
            </div>
        }
    } else {
        <div class="space-y-4">
            for _, key := range keys {
                @KeyCard(key)
            }
        </div>
    }
}

templ KeyCard(key *models.Key) {
    <div id={ "key-" + key.ID.String() }
         class="bg-[#1a1625] border-l-4 border-l-emerald-500 border border-[#4a3f5c] rounded-lg p-4 hover:border-purple-500/30 transition-colors">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div class="flex-1 min-w-0">
                <div class="flex items-center gap-3">
                    <h3 class="text-lg font-semibold text-white truncate">🔑 { key.Name }</h3>
                    <span class="text-xs text-purple-400 bg-purple-500/10 px-2 py-0.5 rounded">
                        { key.Namespace }
                    </span>
                </div>
                <p class="mt-1 font-mono text-sm text-gray-400 truncate">
                    { key.Address }
                </p>
                <p class="mt-1 text-xs text-gray-500">
                    Created { formatTimeAgo(key.CreatedAt) }
                </p>
            </div>
            
            <div class="flex items-center gap-2">
                <button onclick={ templ.SafeScript("copyToClipboard", key.Address) }
                        class="p-2 text-gray-400 hover:text-white hover:bg-white/5 rounded-lg"
                        title="Copy address">
                    📋
                </button>
                <a href={ templ.SafeURL("/keys/" + key.ID.String()) }
                   hx-get={ "/keys/" + key.ID.String() }
                   hx-target="#main-content"
                   hx-push-url="true"
                   class="px-3 py-1.5 text-sm text-purple-400 hover:text-purple-300 hover:bg-purple-500/10 rounded-lg">
                    View Details
                </a>
                <button hx-post={ "/keys/" + key.ID.String() + "/sign-test" }
                        hx-target="#sign-result"
                        class="px-3 py-1.5 text-sm text-cyan-400 hover:text-cyan-300 hover:bg-cyan-500/10 rounded-lg">
                    Sign Test
                </button>
            </div>
        </div>
    </div>
}
```

### 3.2 Key Details

**File:** `templates/pages/keys_detail.templ`

```go
package pages

templ KeyDetailPage(user *models.User, org *models.Organization, key *models.Key, sigStats *SigningStats) {
    @layouts.Dashboard(key.Name, user, org) {
        <div class="space-y-6">
            <!-- Header -->
            <div class="flex items-center gap-4">
                <a href="/keys" 
                   hx-get="/keys" 
                   hx-target="#main-content" 
                   hx-push-url="true"
                   class="text-gray-400 hover:text-white">
                    ← Back to Keys
                </a>
            </div>
            
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-2xl font-bold text-white flex items-center gap-3">
                        🔑 { key.Name }
                        <span class="w-2 h-2 rounded-full bg-emerald-500"></span>
                    </h1>
                </div>
            </div>
            
            <!-- Key Details Card -->
            @components.Card("Key Details") {
                <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                        <dt class="text-sm text-gray-400">ID</dt>
                        <dd class="mt-1 font-mono text-sm text-white flex items-center gap-2">
                            { key.ID.String()[:8] }...
                            <button onclick={ templ.SafeScript("copyToClipboard", key.ID.String()) }
                                    class="text-gray-400 hover:text-white">📋</button>
                        </dd>
                    </div>
                    <div>
                        <dt class="text-sm text-gray-400">Address</dt>
                        <dd class="mt-1 font-mono text-sm text-white flex items-center gap-2">
                            { key.Address[:12] }...{ key.Address[len(key.Address)-6:] }
                            <button onclick={ templ.SafeScript("copyToClipboard", key.Address) }
                                    class="text-gray-400 hover:text-white">📋</button>
                        </dd>
                    </div>
                    <div>
                        <dt class="text-sm text-gray-400">Public Key</dt>
                        <dd class="mt-1 font-mono text-sm text-white flex items-center gap-2">
                            { formatHex(key.PublicKey)[:16] }...
                            <button onclick={ templ.SafeScript("copyToClipboard", formatHex(key.PublicKey)) }
                                    class="text-gray-400 hover:text-white">📋</button>
                        </dd>
                    </div>
                    <div>
                        <dt class="text-sm text-gray-400">Algorithm</dt>
                        <dd class="mt-1 text-sm text-white">{ key.Algorithm }</dd>
                    </div>
                    <div>
                        <dt class="text-sm text-gray-400">Namespace</dt>
                        <dd class="mt-1 text-sm text-white">{ key.Namespace }</dd>
                    </div>
                    <div>
                        <dt class="text-sm text-gray-400">Created</dt>
                        <dd class="mt-1 text-sm text-white">{ key.CreatedAt.Format("Jan 2, 2006 at 3:04 PM") }</dd>
                    </div>
                    <div>
                        <dt class="text-sm text-gray-400">Exportable</dt>
                        <dd class="mt-1 text-sm text-white">
                            if key.Exportable {
                                ✅ Yes
                            } else {
                                ❌ No
                            }
                        </dd>
                    </div>
                </dl>
            }
            
            <!-- Signing Activity Chart -->
            @components.Card("Signing Activity (Last 30 Days)") {
                <canvas id="signing-chart" height="200"></canvas>
                <script>
                    new Chart(document.getElementById('signing-chart'), {
                        type: 'line',
                        data: {
                            labels: { templ.JSONScript(sigStats.Labels) },
                            datasets: [{
                                data: { templ.JSONScript(sigStats.Values) },
                                borderColor: '#a855f7',
                                backgroundColor: 'rgba(168, 85, 247, 0.1)',
                                fill: true,
                                tension: 0.4
                            }]
                        },
                        options: {
                            responsive: true,
                            plugins: { legend: { display: false } },
                            scales: {
                                x: { grid: { color: '#4a3f5c' }, ticks: { color: '#9ca3af' } },
                                y: { grid: { color: '#4a3f5c' }, ticks: { color: '#9ca3af' } }
                            }
                        }
                    });
                </script>
                <div class="mt-4 flex gap-6 text-sm">
                    <div>
                        <span class="text-gray-400">Total:</span>
                        <span class="text-white font-semibold">{ formatNumber(sigStats.Total) }</span>
                    </div>
                    <div>
                        <span class="text-gray-400">Avg/day:</span>
                        <span class="text-white font-semibold">{ formatNumber(sigStats.AvgPerDay) }</span>
                    </div>
                </div>
            }
            
            <!-- Quick Sign Test -->
            @components.Card("Quick Sign Test") {
                <form hx-post={ "/keys/" + key.ID.String() + "/sign-test" }
                      hx-target="#sign-result"
                      class="space-y-4">
                    <div>
                        <label class="block text-sm text-gray-400 mb-1">Data to sign (hex or base64)</label>
                        <textarea name="data" rows="3"
                                  class="w-full px-4 py-2 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg text-white font-mono text-sm focus:border-purple-500 focus:outline-none"
                                  placeholder="0x1234..."></textarea>
                    </div>
                    <button type="submit" class="px-4 py-2 bg-cyan-500 text-white rounded-lg hover:bg-cyan-600">
                        Sign Now
                    </button>
                </form>
                <div id="sign-result" class="mt-4"></div>
            }
            
            <!-- Danger Zone -->
            @components.Card("Danger Zone") {
                <div class="flex items-center justify-between">
                    <div>
                        <p class="text-red-400">Delete this key permanently</p>
                        <p class="text-sm text-gray-500">This action cannot be undone</p>
                    </div>
                    <button hx-delete={ "/keys/" + key.ID.String() }
                            hx-confirm="Are you sure you want to delete this key? This cannot be undone."
                            hx-target="#main-content"
                            hx-push-url="/keys"
                            class="px-4 py-2 bg-red-500/10 text-red-400 border border-red-500/50 rounded-lg hover:bg-red-500/20">
                        Delete Key
                    </button>
                </div>
            }
        </div>
    }
}
```

### 3.3 Create Key Modal

**File:** `templates/partials/key_new.templ`

```go
package partials

templ KeyNewModal(namespaces []*models.Namespace) {
    <div>
        <div class="flex items-center justify-between mb-6">
            <h2 class="text-xl font-bold text-white">Create New Key</h2>
            <button @click="$dispatch('modal-close')" class="text-gray-400 hover:text-white">✕</button>
        </div>
        
        <form hx-post="/keys"
              hx-target="#keys-list"
              hx-swap="innerHTML"
              @htmx:after-request="if(event.detail.successful) $dispatch('modal-close')">
            <div class="space-y-4">
                <div>
                    <label class="block text-sm font-medium text-gray-300 mb-1">Key Name</label>
                    <input type="text" name="name" required
                           class="w-full px-4 py-2 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg text-white focus:border-purple-500 focus:outline-none"
                           placeholder="sequencer-mainnet"/>
                </div>
                
                <div>
                    <label class="block text-sm font-medium text-gray-300 mb-1">Namespace</label>
                    <select name="namespace_id" required
                            class="w-full px-4 py-2 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg text-white focus:border-purple-500 focus:outline-none">
                        for _, ns := range namespaces {
                            <option value={ ns.ID.String() }>{ ns.Name }</option>
                        }
                    </select>
                </div>
                
                <div>
                    <label class="flex items-center gap-2">
                        <input type="checkbox" name="exportable" value="true"
                               class="w-4 h-4 rounded border-[#4a3f5c] bg-[#0c0a14] text-purple-500 focus:ring-purple-500"/>
                        <span class="text-sm text-gray-300">Allow export (less secure)</span>
                    </label>
                </div>
            </div>
            
            <div class="flex gap-3 mt-6">
                <button type="button" @click="$dispatch('modal-close')"
                        class="flex-1 px-4 py-2 border border-[#4a3f5c] text-gray-300 rounded-lg hover:bg-white/5">
                    Cancel
                </button>
                <button type="submit"
                        class="flex-1 px-4 py-2 bg-gradient-to-r from-purple-500 to-orange-500 text-white rounded-lg">
                    Create Key
                </button>
            </div>
        </form>
    </div>
}
```

---

## 4. Handlers

**File:** `internal/handler/web/keys.go`

```go
package web

import (
    "net/http"
    
    "github.com/a-h/templ"
    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    
    "github.com/Bidon15/banhbaoring/control-plane/templates/pages"
    "github.com/Bidon15/banhbaoring/control-plane/templates/partials"
)

func (h *WebHandler) KeysList(w http.ResponseWriter, r *http.Request) {
    user, org := h.getUserAndOrg(r)
    
    var nsID *uuid.UUID
    if ns := r.URL.Query().Get("namespace"); ns != "" {
        id, _ := uuid.Parse(ns)
        nsID = &id
    }
    
    keys, _ := h.keyService.List(r.Context(), org.ID, nsID)
    namespaces, _ := h.orgService.ListNamespaces(r.Context(), org.ID)
    
    // If HTMX request, return partial
    if r.Header.Get("HX-Request") == "true" {
        templ.Handler(pages.KeysList(keys)).ServeHTTP(w, r)
        return
    }
    
    templ.Handler(pages.KeysListPage(user, org, keys, namespaces)).ServeHTTP(w, r)
}

func (h *WebHandler) KeysDetail(w http.ResponseWriter, r *http.Request) {
    user, org := h.getUserAndOrg(r)
    keyID, _ := uuid.Parse(chi.URLParam(r, "id"))
    
    key, _ := h.keyService.Get(r.Context(), org.ID, keyID)
    sigStats := h.getSigningStats(r.Context(), keyID)
    
    templ.Handler(pages.KeyDetailPage(user, org, key, sigStats)).ServeHTTP(w, r)
}

func (h *WebHandler) KeysNew(w http.ResponseWriter, r *http.Request) {
    _, org := h.getUserAndOrg(r)
    namespaces, _ := h.orgService.ListNamespaces(r.Context(), org.ID)
    
    templ.Handler(partials.KeyNewModal(namespaces)).ServeHTTP(w, r)
}

func (h *WebHandler) KeysCreate(w http.ResponseWriter, r *http.Request) {
    _, org := h.getUserAndOrg(r)
    
    nsID, _ := uuid.Parse(r.FormValue("namespace_id"))
    exportable := r.FormValue("exportable") == "true"
    
    _, err := h.keyService.Create(r.Context(), service.CreateKeyRequest{
        OrgID:       org.ID,
        NamespaceID: nsID,
        Name:        r.FormValue("name"),
        Exportable:  exportable,
    })
    
    if err != nil {
        // Return error toast
        templ.Handler(components.Toast(err.Error(), components.ToastError)).ServeHTTP(w, r)
        return
    }
    
    // Return updated keys list + success toast
    keys, _ := h.keyService.List(r.Context(), org.ID, nil)
    
    w.Header().Set("HX-Trigger", "modal-close")
    templ.Handler(pages.KeysList(keys)).ServeHTTP(w, r)
}

func (h *WebHandler) KeysSignTest(w http.ResponseWriter, r *http.Request) {
    _, org := h.getUserAndOrg(r)
    keyID, _ := uuid.Parse(chi.URLParam(r, "id"))
    
    data := r.FormValue("data")
    if data == "" {
        data = "test message"
    }
    
    result, err := h.keyService.Sign(r.Context(), org.ID, keyID, []byte(data), false)
    
    templ.Handler(partials.SignResult(result, err)).ServeHTTP(w, r)
}

func (h *WebHandler) KeysDelete(w http.ResponseWriter, r *http.Request) {
    _, org := h.getUserAndOrg(r)
    keyID, _ := uuid.Parse(chi.URLParam(r, "id"))
    
    _ = h.keyService.Delete(r.Context(), org.ID, keyID)
    
    // Return keys list
    keys, _ := h.keyService.List(r.Context(), org.ID, nil)
    templ.Handler(pages.KeysList(keys)).ServeHTTP(w, r)
}
```

---

## 5. Deliverables

| File | Description |
|------|-------------|
| `templates/pages/keys_list.templ` | Keys list page |
| `templates/pages/keys_detail.templ` | Key details page |
| `templates/partials/key_new.templ` | Create key modal |
| `templates/partials/worker_keys.templ` | Worker keys wizard |
| `templates/partials/sign_result.templ` | Sign test result |
| `internal/handler/web/keys.go` | Keys handlers |

---

## 6. Success Criteria

- [ ] Keys list renders with search/filter
- [ ] HTMX partial updates work (no full page reload)
- [ ] Key details page shows all info
- [ ] Create key modal works
- [ ] Worker keys wizard creates batch
- [ ] Sign test shows result
- [ ] Delete key with confirmation
- [ ] Copy to clipboard works

---

## 7. Agent Prompt

```
You are Agent 12C - Dashboard Keys Pages. Implement key management UI.

Read: doc/implementation/IMPL_12C_DASHBOARD_KEYS.md

Deliverables:
1. Keys list page with search/filter (HTMX)
2. Key details page with signing chart
3. Create key modal
4. Worker keys batch creation wizard
5. Sign test functionality
6. Delete key with confirmation
7. Handlers for all routes

Dependencies: Agent 12A and 12B must complete first.

Test: go build ./... && templ generate
```

