//! Error types for the POPSigner SDK.
//!
//! This module provides a unified error type for all SDK operations,
//! with rich error information from the API.

use thiserror::Error;

/// Result type for POPSigner operations.
pub type Result<T> = std::result::Result<T, POPSignerError>;

/// Errors that can occur when using the POPSigner SDK.
#[derive(Error, Debug)]
pub enum POPSignerError {
    /// API error from the POPSigner service.
    #[error("API error ({status_code}): [{code}] {message}")]
    Api {
        /// Error code from the API.
        code: String,
        /// Human-readable error message.
        message: String,
        /// HTTP status code.
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

    /// Key not found.
    #[error("Key not found: {0}")]
    KeyNotFound(String),

    /// Namespace not found.
    #[error("Namespace not found: {0}")]
    NamespaceNotFound(String),

    /// Organization not found.
    #[error("Organization not found: {0}")]
    OrgNotFound(String),

    /// Invalid request.
    #[error("Invalid request: {0}")]
    InvalidRequest(String),

    /// Signing error.
    #[error("Signing error: {0}")]
    SigningError(String),

    /// Batch operation partial failure.
    #[error("Batch operation had {failed} failures out of {total} requests")]
    BatchPartialFailure {
        /// Number of failed operations.
        failed: usize,
        /// Total number of operations.
        total: usize,
    },
}

impl POPSignerError {
    /// Returns true if this is a retryable error.
    pub fn is_retryable(&self) -> bool {
        match self {
            POPSignerError::RateLimited => true,
            POPSignerError::Http(_) => true,
            POPSignerError::Api { status_code, .. } => *status_code >= 500,
            _ => false,
        }
    }

    /// Returns true if this is an authentication error.
    pub fn is_auth_error(&self) -> bool {
        matches!(
            self,
            POPSignerError::Unauthorized
                | POPSignerError::Api { status_code: 401, .. }
                | POPSignerError::Api { status_code: 403, .. }
        )
    }

    /// Returns the HTTP status code if available.
    pub fn status_code(&self) -> Option<u16> {
        match self {
            POPSignerError::Api { status_code, .. } => Some(*status_code),
            POPSignerError::Unauthorized => Some(401),
            POPSignerError::RateLimited => Some(429),
            _ => None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_display() {
        let err = POPSignerError::Api {
            code: "key_not_found".to_string(),
            message: "Key does not exist".to_string(),
            status_code: 404,
        };
        assert_eq!(
            err.to_string(),
            "API error (404): [key_not_found] Key does not exist"
        );
    }

    #[test]
    fn test_is_retryable() {
        let rate_limited = POPSignerError::RateLimited;
        assert!(rate_limited.is_retryable());

        let server_error = POPSignerError::Api {
            code: "internal".to_string(),
            message: "Internal server error".to_string(),
            status_code: 500,
        };
        assert!(server_error.is_retryable());

        let not_found = POPSignerError::Api {
            code: "not_found".to_string(),
            message: "Not found".to_string(),
            status_code: 404,
        };
        assert!(!not_found.is_retryable());
    }

    #[test]
    fn test_is_auth_error() {
        let unauthorized = POPSignerError::Unauthorized;
        assert!(unauthorized.is_auth_error());

        let api_401 = POPSignerError::Api {
            code: "unauthorized".to_string(),
            message: "Invalid API key".to_string(),
            status_code: 401,
        };
        assert!(api_401.is_auth_error());
    }

    #[test]
    fn test_status_code() {
        let err = POPSignerError::Api {
            code: "test".to_string(),
            message: "Test".to_string(),
            status_code: 500,
        };
        assert_eq!(err.status_code(), Some(500));

        let decode_err = POPSignerError::Decode("bad base64".to_string());
        assert_eq!(decode_err.status_code(), None);
    }
}
