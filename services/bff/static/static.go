package static

import (
	"embed"
	"io/fs"
)

//go:embed index.html
var files embed.FS

func Files() fs.FS {
	return files
}
