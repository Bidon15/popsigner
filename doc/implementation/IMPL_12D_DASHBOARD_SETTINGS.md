# Implementation: Dashboard Settings & Billing

## Agent: 12D - Settings & Billing Pages

> **Phase 8.2** - Can run in parallel with 12C after Agent 12B completes.

---

## 1. Overview

Implement settings pages: profile, team, API keys, billing, audit log, usage.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| Profile settings | ✅ |
| Team management | ✅ |
| API keys management | ✅ |
| Billing page (Stripe) | ✅ |
| Audit log viewer | ✅ |
| Usage statistics | ✅ |

---

## 3. Pages

### 3.1 Billing Page

**File:** `templates/pages/billing.templ`

```go
package pages

templ BillingPage(user *models.User, org *models.Organization, sub *SubscriptionInfo, usage *UsageInfo, invoices []*Invoice) {
    @layouts.Dashboard("Billing", user, org) {
        <div class="space-y-6">
            <h1 class="text-2xl font-bold text-white">Billing</h1>
            
            <!-- Current Plan -->
            @components.Card("Current Plan") {
                <div class="flex items-start justify-between">
                    <div>
                        <div class="flex items-center gap-3">
                            <span class="text-3xl">
                                if sub.Plan == "pro" {
                                    🚀
                                } else if sub.Plan == "enterprise" {
                                    💎
                                } else {
                                    🆓
                                }
                            </span>
                            <div>
                                <h3 class="text-xl font-bold text-white uppercase">{ sub.Plan } Plan</h3>
                                <p class="text-gray-400">
                                    if sub.Plan == "free" {
                                        Free forever
                                    } else {
                                        ${ sub.Price }/month
                                    }
                                </p>
                            </div>
                        </div>
                        
                        <!-- Usage Bars -->
                        <div class="mt-6 space-y-4">
                            @UsageBar("Keys", usage.Keys, usage.KeysLimit)
                            @UsageBar("Signatures", usage.Signatures, usage.SignaturesLimit)
                            @UsageBar("Team Members", usage.TeamMembers, usage.TeamMembersLimit)
                        </div>
                    </div>
                    
                    <button hx-get="/settings/billing/upgrade"
                            hx-target="#modal-content"
                            @click="$dispatch('modal-open')"
                            class="px-4 py-2 bg-gradient-to-r from-purple-500 to-orange-500 text-white rounded-lg">
                        Upgrade Plan
                    </button>
                </div>
            }
            
            <!-- Payment Method -->
            @components.Card("Payment Method") {
                <div class="space-y-4">
                    <!-- Card Payment (Stripe) -->
                    <div class="flex items-center gap-4 p-4 border border-purple-500 bg-purple-500/10 rounded-lg">
                        <span class="text-2xl">💳</span>
                        <div class="flex-1">
                            <p class="font-medium text-white">Credit Card</p>
                            if sub.CardLast4 != "" {
                                <p class="text-sm text-gray-400">Visa ending in { sub.CardLast4 }</p>
                            } else {
                                <p class="text-sm text-gray-400">No card on file</p>
                            }
                        </div>
                        <button hx-get="/settings/billing/card"
                                hx-target="#modal-content"
                                @click="$dispatch('modal-open')"
                                class="text-sm text-purple-400 hover:text-purple-300">
                            { sub.CardLast4 != "" ? "Change" : "Add Card" }
                        </button>
                    </div>
                </div>
            }
            
            <!-- Invoices -->
            @components.Card("Invoices") {
                <div class="overflow-x-auto">
                    <table class="w-full">
                        <thead>
                            <tr class="text-left text-sm text-gray-400 border-b border-[#4a3f5c]">
                                <th class="pb-3">Date</th>
                                <th class="pb-3">Description</th>
                                <th class="pb-3">Amount</th>
                                <th class="pb-3">Status</th>
                                <th class="pb-3"></th>
                            </tr>
                        </thead>
                        <tbody class="text-sm">
                            for _, inv := range invoices {
                                <tr class="border-b border-[#4a3f5c]/50">
                                    <td class="py-3 text-white">{ inv.Date.Format("Jan 2, 2006") }</td>
                                    <td class="py-3 text-gray-300">{ inv.Description }</td>
                                    <td class="py-3 text-white">${ formatAmount(inv.Amount) }</td>
                                    <td class="py-3">
                                        if inv.Paid {
                                            <span class="text-emerald-400">Paid ✓</span>
                                        } else {
                                            <span class="text-amber-400">Pending</span>
                                        }
                                    </td>
                                    <td class="py-3">
                                        <a href={ templ.SafeURL(inv.DownloadURL) } class="text-purple-400 hover:text-purple-300">
                                            Download
                                        </a>
                                    </td>
                                </tr>
                            }
                        </tbody>
                    </table>
                </div>
            }
        </div>
    }
}

templ UsageBar(label string, current, limit int64) {
    <div>
        <div class="flex justify-between text-sm mb-1">
            <span class="text-gray-400">{ label }</span>
            <span class="text-white">
                { formatNumber(current) }
                if limit > 0 {
                    / { formatNumber(limit) }
                } else {
                    (unlimited)
                }
            </span>
        </div>
        <div class="h-2 bg-[#4a3f5c] rounded-full overflow-hidden">
            if limit > 0 {
                <div class={ "h-full rounded-full transition-all", usageBarColor(current, limit) }
                     style={ fmt.Sprintf("width: %d%%", min(100, current*100/limit)) }></div>
            } else {
                <div class="h-full w-full bg-emerald-500/50 rounded-full"></div>
            }
        </div>
    </div>
}

func usageBarColor(current, limit int64) string {
    pct := float64(current) / float64(limit) * 100
    if pct >= 90 {
        return "bg-red-500"
    } else if pct >= 70 {
        return "bg-amber-500"
    }
    return "bg-emerald-500"
}
```

### 3.2 Stripe Card Modal

**File:** `templates/partials/stripe_card.templ`

```go
package partials

templ StripeCardModal(clientSecret string) {
    <div>
        <div class="flex items-center justify-between mb-6">
            <h2 class="text-xl font-bold text-white flex items-center gap-2">
                💳 Update Payment Method
            </h2>
            <button @click="$dispatch('modal-close')" class="text-gray-400 hover:text-white">✕</button>
        </div>
        
        <!-- Stripe Elements container -->
        <div id="stripe-elements" class="mb-6">
            <!-- Stripe.js will inject the card element here -->
            <div id="card-element" class="p-4 bg-[#0c0a14] border border-[#4a3f5c] rounded-lg"></div>
            <div id="card-errors" class="mt-2 text-sm text-red-400"></div>
        </div>
        
        <div class="flex gap-3">
            <button type="button" @click="$dispatch('modal-close')"
                    class="flex-1 px-4 py-2 border border-[#4a3f5c] text-gray-300 rounded-lg hover:bg-white/5">
                Cancel
            </button>
            <button id="submit-card"
                    class="flex-1 px-4 py-2 bg-gradient-to-r from-purple-500 to-orange-500 text-white rounded-lg">
                Save Card
            </button>
        </div>
        
        <script>
            // Initialize Stripe Elements
            const stripe = Stripe('{{ .StripePublicKey }}');
            const elements = stripe.elements({ clientSecret: '{ clientSecret }' });
            const cardElement = elements.create('card', {
                style: {
                    base: {
                        color: '#faf5ff',
                        fontFamily: 'Outfit, system-ui, sans-serif',
                        fontSize: '16px',
                        '::placeholder': { color: '#9ca3af' }
                    }
                }
            });
            cardElement.mount('#card-element');
            
            document.getElementById('submit-card').addEventListener('click', async () => {
                const { error } = await stripe.confirmSetup({
                    elements,
                    confirmParams: { return_url: window.location.origin + '/settings/billing' }
                });
                if (error) {
                    document.getElementById('card-errors').textContent = error.message;
                }
            });
        </script>
    </div>
}
```

### 3.3 Audit Log Page

**File:** `templates/pages/audit.templ`

```go
package pages

templ AuditPage(user *models.User, org *models.Organization, logs []*models.AuditLog, nextCursor string) {
    @layouts.Dashboard("Audit Log", user, org) {
        <div class="space-y-6">
            <div class="flex items-center justify-between">
                <h1 class="text-2xl font-bold text-white">Audit Log</h1>
                <button class="px-4 py-2 border border-[#4a3f5c] text-gray-300 rounded-lg hover:bg-white/5">
                    📥 Export CSV
                </button>
            </div>
            
            <!-- Filters -->
            <div class="flex flex-wrap gap-4">
                <select name="event"
                        hx-get="/audit"
                        hx-trigger="change"
                        hx-target="#audit-list"
                        class="px-4 py-2 bg-[#1a1625] border border-[#4a3f5c] rounded-lg text-white">
                    <option value="">All Events</option>
                    <option value="key.created">Key Created</option>
                    <option value="key.signed">Key Signed</option>
                    <option value="key.deleted">Key Deleted</option>
                    <option value="auth.login">Login</option>
                </select>
                
                <select name="period"
                        hx-get="/audit"
                        hx-trigger="change"
                        hx-target="#audit-list"
                        hx-include="[name='event']"
                        class="px-4 py-2 bg-[#1a1625] border border-[#4a3f5c] rounded-lg text-white">
                    <option value="7d">Last 7 days</option>
                    <option value="30d">Last 30 days</option>
                    <option value="90d">Last 90 days</option>
                </select>
            </div>
            
            <!-- Audit Table -->
            <div id="audit-list">
                @AuditList(logs, nextCursor)
            </div>
        </div>
    }
}

templ AuditList(logs []*models.AuditLog, nextCursor string) {
    @components.Card("") {
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <thead>
                    <tr class="text-left text-gray-400 border-b border-[#4a3f5c]">
                        <th class="pb-3">Time</th>
                        <th class="pb-3">Event</th>
                        <th class="pb-3">Resource</th>
                        <th class="pb-3">Actor</th>
                        <th class="pb-3">IP</th>
                    </tr>
                </thead>
                <tbody>
                    for _, log := range logs {
                        <tr class="border-b border-[#4a3f5c]/50 hover:bg-white/5">
                            <td class="py-3 text-gray-400">{ formatTimeAgo(log.CreatedAt) }</td>
                            <td class="py-3">
                                <span class={ eventBadgeClass(log.Event) }>
                                    { log.Event }
                                </span>
                            </td>
                            <td class="py-3 text-white">
                                if log.ResourceType != "" {
                                    { log.ResourceType }
                                } else {
                                    -
                                }
                            </td>
                            <td class="py-3 text-gray-300">{ log.ActorType }</td>
                            <td class="py-3 text-gray-500 font-mono text-xs">{ log.IPAddress }</td>
                        </tr>
                    }
                </tbody>
            </table>
        </div>
        
        if nextCursor != "" {
            <div class="mt-4 text-center">
                <button hx-get={ "/audit?cursor=" + nextCursor }
                        hx-target="#audit-list"
                        hx-swap="innerHTML"
                        class="text-purple-400 hover:text-purple-300">
                    Load more...
                </button>
            </div>
        }
    }
}
```

---

## 4. Deliverables

| File | Description |
|------|-------------|
| `templates/pages/settings_profile.templ` | Profile settings |
| `templates/pages/settings_team.templ` | Team management |
| `templates/pages/settings_apikeys.templ` | API keys management |
| `templates/pages/billing.templ` | Billing page |
| `templates/pages/audit.templ` | Audit log |
| `templates/pages/usage.templ` | Usage statistics |
| `templates/partials/stripe_card.templ` | Stripe card modal |
| `internal/handler/web/settings.go` | Settings handlers |
| `internal/handler/web/billing.go` | Billing handlers |
| `internal/handler/web/audit.go` | Audit handlers |

---

## 5. Success Criteria

- [ ] Profile settings update works
- [ ] Team invite/remove works
- [ ] API keys create/revoke works
- [ ] Billing page shows plan and usage
- [ ] Upgrade plan modal works
- [ ] Stripe card update works
- [ ] Audit log with filters works
- [ ] Usage charts render correctly

---

## 6. Agent Prompt

```
You are Agent 12D - Dashboard Settings & Billing. Implement settings and billing UI.

Read: doc/implementation/IMPL_12D_DASHBOARD_SETTINGS.md

Deliverables:
1. Profile settings page
2. Team management (invite, remove, roles)
3. API keys management
4. Billing page with Stripe (plan and usage)
5. Stripe card update modal
6. Audit log with filters
7. Usage statistics page
8. All handlers

Dependencies: Agent 12A and 12B must complete first.

Test: go build ./... && templ generate
```

