//go:build tools

package build

import (
	_ "github.com/t-yuki/gocover-cobertura"
	_ "github.com/vektra/mockery/v2"
	_ "gotest.tools/gotestsum"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
