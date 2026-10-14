// Package migrations встраивает SQL-миграции в бинарник,
// чтобы мигратору не нужны были файлы на диске (удобно в Docker).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
