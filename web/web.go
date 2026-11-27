package web

import (
	"embed"
)

//go:embed all:static
var Static embed.FS

var Index string
var IndexV2 string

func init() {
	data, _ := Static.ReadFile("static/index.html")
	Index = string(data)
	dataV2, _ := Static.ReadFile("static/v2/index.html")
	IndexV2 = string(dataV2)
}
