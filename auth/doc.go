// Package auth defines the credential-provider model for the Intelligence
// Cloud Go SDK. Consumers supply a [CredentialProvider] implementation to
// the SDK client at construction time; the client calls
// [CredentialProvider.Token] before each outbound API request to obtain a
// bearer token.
//
// A [Token] carries an access token string and an optional expiry time.
// A zero-value ExpiresAt signals that the token never expires.
//
// # Built-in adapters
//
// Two adapters ship in v1:
//
//   - [StaticToken] wraps a fixed bearer-token string. It always returns
//     the same token with a zero expiry (never expires). Suitable for
//     service-to-service calls where the token is injected from a secret
//     store at process start.
//
//   - [RefreshFunc] wraps a caller-supplied callback that fetches a fresh
//     token on demand. The adapter maintains an in-memory cache with a
//     30-second safety margin before the reported expiry. Concurrent
//     callers observe single-flight semantics: at most one invocation of
//     the callback runs at a time.
//
// # Companion modules
//
// The core auth package intentionally does not bundle OAuth, MSAL,
// device-code, or managed-identity flows. Those concerns belong to
// separately versioned companion modules (e.g. auth/azure) that may be
// added in future releases without a breaking change to the core SDK.
package auth
