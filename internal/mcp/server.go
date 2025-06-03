package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/nobl9/nobl9-go/sdk"
)

func StartServer() error {
	s := server.NewMCPServer("Nobl9", "0.1.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
	)

	r := mcp.NewResource("nobl9://{project}/slos",
		"Nobl9 SLOs",
		mcp.WithResourceDescription("Nobl9 SLOs are used to measure the reliability of your services. Get "),
		mcp.WithMIMEType("application/yaml"),
	)
	s.AddResource(r, getSLOs)

	t := mcp.NewTool("get_slos",
		mcp.WithDescription("Get SLO or multiple SLOs"),
		mcp.WithString("name",
			mcp.Description("The SLO name"),
			mcp.DefaultString(""),
		),
		mcp.WithString("project",
			mcp.Required(),
			mcp.DefaultString("*"),
			mcp.Description("The project in which to find SLOs."),
		),
		mcp.WithBoolean("all_projects",
			mcp.Required(),
			mcp.DefaultBool(false),
			mcp.Description("The project in which to find SLOs."),
		),
		mcp.WithString("format",
			mcp.Required(),
			mcp.DefaultString("yaml"),
			mcp.Description("The output format. Supported formats: yaml, json."),
		),
	)
	s.AddTool(t, getSLOsTool)

	t = mcp.NewTool("get_projects",
		mcp.WithDescription("Get Projects"),
		mcp.WithString("name",
			mcp.Description("The Project name"),
		),
		mcp.WithString("format",
			mcp.Required(),
			mcp.DefaultString("yaml"),
			mcp.Description("The output format for. Supported formats: yaml, json."),
		),
	)
	s.AddTool(t, getProjectsTool)

	t = mcp.NewTool("get_status",
		mcp.WithDescription("Get SLO budget status"),
		mcp.WithString("name",
			mcp.Description("The SLO name"),
		),
		mcp.WithString("project_name",
			mcp.Description("The Project name"),
		),
	)
	s.AddTool(t, getSLOStatus)

	t = mcp.NewTool("get_ebs",
		mcp.WithDescription("Get Error Budget Status for multiple SLOs"),
		mcp.WithString("project",
			mcp.Description("The Project name"),
		),
		mcp.WithString("session_id",
			mcp.Description("The web browser sessionID"),
			mcp.Required(),
		),
	)
	s.AddTool(t, getEBSTool)

	t = mcp.NewTool("apply",
		mcp.WithDescription("Apply changes to nobl9"),
		mcp.WithString("file_name",
			mcp.Description("The file to apply"),
			mcp.Required(),
		),
	)
	s.AddTool(t, applyTool)

	t = mcp.NewTool("replay",
		mcp.WithDescription("Replay slo"),
		mcp.WithString("slo",
			mcp.Description("The SLO name"),
			mcp.Required(),
		),
		mcp.WithString("project",
			mcp.Description("The Project name"),
			mcp.Required(),
		),
		mcp.WithString("from",
			mcp.Description("The start time for the replay (RFC3339 format)"),
			mcp.Required(),
		),
	)
	s.AddTool(t, replay)

	slog.Info("Starting Nobl9 MCP server", "version", "0.1.0")
	return server.ServeStdio(s)
}

func getSLOs(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	project := req.Params.URI

	outputFile := fmt.Sprintf("%s_%d.yaml", "get_slos", time.Now().Unix())
	outFile, err := os.Create(outputFile)
	if err != nil {
		slog.Error("Failed to create output file", "error", err)
		return nil, err
	}

	cmd := exec.Command("sloctl", "get", "slo", "--project", project)
	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		slog.Error("Failed to run sloctl command", "error", err)
		return nil, err
	}

	return nil, nil
}

func getSLOsTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.Params.Arguments["project"].(string)
	format := req.Params.Arguments["format"].(string)
	name := req.Params.Arguments["name"].(string)

	outputFile := fmt.Sprintf("%s_%d.%s", "get_slos", time.Now().Unix(), format)
	outFile, err := os.Create(outputFile)
	if err != nil {
		slog.Error("Failed to create output file", "error", err)
		return nil, err
	}
	defer outFile.Close()

	cmd := exec.Command("sloctl", "get", "slo", "--project", project, "-o", format, name)
	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		slog.Error("Failed to run sloctl command", "error", err)
		return nil, err
	}

	buff := bytes.Buffer{}
	buff.WriteString("SLOs retrieved successfully. Output written to " + outputFile + "\n")

	objs, err := sdk.ReadObjects(context.Background(), outputFile)
	if err != nil {
		slog.Error("Failed to read SLO objects", "error", err)
		return nil, err
	}

	buff.WriteString(fmt.Sprintf("Retrieved %d SLOs:\n", len(objs)))
	for _, obj := range objs {
		buff.WriteString(obj.GetName())
		buff.WriteString("\n")
	}

	return mcp.NewToolResultText(buff.String()), nil
}

func getProjectsTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	format := req.Params.Arguments["format"].(string)

	outputFile := fmt.Sprintf("%s_%d.%s", "get_projects", time.Now().Unix(), format)
	outFile, err := os.Create(outputFile)
	if err != nil {
		slog.Error("Failed to create output file", "error", err)
		return nil, err
	}
	defer outFile.Close()

	cmd := exec.Command("sloctl", "get", "project", "-o", format)
	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		slog.Error("Failed to run sloctl command", "error", err)
		return nil, err
	}

	buff := bytes.Buffer{}
	buff.WriteString("SLOs retrieved successfully. Output written to " + outputFile + "\n")

	objs, err := sdk.ReadObjects(context.Background(), outputFile)
	if err != nil {
		slog.Error("Failed to read Project objects", "error", err)
		return nil, err
	}

	buff.WriteString(fmt.Sprintf("Retrieved %d Projects:\n", len(objs)))
	for _, obj := range objs {
		buff.WriteString(obj.GetName())
		buff.WriteString("\n")
	}

	return mcp.NewToolResultText(buff.String()), nil
}

func getSLOStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sloName, okSlo := req.Params.Arguments["name"].(string)
	if !okSlo || sloName == "" {
		return nil, fmt.Errorf("SLO name is required")
	}
	projectName, okProject := req.Params.Arguments["project_name"].(string)
	if !okProject || projectName == "" {
		return nil, fmt.Errorf("project name is required")
	}

	client, err := sdk.DefaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Nobl9 client: %w", err)
	}
	status, err := client.SLOStatusAPI().V2().GetSLO(ctx, projectName, sloName)
	if err != nil {
		return nil, fmt.Errorf("failed to get SLO status: %w", err)
	}

	// TODO encode status to json
	b, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SLO status: %w", err)
	}

	return mcp.NewToolResultText(string(b)), nil
}

func getEBSTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.Params.Arguments["project"].(string)
	sessionId := req.Params.Arguments["session_id"].(string)

	r, err := getEBS(ctx, sessionId, project)
	if err != nil {
		return nil, fmt.Errorf("failed to get EBS: %w", err)
	}
	outputFile := fmt.Sprintf("%s_%d.yaml", "get_ebs", time.Now().Unix())
	outFile, err := os.Create(outputFile)
	if err != nil {
		slog.Error("Failed to create output file", "error", err)
		return nil, err
	}
	outFile.WriteString(r)

	return mcp.NewToolResultText("Retrieved Error Budget status. Saved it in file " + outputFile), nil
}

func applyTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fileName := req.Params.Arguments["file_name"].(string)

	cmd := exec.Command("sloctl", "apply", "-f", fileName)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to apply changes: %w", err)
	}

	return mcp.NewToolResultText(string(b)), nil
}

func replay(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sloName := req.Params.Arguments["slo"].(string)
	projectName := req.Params.Arguments["project"].(string)
	from := req.Params.Arguments["from"].(string)

	cmd := exec.Command("sloctl", "replay", sloName, "--project", projectName, "--from", from)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to replay SLO: %w; %s", err, string(b))
	}

	return mcp.NewToolResultText(string(b)), nil
}
