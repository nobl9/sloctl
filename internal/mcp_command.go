package internal

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/sdk"

	"github.com/nobl9/sloctl/internal/mcp"
)

type MCPServerCmd struct {
	client *sdk.Client
	// TODO add server port
}

func (r *RootCmd) NewMCPServer() *cobra.Command {
	mcp := &MCPServerCmd{}

	cmd := &cobra.Command{
		Use:              "mcp",
		Short:            "Start the MCP server",
		Long:             "",
		Example:          "sloctl mcp",
		Args:             positionalArgsCondition,
		PersistentPreRun: nil,
		RunE:             mcp.Run,
	}

	return cmd
}

func (m MCPServerCmd) Run(cmd *cobra.Command, args []string) error {
	return mcp.StartServer()
}
