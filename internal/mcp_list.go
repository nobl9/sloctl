package internal

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/printer"
)

type MCPListCmd struct {
	server     *server.MCPServer
	printer    *printer.Printer
	reqCounter int
}

func (r *MCPCmd) NewMCPListCmd() *cobra.Command {
	mcpListCmd := &MCPListCmd{
		printer: printer.NewPrinter(printer.Config{OutputFormat: printer.YAMLFormat}),
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all MCP primitives supported by the server, such as tools or resources",
		Example: "sloctl mcp list",
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := newMCPServer(cmd, r.client)
			if err := srv.RegisterToolsAndResources(); err != nil {
				return err
			}
			mcpListCmd.server = srv.server
			return mcpListCmd.listToolsAndResources(cmd)
		},
	}

	mcpListCmd.printer.MustRegisterFlags(cmd)

	return cmd
}

func (c *MCPListCmd) listToolsAndResources(cmd *cobra.Command) error {
	toolsResponse, err := c.runMCPRequest(cmd.Context(), "tools/list")
	if err != nil {
		return err
	}
	tools, err := mcpResponseToResult[mcp.ListToolsResult](toolsResponse)
	if err != nil {
		return err
	}

	resourcesResponse, err := c.runMCPRequest(cmd.Context(), "resources/list")
	if err != nil {
		return err
	}
	resources, err := mcpResponseToResult[mcp.ListResourcesResult](resourcesResponse)
	if err != nil {
		return err
	}

	return c.printer.Print(struct {
		Tools     any `json:"tools"`
		Resources any `json:"resources"`
	}{
		Tools:     tools.Tools,
		Resources: resources.Resources,
	})
}

func (c *MCPListCmd) runMCPRequest(ctx context.Context, method string) (*mcp.JSONRPCResponse, error) {
	c.reqCounter++
	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      mcp.NewRequestId(c.reqCounter),
		Request: mcp.Request{
			Method: method,
		},
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to encode '%s' request", method)
	}
	msg := c.server.HandleMessage(ctx, reqData)
	resp, ok := msg.(mcp.JSONRPCResponse)
	if !ok {
		return nil, errors.Errorf("expected MCP server response to be of type: %T, but was: %T", resp, msg)
	}
	return &resp, nil
}

func mcpResponseToResult[T any](resp *mcp.JSONRPCResponse) (v T, err error) {
	v, ok := resp.Result.(T)
	if !ok {
		return v, errors.Errorf("expected MCP server response results to be of type: %T, but was: %T", v, resp.Result)
	}
	return v, nil
}
