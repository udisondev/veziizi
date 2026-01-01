// Package migrations contains database migrations embedded as filesystem.
package migrations

import "embed"

// FS contains all SQL migration files embedded in the binary.
// This allows running migrations without depending on file system paths.
//
//go:embed *.sql
var FS embed.FS
