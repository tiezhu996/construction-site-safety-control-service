package model

import (
	"strings"
	"time"
)

// SafetyTraining 安全培训实体。
type SafetyTraining struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Topic            string    `gorm:"size:200;not null" json:"topic"`
	TrainingType     string    `gorm:"size:30;not null;default:regular" json:"training_type"`
	TrainingDate     time.Time `json:"training_date"`
	DurationHours    int       `gorm:"not null;default:1" json:"duration_hours"`
	Trainer          string    `gorm:"size:50;not null;default:''" json:"trainer"`
	Location         string    `gorm:"size:255;not null;default:''" json:"location"`
	ContentSummary   string    `gorm:"type:text" json:"content_summary"`
	ParticipantIDs   JSONList  `gorm:"type:json" json:"participant_ids"`
	AssessmentMethod string    `gorm:"size:30;not null;default:笔试" json:"assessment_method"`
	PassRate         float64   `gorm:"type:decimal(5,2);not null;default:0" json:"pass_rate"`
	CreatedAt        time.Time `json:"created_at"`
}

// TableName 指定表名。
func (SafetyTraining) TableName() string { return "safety_trainings" }

// Snapshot 返回培训数据的独立副本，切断与原对象的切片共享。
func (t *SafetyTraining) Snapshot() *SafetyTraining {
	cp := *t
	// ParticipantIDs 是唯一的引用类型字段，必须深拷贝，否则副本仍与原对象共享底层数组。
	if t.ParticipantIDs != nil {
		dup := make(JSONList, len(t.ParticipantIDs))
		copy(dup, t.ParticipantIDs)
		cp.ParticipantIDs = dup
	}
	return &cp
}

// FilterParticipantIDs 返回匹配前缀的列表，不修改入参切片。
func FilterParticipantIDs(ids JSONList, prefix string) JSONList {
	// 使用独立底层数组，避免 ids[:0] 复用入参数组导致过滤时改写源数据。
	result := make(JSONList, 0, len(ids))
	for _, id := range ids {
		if len(prefix) == 0 || strings.HasPrefix(id, prefix) {
			result = append(result, id)
		}
	}
	return result
}
