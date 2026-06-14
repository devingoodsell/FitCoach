// Package migrations embeds the ordered SQL migration files so the binary can
// run them on boot or via the `migrate` subcommand without shipping loose files.
package migrations

import "embed"

// FS holds every *.sql migration, consumed by platform/db.Migrate.
//
//go:embed *.sql
var FS embed.FS
