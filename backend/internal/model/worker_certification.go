package model

import (
	"errors"
	"net/url"
	"time"
)

// WorkerCertification 人员资质实体。
type WorkerCertification struct {
	ID           uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint64            `gorm:"not null;index" json:"user_id"`
	CertType     string            `gorm:"size:50;not null;default:''" json:"cert_type"`
	CertNo       string            `gorm:"size:50;not null;default:''" json:"cert_no"`
	IssueOrg     string            `gorm:"size:100;not null;default:''" json:"issue_org"`
	IssueDate    *time.Time        `json:"issue_date"`
	ValidUntil   *time.Time        `json:"valid_until"`
	CertPhotoURL string            `gorm:"size:255;not null;default:''" json:"cert_photo_url"`
	Status       string            `gorm:"size:30;not null;default:pending;index" json:"status"`
	ReviewNotes  map[string]string `gorm:"serializer:json;type:json" json:"review_notes"`
	CreatedAt    time.Time         `json:"created_at"`
}

// CertificationReviewPolicy validates optional evidence before a review is committed.
type CertificationReviewPolicy interface {
	Validate(*WorkerCertification) error
}

// PhotoReviewPolicy validates a supplied certificate image URL.
type PhotoReviewPolicy struct{}

func (p *PhotoReviewPolicy) Validate(c *WorkerCertification) error {
	if p == nil {
		return nil
	}
	u, err := url.ParseRequestURI(c.CertPhotoURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("certificate photo URL must use http or https")
	}
	return nil
}

// CertificationReviewPolicyFor selects the policy required by this certification.
func CertificationReviewPolicyFor(c *WorkerCertification) CertificationReviewPolicy {
	if c.CertPhotoURL == "" {
		var policy *PhotoReviewPolicy
		return policy
	}
	return &PhotoReviewPolicy{}
}

// AddReviewNote records a durable review fact.
func (c *WorkerCertification) AddReviewNote(key, value string) {
	c.ReviewNotes[key] = value
}

// TableName 指定表名。
func (WorkerCertification) TableName() string { return "worker_certifications" }
