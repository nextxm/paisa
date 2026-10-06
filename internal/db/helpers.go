package db

import (
	"database/sql"
	"encoding/json"
	"time"

	sqlcdb "github.com/ananthakumaran/paisa/internal/db/sqlc"
	"gorm.io/gorm"
)

func Queries(gdb *gorm.DB) *sqlcdb.Queries {
	conn := gdb.ConnPool
	if gdb.Statement != nil && gdb.Statement.ConnPool != nil {
		conn = gdb.Statement.ConnPool
	}
	return sqlcdb.New(conn)
}

func BoolFlag(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func JSONStringArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	data, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func NullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func NullTime(value time.Time) sql.NullTime {
	return sql.NullTime{Time: value, Valid: !value.IsZero()}
}

func NullBool(value bool) sql.NullBool {
	return sql.NullBool{Bool: value, Valid: true}
}

func NullInt64(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: true}
}
