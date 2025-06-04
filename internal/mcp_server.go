package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
)

func newMCPServer(client *sdk.Client) mcpServer {
	return mcpServer{client}
}

type mcpServer struct {
	client *sdk.Client
}

func (s mcpServer) Start() error {
	srv := server.NewMCPServer("Nobl9", "0.1.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
	)

	r := mcp.NewResource("nobl9://{project}/slos",
		"Nobl9 SLOs",
		mcp.WithResourceDescription("Nobl9 SLOs are used to measure the reliability of your services. Get "),
		mcp.WithMIMEType("application/yaml"),
	)
	srv.AddResource(r, s.getSLOs)

	for _, tool := range []struct {
		Kind manifest.Kind
	}{
		{Kind: manifest.KindAgent},
		{Kind: manifest.KindAlertMethod},
		{Kind: manifest.KindAlertPolicy},
		{Kind: manifest.KindAlert},
		{Kind: manifest.KindAlertSilence},
		{Kind: manifest.KindAnnotation},
		{Kind: manifest.KindDataExport},
		{Kind: manifest.KindDirect},
		{Kind: manifest.KindProject},
		{Kind: manifest.KindRoleBinding},
		{Kind: manifest.KindService},
		{Kind: manifest.KindSLO},
		{Kind: manifest.KindUserGroup},
		{Kind: manifest.KindBudgetAdjustment},
		{Kind: manifest.KindReport},
	} {
		kindPlural := pluralForKind(tool.Kind)
		opts := []mcp.ToolOption{
			mcp.WithDescription("Get " + kindPlural),
			mcp.WithString("name",
				mcp.Description(fmt.Sprintf("The %s name", tool.Kind)),
			),
			mcp.WithString("format",
				mcp.Required(),
				mcp.DefaultString("yaml"),
				mcp.Enum("yaml", "json"),
				mcp.Description("The output format"),
			),
		}
		if objectKindSupportsProjectFlag(tool.Kind) {
			opts = append(opts, mcp.WithString("project",
				mcp.Required(),
				mcp.DefaultString("*"),
				mcp.Description(fmt.Sprintf("The project in which to find %s.", kindPlural)),
			))
		}
		srv.AddTool(
			mcp.NewTool("get_"+strings.ToLower(kindPlural), opts...),
			s.getObjectsHandler(tool.Kind),
		)
	}

	t := mcp.NewTool("get_status",
		mcp.WithDescription("Get SLO budget status"),
		mcp.WithString("name",
			mcp.Description("The SLO name"),
		),
		mcp.WithString("project",
			mcp.Description("The Project name"),
		),
	)
	srv.AddTool(t, s.getSLOStatus)

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
	srv.AddTool(t, s.getEBSTool)

	t = mcp.NewTool("apply",
		mcp.WithDescription("Apply changes to nobl9"),
		mcp.WithString("file_name",
			mcp.Description("The file to apply"),
			mcp.Required(),
		),
	)
	srv.AddTool(t, s.applyTool)

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
	srv.AddTool(t, s.replay)

	slog.Info("Starting Nobl9 MCP server", "version", "0.1.0")
	return server.ServeStdio(srv)
}

func (s mcpServer) getSLOs(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
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

func (s mcpServer) getObjectsHandler(kind manifest.Kind) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		format, _ := req.Params.Arguments["format"].(string)
		project, _ := req.Params.Arguments["project"].(string)
		name, _ := req.Params.Arguments["name"].(string)

		header := http.Header{}
		query := url.Values{}
		if project != "" {
			header.Set(sdk.HeaderProject, project)
		}
		if name != "" {
			query.Set(objectsV1.QueryKeyName, name)
		}
		objects, err := s.client.Objects().V1().Get(ctx, kind, header, query)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to fetch %s", pluralForKind(kind)), "error", err)
			return nil, err
		}
		if len(objects) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("Found no %s\n", pluralForKind(kind))), nil
		}

		filename, err := s.writeObjectsToFile(format, objects)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write %s to a file", pluralForKind(kind))
		}

		buf := strings.Builder{}
		buf.WriteString(fmt.Sprintf("Retrieved %d %s. Output written to: %s\n", len(objects), pluralForKind(kind), filename))

		for _, obj := range objects {
			buf.WriteString(obj.GetName())
			buf.WriteString("\n")
		}

		return mcp.NewToolResultText(buf.String()), nil
	}
}

func (s mcpServer) writeObjectsToFile(formatStr string, objects []manifest.Object) (string, error) {
	format, err := manifest.ParseObjectFormat(strings.ToUpper(formatStr))
	if err != nil {
		return "", err
	}
	outputFile := fmt.Sprintf("get_%s_%d.%s", objects[0].GetKind(), time.Now().Unix(), format)
	outFile, err := os.Create(outputFile)
	if err != nil {
		slog.Error("Failed to create output file",
			slog.String("error", err.Error()),
			slog.String("filename", outputFile))
		return "", err
	}
	defer func() { _ = outFile.Close() }()
	return outputFile, sdk.EncodeObjects(objects, outFile, format)
}

func (s mcpServer) getSLOStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sloName, okSlo := req.Params.Arguments["name"].(string)
	if !okSlo || sloName == "" {
		return nil, fmt.Errorf("SLO name is required")
	}
	projectName, okProject := req.Params.Arguments["project"].(string)
	if !okProject || projectName == "" {
		return nil, fmt.Errorf("project name is required")
	}

	status, err := s.client.SLOStatusAPI().V2().GetSLO(ctx, projectName, sloName)
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

func (s mcpServer) applyTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fileName := req.Params.Arguments["file_name"].(string)

	objects, err := readObjectsDefinitions(
		ctx,
		s.client.Config,
		nil,
		[]string{fileName},
		newFilesPrompt(s.client.Config.FilesPromptEnabled, true, s.client.Config.FilesPromptThreshold),
		false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read objects from '%s' file", fileName)
	}
	if err = s.client.Objects().V1().Apply(ctx, objects); err != nil {
		return nil, errors.Wrap(err, "failed to apply objects")
	}

	return mcp.NewToolResultText("The objects were successfully applied."), nil
}

func (s mcpServer) replay(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func (s mcpServer) getEBSTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.Params.Arguments["project"].(string)
	sessionId := req.Params.Arguments["session_id"].(string)

	r, err := s.getEBS(ctx, sessionId, project)
	if err != nil {
		return nil, fmt.Errorf("failed to get EBS: %w", err)
	}
	outputFile := fmt.Sprintf("%s_%d.yaml", "get_ebs", time.Now().Unix())
	outFile, err := os.Create(outputFile)
	if err != nil {
		slog.Error("Failed to create output file", "error", err)
		return nil, err
	}
	defer func() { _ = outFile.Close() }()
	outFile.WriteString(r)

	return mcp.NewToolResultText("Retrieved Error Budget status. Saved it in file " + outputFile), nil
}

func (s mcpServer) getEBS(ctx context.Context, sessionId, project string) (string, error) {
	search := `{}`
	if project != "" {
		search = fmt.Sprintf(`{"textSearch":"%s"}`, project)
	}
	body := strings.NewReader(search)
	url := fmt.Sprintf("%s/dashboards/servicehealth/error-budget", s.client.Config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.client.Config.AccessToken))
	req.Header.Set("organization", s.client.Config.Organization)
	req.Header.Set("n9-session-id", sessionId)

	httpClient := http.Client{
		Timeout: time.Second * 20,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		// TODO call sloctl to refresh token, redo the request
	}
	defer func() { resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(b), nil
}
