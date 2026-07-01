package admin

import "testing"

func TestNormalizeStoreMutationRequiresPasswordWhenCreating(t *testing.T) {
	req := storeMutationRequest{
		Account:   "merchant-new",
		TypeID:    1,
		StoreName: "新店",
	}

	if err := normalizeStoreMutationRequest(&req, true); err == nil {
		t.Fatal("expected missing password to fail when creating store")
	}
}

func TestNormalizeStoreMutationAllowsBlankPasswordWhenEditing(t *testing.T) {
	req := storeMutationRequest{
		Account:              "  merchant-new  ",
		TypeID:               1,
		StoreName:            "  新店  ",
		PrimaryPlatformStyle: "",
	}

	if err := normalizeStoreMutationRequest(&req, false); err != nil {
		t.Fatalf("expected blank password to be allowed when editing, got %v", err)
	}
	if req.Account != "merchant-new" {
		t.Fatalf("account got %q, want trimmed merchant-new", req.Account)
	}
	if req.StoreName != "新店" {
		t.Fatalf("store name got %q, want trimmed 新店", req.StoreName)
	}
	if req.PrimaryPlatformStyle != "dianping" {
		t.Fatalf("platform got %q, want default dianping", req.PrimaryPlatformStyle)
	}
}

func TestNormalizeStoreMutationAllowsBlankPlatformURL(t *testing.T) {
	req := storeMutationRequest{
		Account:   "merchant-new",
		Password:  "123456",
		TypeID:    1,
		StoreName: "新店",
	}

	if err := normalizeStoreMutationRequest(&req, true); err != nil {
		t.Fatalf("expected blank platform url to be allowed, got %v", err)
	}
}

func TestNormalizeStoreMutationRejectsUnsupportedPlatformURL(t *testing.T) {
	req := storeMutationRequest{
		Account:     "merchant-new",
		Password:    "123456",
		TypeID:      1,
		StoreName:   "新店",
		PlatformURL: "javascript:alert(1)",
	}

	if err := normalizeStoreMutationRequest(&req, true); err == nil {
		t.Fatal("expected unsupported platform url to fail")
	}
}
