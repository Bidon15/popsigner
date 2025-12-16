package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Bidon15/popsigner/control-plane/internal/database"
)

// RPCRateLimitConfig configures the RPC rate limiter.
type RPCRateLimitConfig struct {
	// RequestsPerSecond is the maximum number of requests allowed per second per address.
	RequestsPerSecond int
	// BurstSize is the maximum burst size (unused in sliding window, kept for compatibility).
	BurstSize int
}

// RPCRateLimit creates middleware that rate limits by Ethereum address.
// It inspects the JSON-RPC request to extract the 'from' address for eth_signTransaction,
// or the address parameter for eth_sign/personal_sign.
func RPCRateLimit(redis *database.Redis, cfg RPCRateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read the body to extract the address
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeRPCError(w, -32700, "Failed to read request")
				return
			}
			r.Body.Close()

			if len(body) == 0 {
				// Empty body, let the handler deal with it
				r.Body = io.NopCloser(bytes.NewReader(body))
				next.ServeHTTP(w, r)
				return
			}

			// Parse to extract address
			address := extractAddressFromRPCRequest(body)

			// If we couldn't extract an address, allow the request
			// (it will fail later with proper JSON-RPC error if invalid)
			if address != "" && redis != nil {
				// Check rate limit
				key := fmt.Sprintf("rpc_ratelimit:%s", strings.ToLower(address))
				allowed, err := checkSlidingWindowRateLimit(r.Context(), redis, key, cfg.RequestsPerSecond)
				if err != nil {
					// Log error but allow request (fail open)
					slog.Warn("Rate limit check failed",
						slog.String("error", err.Error()),
						slog.String("address", address),
					)
				} else if !allowed {
					slog.Info("Rate limit exceeded",
						slog.String("address", address),
						slog.Int("limit", cfg.RequestsPerSecond),
					)
					writeRPCError(w, -32029, "Rate limit exceeded")
					return
				}
			}

			// Reconstruct body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(body))

			next.ServeHTTP(w, r)
		})
	}
}

// extractAddressFromRPCRequest extracts the Ethereum address from a JSON-RPC request.
func extractAddressFromRPCRequest(body []byte) string {
	// Handle batch requests - extract from first request
	if len(body) > 0 && body[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(body, &batch); err != nil || len(batch) == 0 {
			return ""
		}
		return extractAddressFromSingleRequest(batch[0])
	}

	return extractAddressFromSingleRequest(body)
}

// extractAddressFromSingleRequest extracts address from a single JSON-RPC request.
func extractAddressFromSingleRequest(body []byte) string {
	var req struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}

	switch req.Method {
	case "eth_signTransaction":
		// Params: [{from: "0x...", ...}]
		if len(req.Params) > 0 {
			var txArgs struct {
				From string `json:"from"`
			}
			if err := json.Unmarshal(req.Params[0], &txArgs); err == nil {
				return txArgs.From
			}
		}

	case "eth_sign":
		// Params: [address, data]
		if len(req.Params) > 0 {
			var addr string
			if err := json.Unmarshal(req.Params[0], &addr); err == nil {
				return addr
			}
		}

	case "personal_sign":
		// Params: [data, address]
		if len(req.Params) > 1 {
			var addr string
			if err := json.Unmarshal(req.Params[1], &addr); err == nil {
				return addr
			}
		}

	case "eth_accounts":
		// No address to extract, use org-based rate limiting instead
		// Fall through to allow the request
		return ""
	}

	return ""
}

// checkSlidingWindowRateLimit checks if the request is within rate limits using sliding window.
func checkSlidingWindowRateLimit(ctx context.Context, redis *database.Redis, key string, requestsPerSecond int) (bool, error) {
	now := time.Now().UnixNano()
	windowStart := now - int64(time.Second) // 1 second window

	// Use Redis pipeline for atomic operations
	client := redis.Client()
	pipe := client.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// Count entries in current window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request with timestamp as score
	pipe.ZAdd(ctx, key, redisZ{Score: float64(now), Member: fmt.Sprintf("%d", now)})

	// Set expiry on the key (2 seconds to handle edge cases)
	pipe.Expire(ctx, key, 2*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count := countCmd.Val()
	return count < int64(requestsPerSecond), nil
}

// redisZ is a type alias for Redis sorted set member.
type redisZ = struct {
	Score  float64
	Member interface{}
}

// writeRPCError writes a JSON-RPC error response.
func writeRPCError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	if code == -32029 {
		w.WriteHeader(http.StatusTooManyRequests)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	resp := fmt.Sprintf(`{"jsonrpc":"2.0","error":{"code":%d,"message":"%s"},"id":null}`, code, message)
	w.Write([]byte(resp))
}

