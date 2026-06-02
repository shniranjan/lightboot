package web

import "embed"

// Dist embeds the built Vue.js frontend assets.
//
//go:embed dist
var Dist embed.FS
