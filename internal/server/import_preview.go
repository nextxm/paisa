package server

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/ananthakumaran/paisa/internal/model/template"
)

type ImportPreviewRequest struct {
	Template  string `json:"template" binding:"required"`
	Content   string `json:"content" binding:"required"`
	Delimiter string `json:"delimiter"`
	DryRun    *bool  `json:"dry_run,omitempty"`
}

type ImportPreviewRow struct {
	Index  int               `json:"index"`
	Row    map[string]string `json:"row"`
	Valid  bool              `json:"valid"`
	Errors []string          `json:"errors,omitempty"`
}

func PreviewImport(req ImportPreviewRequest) ([]ImportPreviewRow, error) {
	if req.DryRun != nil && !*req.DryRun {
		return nil, fmt.Errorf("import preview only supports dry_run=true")
	}

	if !templateExists(req.Template) {
		return nil, fmt.Errorf("template %q not found", req.Template)
	}

	reader := csv.NewReader(strings.NewReader(req.Content))
	reader.FieldsPerRecord = -1

	if req.Delimiter != "" {
		if len([]rune(req.Delimiter)) != 1 {
			return nil, fmt.Errorf("delimiter must be a single character")
		}
		reader.Comma = []rune(req.Delimiter)[0]
	}

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse csv content: %w", err)
	}

	expectedColumns := -1
	rows := make([]ImportPreviewRow, 0, len(records))
	for i, record := range records {
		if expectedColumns == -1 {
			expectedColumns = len(record)
		}

		errors := []string{}
		if len(record) != expectedColumns {
			errors = append(errors, fmt.Sprintf("expected %d columns, got %d", expectedColumns, len(record)))
		}

		hasValue := false
		for _, value := range record {
			if strings.TrimSpace(value) != "" {
				hasValue = true
				break
			}
		}
		if !hasValue {
			errors = append(errors, "row is empty")
		}

		row := map[string]string{
			"index": fmt.Sprintf("%d", i),
		}
		for j, value := range record {
			row[columnName(j)] = value
		}

		rows = append(rows, ImportPreviewRow{
			Index:  i,
			Row:    row,
			Valid:  len(errors) == 0,
			Errors: errors,
		})
	}

	return rows, nil
}

func templateExists(name string) bool {
	for _, t := range template.All() {
		if t.Name == name {
			return true
		}
	}
	return false
}

func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+(index%26))) + name
		index = (index / 26) - 1
	}
	return name
}
