//go:build tools
// +build tools

// https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md
package tools

import (
	_ "github.com/air-verse/air"
	_ "github.com/google/wire/cmd/wire"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
	_ "github.com/swaggo/swag/cmd/swag"
	_ "go.uber.org/mock/mockgen"
)
