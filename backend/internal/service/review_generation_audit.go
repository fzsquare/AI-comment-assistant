package service

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"ppk/backend/internal/model"
)

type generationAuditInput struct {
	Task       model.ReviewGenerationTask
	Stage      string
	Level      string
	Status     string
	Message    string
	Detail     map[string]interface{}
	Duration   time.Duration
	HTTPStatus int
}

type auditEndpointProvider interface {
	AuditEndpoint() string
}

func (s *ReviewPoolService) recordGenerationAudit(input generationAuditInput) {
	if s == nil || s.DB == nil || input.Task.ID == 0 {
		return
	}
	detail := ""
	if len(input.Detail) > 0 {
		if data, err := json.Marshal(input.Detail); err == nil {
			detail = string(data)
		}
	}
	row := model.ReviewGenerationAuditLog{
		TaskID:                 input.Task.ID,
		StoreID:                input.Task.StoreID,
		PlatformStyle:          input.Task.PlatformStyle,
		TriggerType:            input.Task.TriggerType,
		Stage:                  truncateForAudit(input.Stage, 64),
		Level:                  truncateForAudit(defaultAuditValue(input.Level, "info"), 16),
		Status:                 truncateForAudit(defaultAuditValue(input.Status, input.Task.Status), 32),
		Message:                truncateForAudit(input.Message, 512),
		Detail:                 detail,
		AgentEndpoint:          truncateForAudit(s.generationAuditEndpoint(), 255),
		HTTPStatus:             input.HTTPStatus,
		DurationMS:             input.Duration.Milliseconds(),
		TargetCount:            input.Task.TargetCount,
		GeneratedRawCount:      input.Task.GeneratedRawCount,
		InsertedRowCount:       input.Task.InsertedRowCount,
		DuplicateFilteredCount: input.Task.DuplicateFilteredCount,
	}
	if err := s.DB.Create(&row).Error; err != nil {
		log.Printf("review_generation_audit_write_failed task_id=%d stage=%s err=%v", input.Task.ID, input.Stage, err)
		return
	}
	log.Printf(
		"review_generation_audit task_id=%d store_id=%d platform=%s stage=%s level=%s status=%s duration_ms=%d message=%s",
		row.TaskID,
		row.StoreID,
		row.PlatformStyle,
		row.Stage,
		row.Level,
		row.Status,
		row.DurationMS,
		row.Message,
	)
}

func (s *ReviewPoolService) generationAuditEndpoint() string {
	if s == nil || s.Generator == nil {
		return ""
	}
	if provider, ok := s.Generator.(auditEndpointProvider); ok {
		return provider.AuditEndpoint()
	}
	return ""
}

func auditHTTPStatus(err error) int {
	var agentErr *AgentHTTPError
	if errors.As(err, &agentErr) {
		return agentErr.StatusCode
	}
	return 0
}

func truncateForAudit(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func defaultAuditValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
