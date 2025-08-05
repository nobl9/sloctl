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
		Short:   "Start the MCP server listening on stdio",
		Long:    "This feature is experimental and subject to bugs and breaking changes!",
		Example: "sloctl mcp",
		Args:    noPositionalArgsCondition,
		PersistentPreRun: func(*cobra.Command, []string) {
			mcpCmd.client = r.GetClient()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := newMCPServer(cmd, mcpCmd.client)
			if err := srv.RegisterToolsAndResources(); err != nil {
				return err
			}
			return srv.Start()
		},
	}

	cmd.AddCommand(mcpCmd.NewMCPListCmd())

	return cmd
}
