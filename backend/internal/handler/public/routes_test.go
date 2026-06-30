package public

import (
	"testing"

	"ppk/backend/internal/model"
)

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
		{feedbackType: model.ReviewFeedbackAccepted, wantStatus: model.ReviewStatusDeleted, wantUpdate: true},
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
