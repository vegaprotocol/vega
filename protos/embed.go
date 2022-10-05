package protos

import (
	"embed"
	_ "embed"
)

//go:embed generated
var Generated embed.FS
