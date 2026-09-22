// Package migrations embuti os arquivos SQL de migration no binário, para que
// cmd/api e cmd/admin possam aplicá-los sem depender do filesystem em produção.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
