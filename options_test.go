package intelligencecloud

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// [A-13] CallOption / ListOption helper constructors
// ---------------------------------------------------------------------------

func TestWithIdempotencyKey(t *testing.T) {
	cfg := &callConfig{}
	opt := WithIdempotencyKey("key-123")
	opt.applyCall(cfg)
	if cfg.idempotencyKey != "key-123" {
		t.Errorf("idempotencyKey = %q, want %q", cfg.idempotencyKey, "key-123")
	}
}

func TestWithRequestTimeout(t *testing.T) {
	cfg := &callConfig{}
	opt := WithRequestTimeout(5 * time.Second)
	opt.applyCall(cfg)
	if cfg.requestTimeout != 5*time.Second {
		t.Errorf("requestTimeout = %v, want 5s", cfg.requestTimeout)
	}
}

func TestWithExtraHeader(t *testing.T) {
	cfg := &callConfig{}
	opt := WithExtraHeader("X-Custom", "value")
	opt.applyCall(cfg)
	if len(cfg.extraHeaders) != 1 {
		t.Fatalf("extraHeaders len = %d, want 1", len(cfg.extraHeaders))
	}
	if cfg.extraHeaders["X-Custom"] != "value" {
		t.Errorf("extraHeaders[X-Custom] = %q, want %q", cfg.extraHeaders["X-Custom"], "value")
	}
}

func TestWithExtraHeader_MultipleHeaders(t *testing.T) {
	cfg := &callConfig{}
	WithExtraHeader("X-One", "1").applyCall(cfg)
	WithExtraHeader("X-Two", "2").applyCall(cfg)
	if len(cfg.extraHeaders) != 2 {
		t.Fatalf("extraHeaders len = %d, want 2", len(cfg.extraHeaders))
	}
}

func TestWithExtraHeader_RejectsAuthorization(t *testing.T) {
	// Must reject Authorization header (case-insensitive).
	cases := []string{"Authorization", "authorization", "AUTHORIZATION", "AuThOrIzAtIoN"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Error("expected panic for Authorization header, got none")
				}
				msg, ok := r.(string)
				if !ok {
					t.Errorf("panic value is not string: %v", r)
					return
				}
				if !strings.Contains(strings.ToLower(msg), "authorization") {
					t.Errorf("panic message %q does not mention authorization", msg)
				}
			}()
			WithExtraHeader(name, "bearer secret")
		})
	}
}

func TestWithPageSize(t *testing.T) {
	cfg := &listConfig{}
	opt := WithPageSize(50)
	opt.applyList(cfg)
	if cfg.pageSize != 50 {
		t.Errorf("pageSize = %d, want 50", cfg.pageSize)
	}
}

func TestWithPageSize_RejectsZero(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for pageSize=0, got none")
		}
	}()
	WithPageSize(0)
}

func TestWithPageSize_RejectsNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for pageSize=-1, got none")
		}
	}()
	WithPageSize(-1)
}

func TestWithMaxItems(t *testing.T) {
	cfg := &listConfig{}
	opt := WithMaxItems(100)
	opt.applyList(cfg)
	if cfg.maxItems != 100 {
		t.Errorf("maxItems = %d, want 100", cfg.maxItems)
	}
}

func TestWithMaxItems_ZeroMeansUnlimited(t *testing.T) {
	cfg := &listConfig{}
	opt := WithMaxItems(0)
	opt.applyList(cfg)
	if cfg.maxItems != 0 {
		t.Errorf("maxItems = %d, want 0 (unlimited)", cfg.maxItems)
	}
}

func TestWithMaxItems_RejectsNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for maxItems=-1, got none")
		}
	}()
	WithMaxItems(-1)
}

func TestWithFilter(t *testing.T) {
	cfg := &listConfig{}
	WithFilter("status", "active").applyList(cfg)
	WithFilter("type", "reseller").applyList(cfg)
	if len(cfg.filters) != 2 {
		t.Fatalf("filters len = %d, want 2", len(cfg.filters))
	}
	if cfg.filters["status"] != "active" {
		t.Errorf("filters[status] = %q, want %q", cfg.filters["status"], "active")
	}
	if cfg.filters["type"] != "reseller" {
		t.Errorf("filters[type] = %q, want %q", cfg.filters["type"], "reseller")
	}
}

func TestCallOption_InterfaceSatisfied(t *testing.T) {
	// Verify all call-option helpers return CallOption.
	var _ CallOption = WithIdempotencyKey("k")
	var _ CallOption = WithRequestTimeout(time.Second)
	var _ CallOption = WithExtraHeader("X-Foo", "bar")
}

func TestListOption_InterfaceSatisfied(t *testing.T) {
	// Verify all list-option helpers return ListOption.
	var _ ListOption = WithPageSize(1)
	var _ ListOption = WithMaxItems(0)
	var _ ListOption = WithFilter("k", "v")
}
