// Package db embeds the goose migrations, so the binary carries its own schema.
package db

import "embed"

// Migrations holds every file under migrations/.
//
//go:embed migrations/*.sql
var Migrations embed.FS
