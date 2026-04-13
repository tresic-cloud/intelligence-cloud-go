package generated_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// TestGenerated_Compiles constructs representative request and response types
// from the generated package to prove they compile and their fields are
// accessible. This catches silent codegen breakage where files parse but
// types are empty or mis-shaped.
func TestGenerated_Compiles(t *testing.T) {
	// Request type: CreateResellerRequest
	req := generated.CreateResellerRequest{
		ResellerName:        "Test Reseller",
		PrimaryContactName:  "Jane Doe",
		PrimaryContactEmail: "jane@example.com",
		PrimaryContactPhone: "+15551234567",
	}
	if req.ResellerName != "Test Reseller" {
		t.Errorf("CreateResellerRequest.ResellerName = %q, want %q", req.ResellerName, "Test Reseller")
	}

	// Response type: Reseller
	name := "Acme Corp"
	active := true
	reseller := generated.Reseller{
		ResellerName: &name,
		IsActive:     &active,
	}
	if reseller.ResellerName == nil || *reseller.ResellerName != "Acme Corp" {
		t.Error("Reseller.ResellerName not set correctly")
	}

	// Response type: UserProfile
	profile := generated.UserProfile{
		Id:          "550e8400-e29b-41d4-a716-446655440000",
		Email:       "admin@tresic.com",
		FirstName:   "John",
		LastName:    "Doe",
		DisplayName: "John Doe",
		Role:        "super_admin",
		Status:      generated.UserProfileStatusActive,
		UserType:    generated.UserProfileUserTypeTresicAdmin,
		Scopes:      []string{"admin:*"},
	}
	if profile.UserType != generated.UserProfileUserTypeTresicAdmin {
		t.Errorf("UserProfile.UserType = %q, want %q", profile.UserType, generated.UserProfileUserTypeTresicAdmin)
	}

	// LoginRequest
	login := generated.LoginRequest{
		Email:    "admin@tresic.com",
		Password: "test",
	}
	if login.Email != "admin@tresic.com" {
		t.Errorf("LoginRequest.Email = %q, want %q", login.Email, "admin@tresic.com")
	}

	// Product
	product := generated.Product{
		Key:  "insights-hub",
		Name: "Insights Hub",
	}
	if product.Key != "insights-hub" {
		t.Errorf("Product.Key = %q, want %q", product.Key, "insights-hub")
	}

	// NotificationPreferences
	boolTrue := true
	prefs := generated.NotificationPreferences{
		EmailNotifications: &boolTrue,
	}
	if prefs.EmailNotifications == nil || !*prefs.EmailNotifications {
		t.Error("NotificationPreferences.EmailNotifications not set correctly")
	}
}

// TestGenerated_HasClient verifies that the generated Client type exists,
// can be constructed, and exposes the expected method surface. This test
// uses reflection to check that key methods are present on the client.
func TestGenerated_HasClient(t *testing.T) {
	client, err := generated.NewClient("http://localhost:8080")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	clientType := reflect.TypeOf(client)

	// Every operationId should map to a method on *Client.
	expectedMethods := []string{
		"Login",
		"GetMe",
		"CreateAvatarUploadURL",
		"ConfirmAvatar",
		"DeleteAvatar",
		"GetNotificationPreferences",
		"PatchNotificationPreferences",
		"GetDailyRecapPreferences",
		"PatchDailyRecapPreferences",
		"CreateReseller",
		"ListResellers",
		"GetReseller",
		"UpdateReseller",
		"PatchReseller",
		"DeactivateReseller",
		"ReactivateReseller",
		"ListResellerAuditLogs",
		"ListLocationConnectors",
		"ListCompanyConnectors",
		"PatchCompanyUser",
		"ListProducts",
	}

	for _, name := range expectedMethods {
		_, ok := clientType.MethodByName(name)
		if !ok {
			t.Errorf("Client is missing method %s", name)
		}
	}
}

// TestGenerated_ClientMethodSignatures verifies that a few representative
// client methods have the expected call signature (context as first arg,
// returns *http.Response and error).
func TestGenerated_ClientMethodSignatures(t *testing.T) {
	client, err := generated.NewClient("http://localhost:8080")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// GetMe: func(context.Context) (*http.Response, error)
	ctx := context.Background()
	_ = ctx
	_ = client

	// Verify the method is callable with correct types at compile time.
	var _ func(context.Context) (*http.Response, error) = client.GetMe
	var _ func(context.Context) (*http.Response, error) = client.ListProducts
	var _ func(context.Context) (*http.Response, error) = client.GetNotificationPreferences
	var _ func(context.Context) (*http.Response, error) = client.GetDailyRecapPreferences
	var _ func(context.Context) (*http.Response, error) = client.DeleteAvatar
}

// TestGenerated_RequestConstructors verifies that the New*Request constructor
// functions exist and produce valid *http.Request values.
func TestGenerated_RequestConstructors(t *testing.T) {
	server := "http://localhost:8080"

	t.Run("NewGetMeRequest", func(t *testing.T) {
		req, err := generated.NewGetMeRequest(server)
		if err != nil {
			t.Fatalf("NewGetMeRequest() error = %v", err)
		}
		if req.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", req.Method)
		}
		if !contains(req.URL.Path, "/api/v1/me") {
			t.Errorf("path = %s, want to contain /api/v1/me", req.URL.Path)
		}
	})

	t.Run("NewListResellersRequest", func(t *testing.T) {
		req, err := generated.NewListResellersRequest(server, nil)
		if err != nil {
			t.Fatalf("NewListResellersRequest() error = %v", err)
		}
		if req.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", req.Method)
		}
	})

	t.Run("NewListProductsRequest", func(t *testing.T) {
		req, err := generated.NewListProductsRequest(server)
		if err != nil {
			t.Fatalf("NewListProductsRequest() error = %v", err)
		}
		if req.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", req.Method)
		}
	})

	t.Run("NewLoginRequest", func(t *testing.T) {
		body := generated.LoginJSONRequestBody{
			Email:    "test@example.com",
			Password: "secret",
		}
		req, err := generated.NewLoginRequest(server, body)
		if err != nil {
			t.Fatalf("NewLoginRequest() error = %v", err)
		}
		if req.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", req.Method)
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", req.Header.Get("Content-Type"))
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
