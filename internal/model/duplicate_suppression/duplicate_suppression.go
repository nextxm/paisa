package duplicate_suppression

import "time"

// DuplicateSuppression records a pair of posting IDs that the user has marked
// as not a duplicate, so that the pair is excluded from future duplicate
// detection results.
type DuplicateSuppression struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	PostingID1 uint      `gorm:"column:posting_id_1;not null;index:idx_suppression_pair" json:"posting_id_1"`
	PostingID2 uint      `gorm:"column:posting_id_2;not null;index:idx_suppression_pair" json:"posting_id_2"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
}
