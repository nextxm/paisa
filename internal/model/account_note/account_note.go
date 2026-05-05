package account_note

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AccountNote stores a user-supplied note for a named account.
// Account names are unique; calling Upsert on an existing name updates the note.
type AccountNote struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Account string `gorm:"uniqueIndex;not null" json:"account"`
	Note    string `gorm:"not null" json:"note"`
}

// GetAll returns all account notes ordered by account name.
func GetAll(db *gorm.DB) ([]AccountNote, error) {
	var notes []AccountNote
	if err := db.Order("account asc").Find(&notes).Error; err != nil {
		return nil, err
	}
	return notes, nil
}

// Get returns the note for the given account, or gorm.ErrRecordNotFound if absent.
func Get(db *gorm.DB, account string) (*AccountNote, error) {
	var note AccountNote
	if err := db.Where("account = ?", account).First(&note).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

// Upsert inserts or updates the note for the given account name.
// An empty note string is stored as-is; callers should use Delete to remove notes.
func Upsert(db *gorm.DB, account, note string) (*AccountNote, error) {
	an := &AccountNote{Account: account, Note: note}
	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account"}},
		DoUpdates: clause.AssignmentColumns([]string{"note"}),
	}).Create(an).Error
	if err != nil {
		return nil, err
	}
	// Re-fetch to get the ID assigned by the database (INSERT … ON CONFLICT returns
	// the existing row's ID only when we re-query).
	return Get(db, account)
}

// Delete removes the note for the given account name.
// It is not an error if the account has no note.
func Delete(db *gorm.DB, account string) error {
	return db.Where("account = ?", account).Delete(&AccountNote{}).Error
}
