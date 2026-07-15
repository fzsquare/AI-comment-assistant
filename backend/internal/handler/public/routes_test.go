package public

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"ppk/backend/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type dryRunConn struct{}

func (dryRunConn) PrepareContext(context.Context, string) (*sql.Stmt, error) { return nil, nil }
func (dryRunConn) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (dryRunConn) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (dryRunConn) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return &sql.Row{}
}

func TestFeedbackTypeForAction(t *testing.T) {
	tests := []struct {
		action      string
		wantType    string
		wantTracked bool
	}{
		{action: "review_copy", wantType: model.ReviewFeedbackAccepted, wantTracked: true},
		{action: "platform_link_click", wantType: model.ReviewFeedbackAccepted, wantTracked: true},
		{action: "review_reject", wantType: model.ReviewFeedbackRejected, wantTracked: true},
		{action: "review_switch", wantTracked: false},
		{action: "page_view", wantTracked: false},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			gotType, gotTracked := feedbackTypeForAction(tt.action)
			if gotTracked != tt.wantTracked {
				t.Fatalf("tracked got %v, want %v", gotTracked, tt.wantTracked)
			}
			if gotType != tt.wantType {
				t.Fatalf("feedback type got %q, want %q", gotType, tt.wantType)
			}
		})
	}
}

func TestReviewStatusForFeedback(t *testing.T) {
	tests := []struct {
		feedbackType string
		wantStatus   string
		wantUpdate   bool
	}{
		{feedbackType: model.ReviewFeedbackAccepted, wantStatus: model.ReviewStatusUsed, wantUpdate: true},
		{feedbackType: model.ReviewFeedbackRejected, wantStatus: model.ReviewStatusDisabled, wantUpdate: true},
		{feedbackType: "unknown", wantUpdate: false},
	}

	for _, tt := range tests {
		t.Run(tt.feedbackType, func(t *testing.T) {
			gotStatus, gotUpdate := reviewStatusForFeedback(tt.feedbackType)
			if gotUpdate != tt.wantUpdate {
				t.Fatalf("update got %v, want %v", gotUpdate, tt.wantUpdate)
			}
			if gotStatus != tt.wantStatus {
				t.Fatalf("status got %q, want %q", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestValidateLandingEventShapeRejectsUnknownAndIncompleteActions(t *testing.T) {
	if err := validateLandingEventShape(eventRequest{SessionID: "signed", ActionType: "made_up"}); err == nil {
		t.Fatal("unknown action should be rejected")
	}
	if err := validateLandingEventShape(eventRequest{SessionID: "signed", ActionType: "review_copy", PlatformCode: "meituan"}); err == nil {
		t.Fatal("review_copy without review item should be rejected")
	}
	if err := validateLandingEventShape(eventRequest{SessionID: "signed", ActionType: "page_view"}); err != nil {
		t.Fatalf("page_view should be accepted: %v", err)
	}
}

func TestFeedbackInsertIsIdempotentAndReviewReferenceIsSessionBound(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      dryRunConn{},
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry-run mysql: %v", err)
	}

	feedback := model.ReviewFeedback{StoreID: 7, ReviewItemID: 11, SessionID: "signed-session", FeedbackType: model.ReviewFeedbackAccepted}
	insert := insertFeedbackIfAbsent(db, &feedback)
	if insert.Error != nil {
		t.Fatalf("build idempotent feedback insert: %v", insert.Error)
	}
	if sql := insert.Statement.SQL.String(); !strings.Contains(sql, "ON DUPLICATE KEY UPDATE") {
		t.Fatalf("feedback insert must ignore concurrent unique-key duplicates: %s", sql)
	}

	var count int64
	reference := dispatchedReviewReferenceQuery(db, model.Store{ID: 7}, eventRequest{
		ReviewItemID: 11,
		SessionID:    "signed-session",
	}, "meituan").Count(&count)
	if reference.Error != nil {
		t.Fatalf("build dispatched review query: %v", reference.Error)
	}
	if sql := reference.Statement.SQL.String(); !strings.Contains(sql, "dispatched_session_id") || !strings.Contains(sql, "platform_style") {
		t.Fatalf("review reference must bind platform and dispatched session: %s", sql)
	}
}
