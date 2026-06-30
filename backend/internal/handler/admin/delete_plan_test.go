package admin

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStoreDeletionPlanClearsChildrenBeforeDeletingStore(t *testing.T) {
	steps := storeDeletionPlan(false)

	want := []string{
		"unbind_nfc_tags",
		"delete_review_feedbacks",
		"delete_review_display_logs",
		"delete_review_generation_tasks",
		"delete_review_items",
		"delete_store_platform_links",
		"delete_store_images",
		"delete_store_keywords",
		"delete_store",
	}

	if len(steps) != len(want) {
		t.Fatalf("got %d steps, want %d: %#v", len(steps), len(want), steps)
	}
	for i := range want {
		if steps[i] != want[i] {
			t.Fatalf("step %d got %q, want %q; full plan: %#v", i, steps[i], want[i], steps)
		}
	}
}

func TestStoreDeletionPlanDeletesMerchantAfterStoreWhenRequested(t *testing.T) {
	steps := storeDeletionPlan(true)
	if len(steps) == 0 {
		t.Fatal("expected deletion steps")
	}
	if got := steps[len(steps)-1]; got != "delete_merchant" {
		t.Fatalf("last step got %q, want delete_merchant; full plan: %#v", got, steps)
	}
}

func TestAdminRegisterExposesDeleteRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	h := &Handler{}
	h.Register(r.Group("/api"))

	routes := map[string]bool{}
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	for _, route := range []string{
		"DELETE /api/admin/merchants/:id",
		"DELETE /api/admin/stores/:id",
		"PUT /api/admin/stores/:id",
	} {
		if !routes[route] {
			t.Fatalf("route %s not registered; routes=%v", route, routes)
		}
	}
}
