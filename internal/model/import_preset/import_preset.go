package import_preset

import (
	"fmt"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PresetType string

const (
	Builtin PresetType = "builtin"
	Custom  PresetType = "custom"
)

// Preset stores user-defined import preset configuration in SQLite.
type Preset struct {
	ID              uint              `gorm:"primaryKey"`
	Name            string            `gorm:"uniqueIndex;not null"`
	ColumnMappings  map[string]string `gorm:"serializer:json;type:text;not null"`
	DateFormat      string            `gorm:"not null"`
	DefaultAccounts map[string]string `gorm:"serializer:json;type:text;not null"`
	Delimiter       string            `gorm:"not null;default:','"`
}

// ImportPreset is the API shape returned to frontend clients.
type ImportPreset struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	ColumnMappings  map[string]string `json:"column_mappings"`
	DateFormat      string            `json:"date_format"`
	DefaultAccounts map[string]string `json:"default_accounts"`
	Delimiter       string            `json:"delimiter"`
	PresetType      PresetType        `json:"preset_type"`
}

func All(db *gorm.DB) ([]ImportPreset, error) {
	var custom []Preset
	if err := db.Order("name asc").Find(&custom).Error; err != nil {
		return nil, err
	}

	presets := make([]ImportPreset, 0, len(custom)+len(BuiltinPresets()))
	for _, p := range custom {
		presets = append(presets, toCustomAPI(p))
	}
	presets = append(presets, BuiltinPresets()...)
	sort.SliceStable(presets, func(i, j int) bool {
		if presets[i].PresetType != presets[j].PresetType {
			return presets[i].PresetType < presets[j].PresetType
		}
		return presets[i].Name < presets[j].Name
	})
	return presets, nil
}

func Upsert(db *gorm.DB, preset ImportPreset) (ImportPreset, error) {
	if isBuiltin(preset.Name) {
		return ImportPreset{}, fmt.Errorf("cannot overwrite builtin preset %q", preset.Name)
	}

	entity := Preset{
		Name:            preset.Name,
		ColumnMappings:  preset.ColumnMappings,
		DateFormat:      preset.DateFormat,
		DefaultAccounts: preset.DefaultAccounts,
		Delimiter:       preset.Delimiter,
	}

	if entity.ColumnMappings == nil {
		entity.ColumnMappings = map[string]string{}
	}
	if entity.DefaultAccounts == nil {
		entity.DefaultAccounts = map[string]string{}
	}
	if entity.Delimiter == "" {
		entity.Delimiter = ","
	}

	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"column_mappings", "date_format", "default_accounts", "delimiter",
		}),
	}).Create(&entity).Error
	if err != nil {
		return ImportPreset{}, err
	}

	var saved Preset
	if err := db.Where("name = ?", entity.Name).First(&saved).Error; err != nil {
		return ImportPreset{}, err
	}
	return toCustomAPI(saved), nil
}

func Delete(db *gorm.DB, name string) error {
	if isBuiltin(name) {
		return fmt.Errorf("cannot delete builtin preset %q", name)
	}
	return db.Where("name = ?", name).Delete(&Preset{}).Error
}

func BuiltinPresets() []ImportPreset {
	return []ImportPreset{
		{
			ID:   buildID("Generic Bank CSV", Builtin),
			Name: "Generic Bank CSV",
			ColumnMappings: map[string]string{
				"date": "A", "description": "B", "debit": "C", "credit": "D", "balance": "E",
			},
			DateFormat:      "YYYY-MM-DD",
			DefaultAccounts: map[string]string{"asset": "Assets:Checking"},
			Delimiter:       ",",
			PresetType:      Builtin,
		},
		{
			ID:   buildID("Chase Credit Card CSV", Builtin),
			Name: "Chase Credit Card CSV",
			ColumnMappings: map[string]string{
				"date": "A", "description": "B", "category": "C", "type": "D", "amount": "E",
			},
			DateFormat:      "MM/DD/YYYY",
			DefaultAccounts: map[string]string{"liability": "Liabilities:CreditCard:Chase"},
			Delimiter:       ",",
			PresetType:      Builtin,
		},
		{
			ID:   buildID("SBI Account Statement CSV", Builtin),
			Name: "SBI Account Statement CSV",
			ColumnMappings: map[string]string{
				"date": "A", "description": "C", "debit": "E", "credit": "F",
			},
			DateFormat:      "D MMM YYYY",
			DefaultAccounts: map[string]string{"asset": "Assets:Checking:SBI"},
			Delimiter:       ",",
			PresetType:      Builtin,
		},
		{
			ID:   buildID("ICICI Credit Card CSV", Builtin),
			Name: "ICICI Credit Card CSV",
			ColumnMappings: map[string]string{
				"date": "A", "description": "C", "amount": "F", "dr_cr": "G",
			},
			DateFormat:      "DD/MM/YYYY",
			DefaultAccounts: map[string]string{"liability": "Liabilities:CreditCard:ICICI"},
			Delimiter:       ",",
			PresetType:      Builtin,
		},
	}
}

func toCustomAPI(p Preset) ImportPreset {
	return ImportPreset{
		ID:              buildID(p.Name, Custom),
		Name:            p.Name,
		ColumnMappings:  p.ColumnMappings,
		DateFormat:      p.DateFormat,
		DefaultAccounts: p.DefaultAccounts,
		Delimiter:       p.Delimiter,
		PresetType:      Custom,
	}
}

func buildID(name string, presetType PresetType) string {
	return fmt.Sprintf("%s:%s", presetType, name)
}

func isBuiltin(name string) bool {
	for _, preset := range BuiltinPresets() {
		if preset.Name == name {
			return true
		}
	}
	return false
}
