# Agent Task: Rebrand SDK-Rust

> **Parallel Execution:** ✅ Can run independently
> **Dependencies:** None
> **Estimated Time:** 1-2 hours

---

## Objective

Rename SDK-Rust from `banhbaoring` to `popsigner`.

---

## Scope

### Files to Modify

| File | Changes |
|------|---------|
| `sdk-rust/Cargo.toml` | Crate name, description |
| `sdk-rust/Cargo.lock` | Crate name (regenerate) |
| `sdk-rust/src/lib.rs` | Module docs, exports |
| `sdk-rust/src/client.rs` | Types, API key validation |
| `sdk-rust/src/error.rs` | Error types |
| `sdk-rust/src/types.rs` | Type docs |
| `sdk-rust/src/keys.rs` | Docs |
| `sdk-rust/src/sign.rs` | Docs |
| `sdk-rust/src/orgs.rs` | Docs |
| `sdk-rust/src/audit.rs` | Docs |
| `sdk-rust/tests/*.rs` | Imports, API keys |
| `sdk-rust/examples/*.rs` | Imports, API keys |

---

## Implementation

### Step 1: Update Cargo.toml

```toml
# Before
[package]
name = "banhbaoring"
version = "0.1.0"
description = "BanhBaoRing Rust SDK"

# After
[package]
name = "popsigner"
version = "1.0.0"
description = "POPSigner Rust SDK - Point-of-Presence signing infrastructure"
documentation = "https://docs.popsigner.io"
homepage = "https://popsigner.io"
repository = "https://github.com/Bidon15/banhbaoring"
keywords = ["signing", "cryptography", "celestia", "cosmos", "infrastructure"]
categories = ["cryptography", "api-bindings"]
```

### Step 2: Update src/lib.rs

```rust
// Before
//! BanhBaoRing Rust SDK
//!
//! Provides a client for the BanhBaoRing Control Plane API.

pub use client::Client;
pub use error::BanhBaoRingError;

// After
//! POPSigner Rust SDK
//!
//! Point-of-Presence signing infrastructure.
//! Deploy inline with execution. Keys remain remote. You remain sovereign.
//!
//! # Example
//!
//! ```rust
//! use popsigner::Client;
//!
//! let client = Client::new("psk_live_xxx");
//! ```

pub use client::Client;
pub use error::POPSignerError;
```

### Step 3: Update src/error.rs

```rust
// Before
#[derive(Debug, thiserror::Error)]
pub enum BanhBaoRingError {
    #[error("banhbaoring: unauthorized")]
    Unauthorized,
    // ...
}

// After
#[derive(Debug, thiserror::Error)]
pub enum POPSignerError {
    #[error("popsigner: unauthorized")]
    Unauthorized,
    // ...
}
```

### Step 4: Update src/client.rs

```rust
// Before
const API_KEY_PREFIX: &str = "bbr_";

impl Client {
    pub fn new(api_key: &str) -> Result<Self, BanhBaoRingError> {
        if !api_key.starts_with(API_KEY_PREFIX) {
            return Err(BanhBaoRingError::InvalidApiKey);
        }
        // ...
    }
}

// After
const API_KEY_PREFIX: &str = "psk_";

impl Client {
    pub fn new(api_key: &str) -> Result<Self, POPSignerError> {
        if !api_key.starts_with(API_KEY_PREFIX) {
            return Err(POPSignerError::InvalidApiKey);
        }
        // ...
    }
}
```

### Step 5: Update HTTP Headers

```rust
// Before
request.header("X-BanhBaoRing-API-Key", &self.api_key)

// After
request.header("X-POPSigner-API-Key", &self.api_key)
```

### Step 6: Update Examples

```rust
// sdk-rust/examples/basic.rs

// Before
use banhbaoring::Client;

fn main() {
    let client = Client::new("bbr_live_xxx").unwrap();
}

// After
use popsigner::Client;

fn main() {
    let client = Client::new("psk_live_xxx").unwrap();
}
```

### Step 7: Update Tests

```rust
// sdk-rust/tests/client_test.rs

// Before
use banhbaoring::{Client, BanhBaoRingError};

#[test]
fn test_invalid_api_key() {
    let result = Client::new("invalid");
    assert!(matches!(result, Err(BanhBaoRingError::InvalidApiKey)));
}

// After
use popsigner::{Client, POPSignerError};

#[test]
fn test_invalid_api_key() {
    let result = Client::new("invalid");
    assert!(matches!(result, Err(POPSignerError::InvalidApiKey)));
}
```

### Step 8: Regenerate Cargo.lock

```bash
cd sdk-rust
rm Cargo.lock
cargo build
```

---

## Verification

```bash
cd sdk-rust

# Build
cargo build

# Test
cargo test

# Check for remaining references
grep -r "banhbaoring" --include="*.rs" .
grep -r "BanhBaoRing" --include="*.rs" .
grep -r "bbr_" --include="*.rs" .

# Verify examples compile
cargo build --examples
```

---

## Checklist

```
□ Update Cargo.toml - name, description, version
□ Update src/lib.rs - module docs, exports
□ Update src/error.rs - error type names
□ Update src/client.rs - API key prefix, types
□ Update HTTP header name
□ Update src/types.rs - docs
□ Update src/keys.rs - docs
□ Update src/sign.rs - docs
□ Update src/orgs.rs - docs
□ Update src/audit.rs - docs
□ Update examples/basic.rs
□ Update examples/parallel_workers.rs
□ Update tests/client_test.rs
□ Update tests/keys_test.rs
□ Update tests/sign_test.rs
□ Regenerate Cargo.lock
□ cargo build passes
□ cargo test passes
□ No remaining "banhbaoring" or "bbr_" references
```

---

## Output

After completion, the crate is usable as:

```rust
use popsigner::Client;

let client = Client::new("psk_live_xxx")?;
```

