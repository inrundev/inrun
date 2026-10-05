//go:build !runtime && !gateway

package cli

import (
	_ "github.com/inrundev/inrun/cmd/cli/cluster"
	_ "github.com/inrundev/inrun/cmd/cli/console"
	_ "github.com/inrundev/inrun/cmd/cli/gate"
	_ "github.com/inrundev/inrun/cmd/cli/generate"
	_ "github.com/inrundev/inrun/cmd/cli/migrate"
	_ "github.com/inrundev/inrun/cmd/cli/notes"
	_ "github.com/inrundev/inrun/cmd/cli/registry"
	_ "github.com/inrundev/inrun/cmd/cli/run"
	_ "github.com/inrundev/inrun/cmd/cli/scaffold"
	_ "github.com/inrundev/inrun/cmd/cli/self"
	_ "github.com/inrundev/inrun/cmd/cli/serve"
	_ "github.com/inrundev/inrun/cmd/cli/testsuite"
	_ "github.com/inrundev/inrun/cmd/cli/validate"
)
