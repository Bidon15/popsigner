# Implementation: Rust SDK

## Agent: 11B - Rust SDK

> **Phase 7** - Can run in parallel with 11A after Control Plane API is stable.

---

## 1. Overview

Create the official Rust SDK for the BanhBaoRing Control Plane API.

**Crate:** `banhbaoring`

> **Why Rust?** Celestia has official clients in Go and Rust only. The Rust client (`celestia-node-rs`) is used by rollup teams building with Rust-based sequencers.

---

## 2. Features

| Feature | Priority |
|---------|----------|
| Authentication (API keys) | P0 |
| Key management (CRUD) | P0 |
| Signing operations | P0 |
| Batch operations | P0 |
| Async/await support | P0 |
| Organizations | P1 |
| Audit logs | P1 |

---

## 3. Project Structure

```
sdk-rust/
├── src/
│   ├── lib.rs              # Main exports
│   ├── client.rs           # BanhBaoRing client
│   ├── keys.rs             # Key management
│   ├── sign.rs             # Signing operations
│   ├── orgs.rs             # Organizations
│   ├── audit.rs            # Audit logs
│   ├── types.rs            # Types
│   ├── error.rs            # Error types
│   └── http.rs             # HTTP client (reqwest)
├── examples/
│   ├── basic.rs
│   ├── parallel_workers.rs
│   └── celestia_integration.rs
├── tests/
│   ├── client_test.rs
│   ├── keys_test.rs
│   └── sign_test.rs
├── Cargo.toml
└── README.md
```

---

## 4. Cargo.toml

```toml
[package]
name = "banhbaoring"
version = "0.1.0"
edition = "2021"
authors = ["BanhBaoRing"]
description = "Official Rust SDK for BanhBaoRing - Secure key management for Celestia"
license = "MIT OR Apache-2.0"
repository = "https://github.com/banhbaoring/sdk-rust"
keywords = ["banhbaoring", "celestia", "cosmos", "keyring", "signing"]
categories = ["cryptography", "api-bindings"]

[dependencies]
reqwest = { version = "0.11", features = ["json", "rustls-tls"] }
tokio = { version = "1", features = ["full"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
thiserror = "1"
base64 = "0.21"
uuid = { version = "1", features = ["v4", "serde"] }
async-trait = "0.1"

[dev-dependencies]
tokio-test = "0.4"
wiremock = "0.5"
```

---

## 5. Client Implementation

**File:** `src/client.rs`

```rust
use crate::error::{BanhBaoRingError, Result};
use crate::keys::KeysClient;
use crate::sign::SignClient;
use crate::orgs::OrgsClient;
use crate::audit::AuditClient;
use reqwest::{Client as HttpClient, header};
use std::time::Duration;

const DEFAULT_BASE_URL: &str = "https://api.banhbaoring.io";
const DEFAULT_TIMEOUT_SECS: u64 = 30;

/// BanhBaoRing API client.
///
/// # Example
///
/// ```rust
/// use banhbaoring::Client;
///
/// #[tokio::main]
/// async fn main() -> Result<(), Box<dyn std::error::Error>> {
///     let client = Client::new("bbr_live_xxxxx");
///     
///     // Create a key
///     let key = client.keys().create(CreateKeyRequest {
///         name: "sequencer".to_string(),
///         namespace_id: namespace_id,
///         ..Default::default()
///     }).await?;
///     
///     // Sign data
///     let sig = client.sign().sign(&key.id, &tx_bytes, false).await?;
///     Ok(())
/// }
/// ```
#[derive(Clone)]
pub struct Client {
    http: HttpClient,
    base_url: String,
    api_key: String,
}

/// Configuration options for the client.
pub struct ClientConfig {
    pub base_url: Option<String>,
    pub timeout: Option<Duration>,
}

impl Default for ClientConfig {
    fn default() -> Self {
        Self {
            base_url: None,
            timeout: None,
        }
    }
}

impl Client {
    /// Create a new BanhBaoRing client with default configuration.
    pub fn new(api_key: impl Into<String>) -> Self {
        Self::with_config(api_key, ClientConfig::default())
    }

    /// Create a new BanhBaoRing client with custom configuration.
    pub fn with_config(api_key: impl Into<String>, config: ClientConfig) -> Self {
        let timeout = config.timeout.unwrap_or(Duration::from_secs(DEFAULT_TIMEOUT_SECS));
        
        let http = HttpClient::builder()
            .timeout(timeout)
            .build()
            .expect("Failed to create HTTP client");

        Self {
            http,
            base_url: config.base_url.unwrap_or_else(|| DEFAULT_BASE_URL.to_string()),
            api_key: api_key.into(),
        }
    }

    /// Get the keys client.
    pub fn keys(&self) -> KeysClient {
        KeysClient::new(self.clone())
    }

    /// Get the sign client.
    pub fn sign(&self) -> SignClient {
        SignClient::new(self.clone())
    }

    /// Get the orgs client.
    pub fn orgs(&self) -> OrgsClient {
        OrgsClient::new(self.clone())
    }

    /// Get the audit client.
    pub fn audit(&self) -> AuditClient {
        AuditClient::new(self.clone())
    }

    /// Make an authenticated GET request.
    pub(crate) async fn get<T: serde::de::DeserializeOwned>(&self, path: &str) -> Result<T> {
        let url = format!("{}{}", self.base_url, path);
        
        let response = self.http
            .get(&url)
            .header(header::AUTHORIZATION, format!("Bearer {}", self.api_key))
            .header(header::CONTENT_TYPE, "application/json")
            .send()
            .await?;

        self.handle_response(response).await
    }

    /// Make an authenticated POST request.
    pub(crate) async fn post<T, B>(&self, path: &str, body: &B) -> Result<T>
    where
        T: serde::de::DeserializeOwned,
        B: serde::Serialize,
    {
        let url = format!("{}{}", self.base_url, path);
        
        let response = self.http
            .post(&url)
            .header(header::AUTHORIZATION, format!("Bearer {}", self.api_key))
            .header(header::CONTENT_TYPE, "application/json")
            .json(body)
            .send()
            .await?;

        self.handle_response(response).await
    }

    /// Make an authenticated DELETE request.
    pub(crate) async fn delete(&self, path: &str) -> Result<()> {
        let url = format!("{}{}", self.base_url, path);
        
        let response = self.http
            .delete(&url)
            .header(header::AUTHORIZATION, format!("Bearer {}", self.api_key))
            .send()
            .await?;

        if response.status().is_success() {
            Ok(())
        } else {
            Err(self.parse_error(response).await)
        }
    }

    async fn handle_response<T: serde::de::DeserializeOwned>(&self, response: reqwest::Response) -> Result<T> {
        if response.status().is_success() {
            let wrapper: ApiResponse<T> = response.json().await?;
            Ok(wrapper.data)
        } else {
            Err(self.parse_error(response).await)
        }
    }

    async fn parse_error(&self, response: reqwest::Response) -> BanhBaoRingError {
        let status = response.status().as_u16();
        let error: std::result::Result<ApiErrorResponse, _> = response.json().await;
        
        match error {
            Ok(e) => BanhBaoRingError::Api {
                code: e.error.code,
                message: e.error.message,
                status_code: status,
            },
            Err(_) => BanhBaoRingError::Api {
                code: "unknown".to_string(),
                message: "Unknown error".to_string(),
                status_code: status,
            },
        }
    }
}

#[derive(serde::Deserialize)]
struct ApiResponse<T> {
    data: T,
}

#[derive(serde::Deserialize)]
struct ApiErrorResponse {
    error: ApiError,
}

#[derive(serde::Deserialize)]
struct ApiError {
    code: String,
    message: String,
}
```

---

## 6. Types

**File:** `src/types.rs`

```rust
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// A cryptographic key.
#[derive(Debug, Clone, Deserialize)]
pub struct Key {
    pub id: Uuid,
    pub name: String,
    pub namespace_id: Uuid,
    pub public_key: String,
    pub address: String,
    pub algorithm: String,
    pub exportable: bool,
    pub metadata: Option<std::collections::HashMap<String, String>>,
    pub created_at: String,
}

/// Request to create a key.
#[derive(Debug, Clone, Serialize, Default)]
pub struct CreateKeyRequest {
    pub name: String,
    pub namespace_id: Uuid,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub algorithm: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub exportable: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<std::collections::HashMap<String, String>>,
}

/// Request to create multiple keys at once.
#[derive(Debug, Clone, Serialize)]
pub struct CreateBatchRequest {
    pub prefix: String,
    pub count: u32,
    pub namespace_id: Uuid,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub exportable: Option<bool>,
}

/// Response from a sign operation.
#[derive(Debug, Clone)]
pub struct SignResponse {
    pub key_id: Uuid,
    pub signature: Vec<u8>,
    pub public_key: String,
}

/// Request to sign data.
#[derive(Debug, Clone, Serialize)]
pub(crate) struct SignRequest {
    pub data: String, // base64
    pub prehashed: bool,
}

/// Request to sign multiple messages in batch.
#[derive(Debug, Clone)]
pub struct BatchSignRequest {
    pub requests: Vec<BatchSignItem>,
}

/// Single item in a batch sign request.
#[derive(Debug, Clone)]
pub struct BatchSignItem {
    pub key_id: Uuid,
    pub data: Vec<u8>,
    pub prehashed: bool,
}

/// An organization.
#[derive(Debug, Clone, Deserialize)]
pub struct Organization {
    pub id: Uuid,
    pub name: String,
    pub slug: String,
    pub plan: String,
    pub created_at: String,
}

/// An audit log entry.
#[derive(Debug, Clone, Deserialize)]
pub struct AuditLog {
    pub id: Uuid,
    pub event: String,
    pub actor_id: Option<Uuid>,
    pub actor_type: String,
    pub resource_type: Option<String>,
    pub resource_id: Option<Uuid>,
    pub metadata: Option<serde_json::Value>,
    pub created_at: String,
}
```

---

## 7. Keys Client

**File:** `src/keys.rs`

```rust
use crate::client::Client;
use crate::error::Result;
use crate::types::{Key, CreateKeyRequest, CreateBatchRequest};
use uuid::Uuid;

/// Client for key management operations.
pub struct KeysClient {
    client: Client,
}

impl KeysClient {
    pub(crate) fn new(client: Client) -> Self {
        Self { client }
    }

    /// Create a new key.
    pub async fn create(&self, request: CreateKeyRequest) -> Result<Key> {
        self.client.post("/v1/keys", &request).await
    }

    /// Create multiple keys in parallel.
    /// Optimized for Celestia's parallel worker pattern.
    ///
    /// # Example
    ///
    /// ```rust
    /// let keys = client.keys().create_batch(CreateBatchRequest {
    ///     prefix: "blob-worker".to_string(),
    ///     count: 4,
    ///     namespace_id,
    ///     exportable: None,
    /// }).await?;
    /// // Creates: blob-worker-1, blob-worker-2, blob-worker-3, blob-worker-4
    /// ```
    pub async fn create_batch(&self, request: CreateBatchRequest) -> Result<Vec<Key>> {
        #[derive(serde::Deserialize)]
        struct Response {
            keys: Vec<Key>,
        }
        
        let response: Response = self.client.post("/v1/keys/batch", &request).await?;
        Ok(response.keys)
    }

    /// Get a key by ID.
    pub async fn get(&self, key_id: &Uuid) -> Result<Key> {
        self.client.get(&format!("/v1/keys/{}", key_id)).await
    }

    /// List all keys, optionally filtered by namespace.
    pub async fn list(&self, namespace_id: Option<&Uuid>) -> Result<Vec<Key>> {
        let path = match namespace_id {
            Some(id) => format!("/v1/keys?namespace_id={}", id),
            None => "/v1/keys".to_string(),
        };
        self.client.get(&path).await
    }

    /// Delete a key.
    pub async fn delete(&self, key_id: &Uuid) -> Result<()> {
        self.client.delete(&format!("/v1/keys/{}", key_id)).await
    }
}
```

---

## 8. Sign Client

**File:** `src/sign.rs`

```rust
use crate::client::Client;
use crate::error::Result;
use crate::types::{SignResponse, BatchSignRequest, BatchSignItem};
use base64::{Engine as _, engine::general_purpose::STANDARD as BASE64};
use uuid::Uuid;

/// Client for signing operations.
pub struct SignClient {
    client: Client,
}

impl SignClient {
    pub(crate) fn new(client: Client) -> Self {
        Self { client }
    }

    /// Sign data with a key.
    pub async fn sign(&self, key_id: &Uuid, data: &[u8], prehashed: bool) -> Result<SignResponse> {
        #[derive(serde::Serialize)]
        struct Request {
            data: String,
            prehashed: bool,
        }

        #[derive(serde::Deserialize)]
        struct Response {
            signature: String,
            public_key: String,
        }

        let request = Request {
            data: BASE64.encode(data),
            prehashed,
        };

        let response: Response = self.client
            .post(&format!("/v1/keys/{}/sign", key_id), &request)
            .await?;

        let signature = BASE64.decode(&response.signature)
            .map_err(|e| crate::error::BanhBaoRingError::Decode(e.to_string()))?;

        Ok(SignResponse {
            key_id: *key_id,
            signature,
            public_key: response.public_key,
        })
    }

    /// Sign multiple messages in parallel.
    /// This is critical for Celestia's parallel blob submission.
    ///
    /// # Example
    ///
    /// ```rust
    /// let results = client.sign().sign_batch(BatchSignRequest {
    ///     requests: vec![
    ///         BatchSignItem { key_id: worker1, data: tx1.clone(), prehashed: false },
    ///         BatchSignItem { key_id: worker2, data: tx2.clone(), prehashed: false },
    ///         BatchSignItem { key_id: worker3, data: tx3.clone(), prehashed: false },
    ///         BatchSignItem { key_id: worker4, data: tx4.clone(), prehashed: false },
    ///     ],
    /// }).await?;
    /// // All 4 sign in parallel - completes in ~200ms, not 800ms!
    /// ```
    pub async fn sign_batch(&self, request: BatchSignRequest) -> Result<Vec<SignResponse>> {
        #[derive(serde::Serialize)]
        struct ApiRequest {
            requests: Vec<ApiRequestItem>,
        }

        #[derive(serde::Serialize)]
        struct ApiRequestItem {
            key_id: Uuid,
            data: String,
            prehashed: bool,
        }

        #[derive(serde::Deserialize)]
        struct ApiResponse {
            signatures: Vec<ApiSignature>,
        }

        #[derive(serde::Deserialize)]
        struct ApiSignature {
            key_id: Uuid,
            signature: String,
            public_key: String,
            error: Option<String>,
        }

        let api_request = ApiRequest {
            requests: request.requests.iter().map(|r| ApiRequestItem {
                key_id: r.key_id,
                data: BASE64.encode(&r.data),
                prehashed: r.prehashed,
            }).collect(),
        };

        let response: ApiResponse = self.client.post("/v1/sign/batch", &api_request).await?;

        let mut results = Vec::new();
        for sig in response.signatures {
            if sig.error.is_none() {
                let signature = BASE64.decode(&sig.signature)
                    .map_err(|e| crate::error::BanhBaoRingError::Decode(e.to_string()))?;
                results.push(SignResponse {
                    key_id: sig.key_id,
                    signature,
                    public_key: sig.public_key,
                });
            }
        }

        Ok(results)
    }
}
```

---

## 9. Error Types

**File:** `src/error.rs`

```rust
use thiserror::Error;

/// Result type for BanhBaoRing operations.
pub type Result<T> = std::result::Result<T, BanhBaoRingError>;

/// Errors that can occur when using the BanhBaoRing SDK.
#[derive(Error, Debug)]
pub enum BanhBaoRingError {
    /// API error from the BanhBaoRing service.
    #[error("API error ({status_code}): [{code}] {message}")]
    Api {
        code: String,
        message: String,
        status_code: u16,
    },

    /// HTTP request error.
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),

    /// Decoding error (base64, etc).
    #[error("Decode error: {0}")]
    Decode(String),

    /// Authentication error.
    #[error("Unauthorized: invalid API key")]
    Unauthorized,

    /// Rate limit exceeded.
    #[error("Rate limit exceeded")]
    RateLimited,

    /// Quota exceeded.
    #[error("Quota exceeded: {0}")]
    QuotaExceeded(String),
}
```

---

## 10. Example: Parallel Workers

**File:** `examples/parallel_workers.rs`

```rust
use banhbaoring::{Client, CreateBatchRequest, BatchSignRequest, BatchSignItem};
use uuid::Uuid;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let api_key = std::env::var("BANHBAORING_API_KEY")?;
    let namespace_id: Uuid = std::env::var("NAMESPACE_ID")?.parse()?;

    let client = Client::new(&api_key);

    // Create 4 worker keys for parallel blob submission
    println!("Creating worker keys...");
    let keys = client.keys().create_batch(CreateBatchRequest {
        prefix: "blob-worker".to_string(),
        count: 4,
        namespace_id,
        exportable: None,
    }).await?;

    println!("Created {} worker keys:", keys.len());
    for key in &keys {
        println!("  - {}: {}", key.name, key.address);
    }

    // Simulate parallel blob transactions
    let transactions: Vec<Vec<u8>> = vec![
        b"blob-tx-1".to_vec(),
        b"blob-tx-2".to_vec(),
        b"blob-tx-3".to_vec(),
        b"blob-tx-4".to_vec(),
    ];

    // Sign all 4 transactions in parallel with one API call
    println!("\nSigning transactions in batch...");
    let results = client.sign().sign_batch(BatchSignRequest {
        requests: keys.iter().zip(transactions.iter()).map(|(key, tx)| {
            BatchSignItem {
                key_id: key.id,
                data: tx.clone(),
                prehashed: false,
            }
        }).collect(),
    }).await?;

    println!("Signed transactions:");
    for (i, result) in results.iter().enumerate() {
        let sig_hex: String = result.signature.iter()
            .take(8)
            .map(|b| format!("{:02x}", b))
            .collect();
        println!("  - TX {}: sig={}...", i + 1, sig_hex);
    }

    Ok(())
}
```

---

## 11. Deliverables

| File | Description |
|------|-------------|
| `src/client.rs` | Main client |
| `src/keys.rs` | Key management |
| `src/sign.rs` | Signing with batch support |
| `src/types.rs` | Type definitions |
| `src/error.rs` | Error types |
| `examples/*.rs` | Usage examples |
| `tests/*.rs` | Integration tests |
| `README.md` | Documentation |

---

## 12. Success Criteria

- [ ] Client works with API key auth
- [ ] keys().create/get/list/delete work
- [ ] sign().sign works
- [ ] sign().sign_batch works in parallel
- [ ] keys().create_batch creates N keys
- [ ] Proper error types with thiserror
- [ ] Async/await throughout
- [ ] Examples compile and run
- [ ] Tests pass

---

## 13. Agent Prompt

```
You are Agent 11B - Rust SDK. Create the official Rust SDK for BanhBaoRing.

Read: doc/implementation/IMPL_11B_SDK_RUST.md

Deliverables:
1. Client with API key auth (reqwest + tokio)
2. KeysClient (CRUD + create_batch)
3. SignClient (sign + sign_batch)
4. Type definitions (serde)
5. Error types (thiserror)
6. Examples (basic, parallel_workers)
7. README

Crate: banhbaoring

Test: cargo test
```

