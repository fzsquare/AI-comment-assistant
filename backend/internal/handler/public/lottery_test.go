package public

import (
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestLotteryEligibilityUsesSavedReviewCopyEvent(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      dryRunConn{},
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry-run mysql: %v", err)
	}

	var count int64
	query := lotteryEligibilityQuery(db, 7, "signed-session").Count(&count)
	if query.Error != nil {
		t.Fatalf("build lottery eligibility query: %v", query.Error)
	}

	foundReviewCopy := false
	for _, value := range query.Statement.Vars {
		if value == "platform_link_click" {
			t.Fatal("lottery eligibility must not depend on leaving the H5 page")
		}
		if value == "review_copy" {
			foundReviewCopy = true
		}
	}
	if !foundReviewCopy {
		t.Fatalf("lottery eligibility must require review_copy, vars=%v", query.Statement.Vars)
	}
}
