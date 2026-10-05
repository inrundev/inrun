//go:build !runtime && !gateway

package cli

import (
	_ "github.com/orkspace/orkestra/cmd/cli/cluster"
	_ "github.com/orkspace/orkestra/cmd/cli/console"
	_ "github.com/orkspace/orkestra/cmd/cli/gate"
	_ "github.com/orkspace/orkestra/cmd/cli/generate"
	_ "github.com/orkspace/orkestra/cmd/cli/migrate"
	_ "github.com/orkspace/orkestra/cmd/cli/notes"
	_ "github.com/orkspace/orkestra/cmd/cli/registry"
	_ "github.com/orkspace/orkestra/cmd/cli/run"
	_ "github.com/orkspace/orkestra/cmd/cli/scaffold"
	_ "github.com/orkspace/orkestra/cmd/cli/self"
	_ "github.com/orkspace/orkestra/cmd/cli/serve"
	_ "github.com/orkspace/orkestra/cmd/cli/testsuite"
	_ "github.com/orkspace/orkestra/cmd/cli/validate"
)
