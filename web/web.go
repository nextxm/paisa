package web

import (
	"embed"
)

//go:embed all:static
var Static embed.FS

var Index string

func init() {
	data, _ := Static.ReadFile("static/index.html")
	Index = string(data)
}
