package cli

import (
	"context"

	"github.com/inrundev/inrun/cmd/cli/cmdutil"
	"github.com/inrundev/inrun/pkg/config"
)

// Execute runs the root command. Which commands exist depends on the build:
// commands_dev.go, commands_runtime.go and commands_gateway.go each import
// only the command packages for their binary.
func Execute(k *config.Config, c context.Context) {
	cmdutil.Execute(k, c)
}
