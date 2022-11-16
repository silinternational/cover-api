package public

import (
	"embed"
	"io/fs"

	"github.com/gobuffalo/buffalo"
)

//go:embed *
var files embed.FS

// FS returns a buffalo FS object that extends embed.FS, but unfortunately hides things like ReadFile
func FS() fs.FS {
	return buffalo.NewFS(files, "public")
}

// EFS returns the low-level embed.FS for situations that do not need the buffalo extensions
func EFS() embed.FS {
	return files
}
