// Package transport provides internal HTTP transport primitives for the
// Intelligence Cloud Go SDK.
//
// This file implements operation-context plumbing that the generated client
// layer uses to attach per-request metadata (operation ID, URL template,
// resource name) to the context before invoking Transport.RoundTrip.
package transport

import "context"

// operationContextKey is the unexported key type for storing OperationContext
// in a context.Context.
type operationContextKey struct{}

// OperationContext carries per-request metadata set by the generated client
// layer. The transport reads it for span naming and attributes.
type OperationContext struct {
	// OperationID is the OpenAPI operationId, e.g. "GetMe".
	OperationID string
	// URLTemplate is the parameterised path, e.g. "/api/v1/me".
	// It is cardinality-safe and suitable for use as a span attribute.
	URLTemplate string
	// Resource is the coarse-grained resource area, e.g. "me", "resellers".
	Resource string
}

// WithOperationContext returns a copy of ctx carrying the given OperationContext.
func WithOperationContext(ctx context.Context, oc OperationContext) context.Context {
	return context.WithValue(ctx, operationContextKey{}, oc)
}

// OperationContextFromContext extracts the OperationContext from ctx.
// The second return value is false when no OperationContext is present.
func OperationContextFromContext(ctx context.Context) (OperationContext, bool) {
	oc, ok := ctx.Value(operationContextKey{}).(OperationContext)
	return oc, ok
}
