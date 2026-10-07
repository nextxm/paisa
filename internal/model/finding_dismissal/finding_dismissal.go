package finding_dismissal

import "time"

// FindingDismissal records a finding fingerprint that the user has dismissed,
// along with an optional user note explaining why it was dismissed.
type FindingDismissal struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Fingerprint string    `gorm:"uniqueIndex;not null" json:"fingerprint"`
	RuleID      string    `gorm:"index" json:"rule_id"`
	Note        string    `json:"note,omitempty"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
}
