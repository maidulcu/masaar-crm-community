package handler

import (
	"errors"
	"strings"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// nowFn is time.Now; tests may replace it.
var nowFn = time.Now

var (
	validMaintenanceTypes = map[domain.MaintenanceType]bool{
		domain.MaintenancePlumbing: true, domain.MaintenanceElectrical: true, domain.MaintenanceHVAC: true,
		domain.MaintenanceFlooring: true, domain.MaintenancePainting: true, domain.MaintenanceStructural: true,
		domain.MaintenanceOther: true,
	}
	validPriorities = map[domain.TaskPriority]bool{
		domain.PriorityLow: true, domain.PriorityMedium: true, domain.PriorityHigh: true, domain.PriorityUrgent: true,
	}
	validMaintenanceStatuses = map[domain.MaintenanceStatus]bool{
		domain.MaintenancePending: true, domain.MaintenanceScheduled: true, domain.MaintenanceInProgress: true,
		domain.MaintenanceCompleted: true, domain.MaintenanceCancelled: true,
	}
	validInspectionTypes = map[domain.InspectionType]bool{
		domain.InspectionGeneral: true, domain.InspectionPreLease: true, domain.InspectionEndLease: true,
		domain.InspectionDamageAssessment: true, domain.InspectionSafety: true,
	}
	validInspectionStatuses = map[domain.InspectionStatus]bool{
		domain.InspectionScheduled: true, domain.InspectionInProgress: true,
		domain.InspectionCompleted: true, domain.InspectionCancelled: true,
	}
	validSeverities = map[domain.SeverityLevel]bool{
		"": true, domain.SeverityGreen: true, domain.SeverityYellow: true, domain.SeverityRed: true,
	}
	validPhotoStages = map[string]bool{"before": true, "during": true, "after": true}
)

// validateMaintenanceTask applies defaults and validates a task in place. Completion bookkeeping is
// done here too, so every path that sets a status keeps the completion date consistent.
func validateMaintenanceTask(t *domain.MaintenanceTask) error {
	t.Description = strings.TrimSpace(t.Description)
	if t.Description == "" {
		return errors.New("description is required")
	}
	if t.MaintenanceType == "" {
		t.MaintenanceType = domain.MaintenanceOther
	}
	if !validMaintenanceTypes[t.MaintenanceType] {
		return errors.New("maintenance_type must be one of: plumbing, electrical, hvac, flooring, painting, structural, other")
	}
	if t.Priority == "" {
		t.Priority = domain.PriorityMedium
	}
	if !validPriorities[t.Priority] {
		return errors.New("priority must be one of: low, medium, high, urgent")
	}
	if t.Status == "" {
		t.Status = domain.MaintenancePending
	}
	if !validMaintenanceStatuses[t.Status] {
		return errors.New("status must be one of: pending, scheduled, in_progress, completed, cancelled")
	}
	if t.ScheduledDate != nil && t.DueDate != nil && t.DueDate.Before(*t.ScheduledDate) {
		return errors.New("due_date must not be before scheduled_date")
	}
	for _, c := range []*float64{t.EstimatedCost, t.ActualCost} {
		if c != nil && badMoney(*c) {
			return errors.New("costs must not be negative")
		}
	}
	if tooLong(t.Description, 5000) || tooLong(t.ContractorName, 150) || tooLong(t.ContractorContact, 255) || tooLong(t.Notes, 10000) {
		return errors.New("a field is too long")
	}
	// A completed task has a completion date; one that is not (any more) must not keep it.
	if t.Status == domain.MaintenanceCompleted && t.CompletionDate == nil {
		now := nowFn()
		t.CompletionDate = &now
	}
	if t.Status != domain.MaintenanceCompleted {
		t.CompletionDate = nil
	}
	return nil
}

// validateInspection validates an inspection in place (status/severity/photos); the same
// completion bookkeeping applies as for tasks.
func validateInspection(i *domain.Inspection) error {
	if i.InspectionType == "" {
		i.InspectionType = string(domain.InspectionGeneral)
	}
	if !validInspectionTypes[domain.InspectionType(i.InspectionType)] {
		return errors.New("inspection_type must be one of: general, pre_lease, end_lease, damage_assessment, safety")
	}
	if i.Status == "" {
		i.Status = domain.InspectionScheduled
	}
	if !validInspectionStatuses[i.Status] {
		return errors.New("status must be one of: scheduled, in_progress, completed, cancelled")
	}
	if !validSeverities[i.SeverityLevel] {
		return errors.New("severity_level must be one of: green, yellow, red")
	}
	if i.ScheduledDate.IsZero() {
		return errors.New("scheduled_date is required")
	}
	if tooLong(i.Findings, 20000) {
		return errors.New("findings are too long")
	}
	if len(i.PhotosURLs) > 100 {
		return errors.New("too many photos")
	}
	for _, u := range i.PhotosURLs {
		if u == "" || tooLong(u, 500) || !safeLink(u) {
			return errors.New("photos_urls must be http(s) URLs")
		}
	}
	if len(i.ChecklistResults) > 500 {
		return errors.New("too many checklist results")
	}
	for _, r := range i.ChecklistResults {
		if r.Status != "" && r.Status != "pass" && r.Status != "fail" && r.Status != "n/a" {
			return errors.New("checklist result status must be pass, fail or n/a")
		}
		if tooLong(r.Notes, 2000) {
			return errors.New("checklist notes are too long")
		}
	}
	if i.Status == domain.InspectionCompleted && i.CompletedDate == nil {
		now := nowFn()
		i.CompletedDate = &now
	}
	if i.Status != domain.InspectionCompleted {
		i.CompletedDate = nil
	}
	return nil
}

// validateInspectionTemplate validates a template in place.
func validateInspectionTemplate(t *domain.InspectionTemplate) error {
	t.TemplateName = strings.TrimSpace(t.TemplateName)
	if t.TemplateName == "" {
		return errors.New("template_name is required")
	}
	if t.InspectionType == "" {
		t.InspectionType = domain.InspectionGeneral
	}
	if !validInspectionTypes[t.InspectionType] {
		return errors.New("inspection_type must be one of: general, pre_lease, end_lease, damage_assessment, safety")
	}
	if tooLong(t.TemplateName, 150) {
		return errors.New("template_name is too long")
	}
	if t.EstimatedDurationMinutes < 0 || t.EstimatedDurationMinutes > 24*60 {
		return errors.New("estimated_duration_minutes must be between 0 and 1440")
	}
	if len(t.ChecklistItems) > 200 {
		return errors.New("too many checklist items")
	}
	for i := range t.ChecklistItems {
		it := &t.ChecklistItems[i]
		it.Description = strings.TrimSpace(it.Description)
		if it.Description == "" || tooLong(it.Description, 500) || tooLong(it.ID, 100) {
			return errors.New("every checklist item needs a description (max 500 characters)")
		}
	}
	return nil
}
