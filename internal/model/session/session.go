package session

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// sessionDuration is how long a newly-issued session token remains valid.
const sessionDuration = 24 * time.Hour

// Session represents an authenticated user session persisted in the database.
type Session struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	Username  string    `gorm:"not null" json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Create inserts a new session for username and returns the persisted record.
func Create(db *gorm.DB, username string) (*Session, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	s := &Session{
		Token:     id.String(),
		Username:  username,
		ExpiresAt: time.Now().Add(sessionDuration),
	}
	if err := db.Create(s).Error; err != nil {
		return nil, err
	}
	return s, nil
}

// FindByToken returns the unexpired session matching token, or an error
// (including gorm.ErrRecordNotFound) if it does not exist or has expired.
func FindByToken(db *gorm.DB, token string) (*Session, error) {
	var s Session
	err := db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteByToken removes the session that matches the given token.
func DeleteByToken(db *gorm.DB, token string) error {
	return db.Where("token = ?", token).Delete(&Session{}).Error
}

// DeleteExpired removes all sessions whose expiry time is in the past.
func DeleteExpired(db *gorm.DB) error {
	return db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now()).Error
}
