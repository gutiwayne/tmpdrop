package web

import "embed"

//go:embed index.html app.js i18n.js style.css fonts
var FS embed.FS
