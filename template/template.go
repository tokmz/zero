package template

import "embed"

//go:embed *.tmpl
var Templates embed.FS
