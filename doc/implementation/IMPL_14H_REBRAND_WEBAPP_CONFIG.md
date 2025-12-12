# Agent Task: Rebrand Web App - Config & Static Assets

> **Parallel Execution:** ✅ Can run independently
> **Dependencies:** None
> **Estimated Time:** 1 hour

---

## Objective

Update control-plane configuration files, static assets, and build files.

---

## Scope

### Files to Modify

| File | Changes |
|------|---------|
| `control-plane/config.yaml` | App name, URLs |
| `control-plane/config/config.example.yaml` | App name, URLs |
| `control-plane/go.mod` | Module description |
| `control-plane/Makefile` | Binary names |
| `control-plane/docker/Dockerfile` | Image labels |
| `control-plane/docker/docker-compose.yml` | Service names |
| `control-plane/tailwind.config.js` | Theme colors (optional) |
| `control-plane/static/css/input.css` | CSS variables |
| `control-plane/static/js/app.js` | Any branding refs |
| `control-plane/README.md` | Documentation |
| `control-plane/internal/config/config.go` | Default values |

---

## Implementation

### config.yaml

```yaml
# Before
app:
  name: BanhBaoRing
  url: https://banhbaoring.io

# After
app:
  name: POPSigner
  url: https://popsigner.com
  description: Point-of-Presence signing infrastructure
```

### config/config.example.yaml

```yaml
# POPSigner Control Plane Configuration
# Copy to config.yaml and update values

app:
  name: POPSigner
  url: https://popsigner.com
  
server:
  host: 0.0.0.0
  port: 8080

# ...
```

### Makefile

```makefile
# Before
.PHONY: build
build:
	go build -o banhbaoring-server ./cmd/server

# After
.PHONY: build
build:
	go build -o popsigner-server ./cmd/server

.PHONY: docker
docker:
	docker build -t popsigner-control-plane:dev .
```

### docker/Dockerfile

```dockerfile
# Before
LABEL org.opencontainers.image.title="BanhBaoRing Control Plane"

# After
LABEL org.opencontainers.image.title="POPSigner Control Plane"
LABEL org.opencontainers.image.description="POPSigner - Point-of-Presence signing infrastructure"
```

### docker/docker-compose.yml

```yaml
# Before
services:
  banhbaoring:
    image: banhbaoring-control-plane:dev
    container_name: banhbaoring

# After
services:
  popsigner:
    image: popsigner-control-plane:dev
    container_name: popsigner
```

### tailwind.config.js - 80s CRT Terminal Aesthetic

```javascript
// 1980s Bloomberg Terminal / CRT Phosphor Aesthetic
// Amber + Green phosphor on black. Stranger Things S5 vibes.
module.exports = {
  darkMode: 'class', // Always dark (CRTs were black)
  theme: {
    extend: {
      colors: {
        // PRIMARY: Amber Phosphor (CRT glow)
        phosphor: {
          amber: '#FFB000',       // Main amber
          'amber-bright': '#FFCC00', // Highlighted
          'amber-dim': '#CC8800',    // Dimmed
          'amber-dark': '#333300',   // Borders
        },
        // SECONDARY: Green Phosphor
        crt: {
          green: '#33FF00',       // Bright green
          'green-dim': '#228B22', // Forest green
          'green-dark': '#1A4D1A', // Very dark green
          'green-bg': '#0D1A0D',  // Subtle green tint
        },
        // ALERT: Terminal Red
        terminal: {
          red: '#FF3333',
          'red-dim': '#CC2222',
          black: '#000000',
        },
      },
      fontFamily: {
        // Monospace ONLY for terminal feel
        mono: ['IBM Plex Mono', 'VT323', 'Courier New', monospace],
        terminal: ['VT323', 'IBM Plex Mono', monospace],
      },
      boxShadow: {
        // Phosphor glow effects
        'glow-amber': '0 0 10px #FFB000, 0 0 20px #FFB000',
        'glow-green': '0 0 10px #33FF00, 0 0 20px #33FF00',
        'glow-red': '0 0 10px #FF3333',
      },
    },
  },
}
```

### static/css/input.css - 80s CRT Phosphor Theme

```css
/* POPSigner 80s CRT Terminal Aesthetic
   Amber + Green phosphor on black
   Stranger Things S5 / Bloomberg 1980s vibes */

@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  /* === PHOSPHOR AMBER (Primary) === */
  --phosphor-amber: #FFB000;
  --phosphor-amber-bright: #FFCC00;
  --phosphor-amber-dim: #CC8800;
  --phosphor-amber-dark: #333300;
  
  /* === PHOSPHOR GREEN (Secondary) === */
  --phosphor-green: #33FF00;
  --phosphor-green-dim: #228B22;
  --phosphor-green-dark: #1A4D1A;
  --phosphor-green-bg: #0D1A0D;
  
  /* === TERMINAL RED (Alerts) === */
  --terminal-red: #FF3333;
  --terminal-red-dim: #CC2222;
  
  /* === CRT BACKGROUNDS === */
  --bg-crt: #000000;
  --bg-surface: #0A0A0A;
  --bg-elevated: #111111;
  --bg-glow: #0D1A0D;
  
  /* === TEXT (Phosphor colors) === */
  --text-primary: #FFB000;    /* Amber */
  --text-secondary: #33FF00;  /* Green */
  --text-muted: #666600;      /* Dim amber */
  --text-dim: #336633;        /* Dim green */
}

/* CRT Monitor base */
body {
  background-color: var(--bg-crt);
  color: var(--text-primary);
  font-family: 'IBM Plex Mono', 'VT323', monospace;
}

/* Phosphor glow utility */
.glow-amber {
  text-shadow: 0 0 8px var(--phosphor-amber);
}
.glow-green {
  text-shadow: 0 0 8px var(--phosphor-green);
}
.glow-red {
  text-shadow: 0 0 8px var(--terminal-red);
}

/* CRT scanline overlay */
.crt-scanlines::before {
  content: '';
  position: absolute;
  inset: 0;
  background: repeating-linear-gradient(
    0deg,
    rgba(0, 0, 0, 0.15),
    rgba(0, 0, 0, 0.15) 1px,
    transparent 1px,
    transparent 2px
  );
  pointer-events: none;
}

/* Terminal button */
.btn-terminal {
  background-color: var(--phosphor-amber);
  color: #000000;
  font-weight: bold;
  text-transform: uppercase;
}
.btn-terminal:hover {
  background-color: var(--phosphor-amber-bright);
  box-shadow: 0 0 20px var(--phosphor-amber);
}

/* Terminal card */
.card-crt {
  background-color: var(--bg-crt);
  border: 1px solid var(--phosphor-amber-dark);
}
.card-crt:hover {
  border-color: var(--phosphor-amber);
  box-shadow: 0 0 10px rgba(255, 176, 0, 0.3);
}

/* Green variant */
.card-crt-green {
  border-color: var(--phosphor-green-dark);
}
.card-crt-green:hover {
  border-color: var(--phosphor-green);
  box-shadow: 0 0 10px rgba(51, 255, 0, 0.3);
}
```

### internal/config/config.go

```go
// Before
const (
    DefaultAppName = "BanhBaoRing"
    DefaultAppURL  = "https://banhbaoring.io"
)

// After
const (
    DefaultAppName = "POPSigner"
    DefaultAppURL  = "https://popsigner.com"
)
```

### control-plane/README.md

```markdown
# POPSigner Control Plane

Point-of-Presence signing infrastructure - Control Plane API.

## Overview

The control plane provides the multi-tenant API for POPSigner,
including key management, signing operations, and billing.

...
```

---

## Static Assets

### Create New Logo

Create `control-plane/static/img/logo.svg`:

```svg
<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
  <!-- Simple geometric diamond/node shape -->
  <path d="M12 2L22 12L12 22L2 12L12 2Z" 
        stroke="currentColor" 
        stroke-width="2" 
        fill="none"/>
</svg>
```

Remove or replace bell-related assets if any exist.

---

## Verification

```bash
cd control-plane

# Build CSS
npx tailwindcss -i static/css/input.css -o static/css/output.css

# Build
go build ./...

# Run
go run ./cmd/server

# Check config loads correctly
# Check for remaining references
grep -r "banhbao" . --include="*.yaml" --include="*.yml" --include="*.go"
grep -r "BanhBao" . --include="*.yaml" --include="*.yml" --include="*.go"
```

---

## Checklist

```
□ config.yaml - app name, URLs
□ config/config.example.yaml - app name, URLs
□ go.mod - module description
□ Makefile - binary names
□ docker/Dockerfile - image labels
□ docker/docker-compose.yml - service names
□ tailwind.config.js - colors (optional)
□ static/css/input.css - CSS variables
□ static/js/app.js - any branding
□ internal/config/config.go - default values
□ README.md - documentation
□ Create new logo.svg (geometric, no emoji)
□ go build passes
□ CSS build passes
□ No remaining "banhbao" references
```

---

## Output

After completion, the control plane config and assets reflect POPSigner branding.

