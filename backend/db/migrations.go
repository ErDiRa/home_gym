// Package db holds the schema migrations, embedded so the API can apply them at startup.
package db

import "embed"

// Migrations are the files of migrations/ (NNNNNN_name.up.sql), applied in version order.
//
//go:embed migrations/*.sql
var Migrations embed.FS
