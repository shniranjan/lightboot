package lightboot

import "embed"

// DocsFS embeds the built MkDocs documentation site.
// Run "make docs" to populate the site/ directory first.
//
//go:embed site
var DocsFS embed.FS
