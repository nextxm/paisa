package web

import (
	"embed"
)

//go:embed all:static
var Static embed.FS

var Index string
var IndexV1 string

func init() {
	data, _ := Static.ReadFile("static/index.html")
	Index = string(data)
	dataV1, _ := Static.ReadFile("static/v1/index.html")
	IndexV1 = string(dataV1)
}
