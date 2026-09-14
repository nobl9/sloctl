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
		Use:   "mcp",
		Short: "Start the Nobl9 MCP proxy over stdio",
		Long: "Start a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) proxy over standard input " +
			"and output. The proxy uses the active sloctl context to authenticate and forwards requests to Nobl9.\n\n" +
			"Use this proxy instead of a direct HTTP connection when the MCP client communicates over stdio, " +
			"requires dynamic client registration, or should reuse credentials from the active sloctl context. " +
			"See [Nobl9 MCP server](https://docs.nobl9.com/tools-and-utilities/mcp-server).",
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
