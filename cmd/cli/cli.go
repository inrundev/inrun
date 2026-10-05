package cli

import (
	"context"

	"github.com/orkspace/orkestra/cmd/cli/cmdutil"
	"github.com/orkspace/orkestra/pkg/konfig"
)

// Execute runs the root command. Which commands exist depends on the build:
// commands_dev.go, commands_runtime.go and commands_gateway.go each import
// only the command packages for their binary.
func Execute(k *konfig.Konfig, c context.Context) {
	cmdutil.Execute(k, c)
}
