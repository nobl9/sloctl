package internal

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/sdk"
)

type MCPCmd struct {
	client *sdk.Client
}

func (r *RootCmd) NewMCPCmd() *cobra.Command {
	mcpCmd := &MCPCmd{}

	cmd := &cobra.Command{
		Use:     "mcp",
		Short:   "Start the MCP proxy listening on stdio",
		Long:    "This feature is experimental and subject to bugs and breaking changes!",
		Example: "sloctl mcp",
		Args:    noPositionalArgsCondition,
		PersistentPreRun: func(*cobra.Command, []string) {
			mcpCmd.client = r.GetClient()
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			srv := newMCPServer(mcpCmd.client)
			return srv.Start()
		},
	}
	return cmd
}
