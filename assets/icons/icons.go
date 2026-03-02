// Package icons embeds basketball tool palette PNG icons.
// Replace any .png file in this directory and rebuild to update icons.
// Icons should be 64x64 white on transparent background.
package icons

import "embed"

//go:embed *.png
var FS embed.FS
