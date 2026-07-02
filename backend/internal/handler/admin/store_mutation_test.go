package admin

import (
	"testing"

	"ppk/backend/internal/model"
)

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

func TestDeriveNFCCardStatusRequiresRealBoundTag(t *testing.T) {
	store := model.Store{ID: 1, UUID: "store-uuid", Status: model.StatusEnabled}

	status := deriveNFCCardStatus(store, 0, 0, 0)
	if status.PrimaryStatus != "unwritten" || status.RouteStatus != "no_bound_tag" {
		t.Fatalf("empty tag status got %#v, want unwritten/no_bound_tag", status)
	}

	status = deriveNFCCardStatus(store, 1, 1, 0)
	if status.PrimaryStatus != "usable" || status.RouteStatus != "ok" || status.WrittenCount != 1 {
		t.Fatalf("bound tag status got %#v, want usable/ok with written count", status)
	}
}

func TestDeriveNFCCardStatusMarksInactiveStoreUnusable(t *testing.T) {
	store := model.Store{ID: 1, UUID: "store-uuid", Status: model.StatusDisabled}

	status := deriveNFCCardStatus(store, 1, 1, 0)
	if status.PrimaryStatus != "unusable" || status.RouteStatus != "store_inactive" {
		t.Fatalf("inactive store status got %#v, want unusable/store_inactive", status)
	}
}
