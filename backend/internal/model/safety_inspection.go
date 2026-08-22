package model

import "time"

const (
	InspectionStateScheduled  = "scheduled"
	InspectionStateInProgress = "in_progress"
	InspectionStateCompleted  = "completed"
	InspectionStateFailed     = "failed"
)

// SafetyInspection 安全检查实体。
type SafetyInspection struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string    `gorm:"size:200;not null" json:"name"`
	InspectionType string    `gorm:"size:30;not null;default:routine" json:"inspection_type"`
	Area           string    `gorm:"size:100;not null;default:''" json:"area"`
	InspectionDate time.Time `json:"inspection_date"`
	InspectorID    uint64    `gorm:"not null" json:"inspector_id"`
	TotalScore     int       `gorm:"not null;default:0" json:"total_score"`
	Status         string    `gorm:"size:30;not null;default:scheduled;index" json:"status"`
	IssueCount     int       `gorm:"not null;default:0" json:"issue_count"`
	PassedCount    int       `gorm:"not null;default:0" json:"passed_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// TableName 指定表名。
func (SafetyInspection) TableName() string { return "safety_inspections" }

// CanInspectionTransition reports whether an execution state change is legal.
func CanInspectionTransition(from, to string) bool {
	switch from {
	case InspectionStateScheduled:
		return to == InspectionStateInProgress || to == InspectionStateCompleted || to == InspectionStateFailed
	case InspectionStateInProgress:
		return to == InspectionStateInProgress
	case InspectionStateCompleted, InspectionStateFailed:
		return to == InspectionStateInProgress
	default:
		return false
	}
}

// InspectionReportVisible reports whether a report may be presented.
func InspectionReportVisible(status string) bool {
	return status == InspectionStateFailed
}
