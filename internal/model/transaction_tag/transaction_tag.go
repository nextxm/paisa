package transaction_tag

import (
	"strings"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransactionTag struct {
	TransactionID string    `gorm:"not null;index:idx_transaction_tags_txid_tag,unique" json:"transaction_id"`
	TagName       string    `gorm:"not null;index:idx_transaction_tags_txid_tag,unique;index" json:"tag_name"`
	CreatedAt     time.Time `gorm:"not null" json:"created_at"`
}

func normalizeTag(tag string) string {
	return strings.TrimSpace(tag)
}

func ListByTransactionID(db *gorm.DB, transactionID string) ([]string, error) {
	var rows []TransactionTag
	if err := db.
		Model(&TransactionTag{}).
		Where("transaction_id = ?", transactionID).
		Order("tag_name asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return lo.Map(rows, func(row TransactionTag, _ int) string { return row.TagName }), nil
}

func ListByTransactionIDs(db *gorm.DB, transactionIDs []string) (map[string][]string, error) {
	result := map[string][]string{}
	if len(transactionIDs) == 0 {
		return result, nil
	}

	var rows []TransactionTag
	if err := db.
		Model(&TransactionTag{}).
		Where("transaction_id in ?", transactionIDs).
		Order("transaction_id asc, tag_name asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.TransactionID] = append(result[row.TransactionID], row.TagName)
	}
	return result, nil
}

func Add(db *gorm.DB, transactionID, tag string) (bool, error) {
	normalized := normalizeTag(tag)
	if normalized == "" {
		return false, nil
	}
	item := TransactionTag{TransactionID: transactionID, TagName: normalized}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&item)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func Delete(db *gorm.DB, transactionID, tag string) error {
	normalized := normalizeTag(tag)
	if normalized == "" {
		return nil
	}
	return db.Where("transaction_id = ? AND tag_name = ?", transactionID, normalized).Delete(&TransactionTag{}).Error
}

func FindTransactionIDsByTags(db *gorm.DB, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	normalizedTags := lo.FilterMap(tags, func(tag string, _ int) (string, bool) {
		n := normalizeTag(tag)
		return n, n != ""
	})
	if len(normalizedTags) == 0 {
		return nil, nil
	}

	var ids []string
	if err := db.Model(&TransactionTag{}).
		Distinct("transaction_id").
		Where("tag_name in ?", normalizedTags).
		Find(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func Autocomplete(db *gorm.DB, prefix string, limit int) ([]string, error) {
	query := db.Model(&TransactionTag{}).Select("tag_name").Group("tag_name").Order("COUNT(*) DESC, tag_name ASC")

	normalized := normalizeTag(prefix)
	if normalized != "" {
		query = query.Where("tag_name LIKE ?", normalized+"%")
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var tags []string
	if err := query.Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}
