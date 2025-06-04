package internal

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/sdk"
)

type MCPServerCmd struct {
	client *sdk.Client
	// TODO add server port
}

func (r *RootCmd) NewMCPServer() *cobra.Command {
	mcpCmd := &MCPServerCmd{}

	cmd := &cobra.Command{
		Use:     "mcp",
		Short:   "Start the MCP server",
		Long:    "",
		Example: "sloctl mcp",
		Args:    positionalArgsCondition,
		PersistentPreRun: func(*cobra.Command, []string) {
			mcpCmd.client = r.GetClient()
		},
		RunE: func(*cobra.Command, []string) error {
			return newMCPServer(mcpCmd.client).Start()
		},
	}

	return cmd
}
