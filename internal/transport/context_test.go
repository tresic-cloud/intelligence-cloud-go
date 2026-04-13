package transport

import (
	"context"
	"testing"
)

func TestWithOperationContext_RoundTrip(t *testing.T) {
	t.Parallel()

	oc := OperationContext{
		OperationID: "GetMe",
		URLTemplate: "/api/v1/me",
		Resource:    "me",
	}

	ctx := WithOperationContext(context.Background(), oc)
	got, ok := OperationContextFromContext(ctx)
	if !ok {
		t.Fatal("OperationContextFromContext returned false")
	}
	if got.OperationID != oc.OperationID {
		t.Errorf("OperationID = %q, want %q", got.OperationID, oc.OperationID)
	}
	if got.URLTemplate != oc.URLTemplate {
		t.Errorf("URLTemplate = %q, want %q", got.URLTemplate, oc.URLTemplate)
	}
	if got.Resource != oc.Resource {
		t.Errorf("Resource = %q, want %q", got.Resource, oc.Resource)
	}
}

func TestOperationContextFromContext_Missing(t *testing.T) {
	t.Parallel()

	_, ok := OperationContextFromContext(context.Background())
	if ok {
		t.Error("expected false for background context")
	}
}

func TestWithOperationContext_Overwrite(t *testing.T) {
	t.Parallel()

	oc1 := OperationContext{OperationID: "First"}
	oc2 := OperationContext{OperationID: "Second"}

	ctx := WithOperationContext(context.Background(), oc1)
	ctx = WithOperationContext(ctx, oc2)

	got, ok := OperationContextFromContext(ctx)
	if !ok {
		t.Fatal("OperationContextFromContext returned false")
	}
	if got.OperationID != "Second" {
		t.Errorf("OperationID = %q, want %q", got.OperationID, "Second")
	}
}

func TestOperationContextFromContext_ZeroValue(t *testing.T) {
	t.Parallel()

	ctx := WithOperationContext(context.Background(), OperationContext{})
	got, ok := OperationContextFromContext(ctx)
	if !ok {
		t.Fatal("expected true for zero-value OperationContext")
	}
	if got.OperationID != "" {
		t.Errorf("OperationID = %q, want empty", got.OperationID)
	}
}
