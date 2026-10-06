package job

import "time"

// Job stores a background task snapshot in SQLite so it survives process restarts.
type Job struct {
	ID             string         `gorm:"primaryKey;size:36"`
	Type           string         `gorm:"not null;index"`
	Status         string         `gorm:"not null;index"`
	Error          string         `gorm:"type:text"`
	Details        []string       `gorm:"serializer:json;type:text"`
	ItemsCompleted int            `json:"items_completed"`
	TotalItems     int            `json:"total_items"`
	Metadata       map[string]any `gorm:"serializer:json;type:text"`
	Payload        string         `gorm:"type:text"`
	CreatedAt      time.Time      `gorm:"not null;index"`
	StartedAt      *time.Time
	FinishedAt     *time.Time
	UpdatedAt      time.Time
}

func (Job) TableName() string {
	return "jobs"
}
