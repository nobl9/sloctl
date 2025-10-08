package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"

	"github.com/nobl9/sloctl/internal/flags"
	"github.com/nobl9/sloctl/internal/printer"
)

func newMCPServer(cmd *cobra.Command, client *sdk.Client) mcpServer {
	return mcpServer{
		cmd:    cmd,
		client: client,
		server: server.NewMCPServer("Nobl9", "0.1.0",
			server.WithResourceCapabilities(true, true),
			server.WithPromptCapabilities(true),
			server.WithToolCapabilities(true),
		),
	}
}

type mcpServer struct {
	cmd     *cobra.Command
	client  *sdk.Client
	server  *server.MCPServer
	tempDir string
}

type mcpToolArguments struct {
	Project  string `json:"project"`
	Format   string `json:"format"`
	From     string `json:"from"`
	Name     string `json:"name"`
	FileName string `json:"file_name"`
}

func (s mcpServer) RegisterToolsAndResources() error {
	// nolint: lll
	for _, object := range []struct {
		Kind                manifest.Kind
		ResourceDescription string
	}{
		{
			Kind:                manifest.KindAgent,
			ResourceDescription: "Agents are middleware between the Nobl9 app and external data sources. They gather metrics data and send it to Nobl9.",
		},
		{
			Kind:                manifest.KindAlertMethod,
			ResourceDescription: "Alert methods define how notifications are sent to external tools or REST endpoints when alerts are triggered.",
		},
		{
			Kind:                manifest.KindAlertPolicy,
			ResourceDescription: "Alert policies define when to trigger an alert via the configured alert method. They accept up to three conditions that must all be met to trigger an alert.",
		},
		{
			Kind:                manifest.KindAlert,
			ResourceDescription: "Alerts represent specific notification events generated when an alert policy's conditions are met for a particular SLO.",
		},
		{
			Kind:                manifest.KindAlertSilence,
			ResourceDescription: "Alert silences allow you to turn off alerts that are attached to an SLO for a defined period of time.",
		},
		{
			Kind:                manifest.KindAnnotation,
			ResourceDescription: "Annotations let Nobl9 users add notes to their metrics. They are placed at specific time points on SLO graphs.",
		},
		{
			Kind:                manifest.KindDataExport,
			ResourceDescription: "Data export configurations allow you to export your data from Nobl9 to external storage systems like S3 or GCS.",
		},
		{
			Kind:                manifest.KindDirect,
			ResourceDescription: "Direct configurations gather metrics data directly from external sources based on provided credentials, without requiring server-side installation.",
		},
		{
			Kind:                manifest.KindProject,
			ResourceDescription: "Projects are the primary grouping of resources in Nobl9. They provide organizational structure for SLOs, services, and other resources.",
		},
		{
			Kind:                manifest.KindRoleBinding,
			ResourceDescription: "Role bindings define the relationship between users and roles, managing access permissions within projects or organizations.",
		},
		{
			Kind:                manifest.KindService,
			ResourceDescription: "Services are high-level groupings of SLOs that represent logical service endpoints like APIs, databases, or applications.",
		},
		{
			Kind:                manifest.KindSLO,
			ResourceDescription: "SLOs are target values or ranges for services measured by service level indicators. They define reliability expectations in terms of customer experience.",
		},
		{
			Kind:                manifest.KindUserGroup,
			ResourceDescription: "User groups facilitate managing user access by synchronizing groups from Identity Providers like Azure AD or Okta.",
		},
		{
			Kind:                manifest.KindBudgetAdjustment,
			ResourceDescription: "Budget adjustments define future periods where planned maintenance, releases, and similar activities won't affect your SLO budget.",
		},
		{
			Kind:                manifest.KindReport,
			ResourceDescription: "Reports allow you to define Error Budget Status, SLO History, and System Health Review report types for monitoring and analysis.",
		},
	} {
		s.addToolForObject(object.Kind)
		s.addResourceForObject(object.Kind, object.ResourceDescription)
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
	s.server.AddTool(t, mcp.NewTypedToolHandler(s.SLOStatusTool))

	t = mcp.NewTool("get_ebs",
		mcp.WithDescription("Get Error Budget Status for multiple SLOs"),
		mcp.WithString("project",
			mcp.Description("The Project name"),
		),
	)
	s.server.AddTool(t, mcp.NewTypedToolHandler(s.EBSTool))

	t = mcp.NewTool("apply",
		mcp.WithDescription("Apply changes to nobl9"),
		mcp.WithString("file_name",
			mcp.Description("The file to apply"),
			mcp.Required(),
		),
	)
	s.server.AddTool(t, mcp.NewTypedToolHandler(s.ApplyTool))

	t = mcp.NewTool("replay",
		mcp.WithDescription("Replay slo"),
		mcp.WithString("name",
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
	s.server.AddTool(t, mcp.NewTypedToolHandler(s.ReplayTool))

	return nil
}

func (s mcpServer) Start() error {
	slog.Info("Starting Nobl9 MCP server", "version", getBuildVersion())
	return server.ServeStdio(s.server)
}

func (s mcpServer) addToolForObject(kind manifest.Kind) {
	kindPlural := pluralForKind(kind)
	opts := []mcp.ToolOption{
		mcp.WithDescription("Get " + kindPlural),
		mcp.WithString("name",
			mcp.Description(fmt.Sprintf("The %s name", kind)),
		),
		mcp.WithString("format",
			mcp.Required(),
			mcp.DefaultString("yaml"),
			mcp.Enum("yaml", "json"),
			mcp.Description("The output format"),
		),
	}
	if objectKindSupportsProjectFlag(kind) {
		opts = append(opts, mcp.WithString("project",
			mcp.Required(),
			mcp.DefaultString("*"),
			mcp.Description(fmt.Sprintf("The project in which to find %s.", kindPlural)),
		))
	}
	s.server.AddTool(
		mcp.NewTool("get_"+strings.ToLower(kindPlural), opts...),
		mcp.NewTypedToolHandler(s.getObjectsToolHandler(kind)),
	)
}

func (s mcpServer) addResourceForObject(kind manifest.Kind, description string) {
	kindPlural := pluralForKind(kind)
	var uri string
	if objectKindSupportsProjectFlag(kind) {
		uri = fmt.Sprintf("nobl9://{project}/%s", strings.ToLower(kindPlural))
	} else {
		uri = fmt.Sprintf("nobl9://%s", strings.ToLower(kindPlural))
	}
	r := mcp.NewResource(uri,
		fmt.Sprintf("Nobl9 %s", kindPlural),
		mcp.WithResourceDescription(description),
		mcp.WithMIMEType("application/yaml"),
	)
	s.server.AddResource(r, s.getObjectsResourceHandler(kind))
}

func (s mcpServer) getObjectsResourceHandler(kind manifest.Kind) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		uri := req.Params.URI
		kindPlural := strings.ToLower(pluralForKind(kind))

		header := http.Header{}
		var project string
		if objectKindSupportsProjectFlag(kind) {
			// URI format: nobl9://{project}/{kindPlural}
			uriWithoutScheme := strings.TrimPrefix(uri, "nobl9://")
			project = strings.TrimSuffix(uriWithoutScheme, "/"+kindPlural)
			header.Set(sdk.HeaderProject, project)
		}

		objects, err := s.client.Objects().V1().Get(ctx, kind, header, nil)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to fetch %s", pluralForKind(kind)), "error", err, "project", project)
			return nil, err
		}

		if len(objects) == 0 {
			var text string
			if project != "" {
				text = fmt.Sprintf("Found no %s in project %s\n", pluralForKind(kind), project)
			} else {
				text = fmt.Sprintf("Found no %s\n", pluralForKind(kind))
			}
			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "text/plain",
					Text:     text,
				},
			}, nil
		}

		var buf bytes.Buffer
		if err = sdk.EncodeObjects(objects, &buf, manifest.ObjectFormatYAML); err != nil {
			return nil, errors.Wrapf(err, "failed to encode %s", pluralForKind(kind))
		}
		filename, err := s.writeCachedFile(objects[0].GetKind().String(), "yaml", buf.String())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write %s to a file", pluralForKind(kind))
		}

		resourceContents := make([]mcp.ResourceContents, 0, len(objects)+1)
		resourceContents = append(resourceContents, &mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "text/plain",
			Text: fmt.Sprintf("Retrieved %d %s. Output written to: %s\n",
				len(objects), pluralForKind(kind), filename),
		})
		for _, obj := range objects {
			var buf bytes.Buffer
			if err := sdk.EncodeObject(obj, &buf, manifest.ObjectFormatYAML); err != nil {
				slog.Error("Failed to encode object", "error", err, "object", obj)
				continue
			}
			resourceContents = append(resourceContents, &mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "application/yaml",
				Text:     buf.String(),
			})
		}
		return resourceContents, nil
	}
}

func (s mcpServer) getObjectsToolHandler(kind manifest.Kind) mcp.TypedToolHandlerFunc[mcpToolArguments] {
	return func(ctx context.Context, _ mcp.CallToolRequest, args mcpToolArguments) (*mcp.CallToolResult, error) {
		header := http.Header{}
		query := url.Values{}
		if args.Project != "" {
			header.Set(sdk.HeaderProject, args.Project)
		}
		if args.Name != "" {
			query.Set(objectsV1.QueryKeyName, args.Name)
		}
		objects, err := s.client.Objects().V1().Get(ctx, kind, header, query)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to fetch %s", pluralForKind(kind)), "error", err)
			return nil, err
		}
		if len(objects) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("Found no %s\n", pluralForKind(kind))), nil
		}

		var buf bytes.Buffer
		format, err := manifest.ParseObjectFormat(strings.ToUpper(args.Format))
		if err != nil {
			return nil, err
		}
		if err = sdk.EncodeObjects(objects, &buf, format); err != nil {
			return nil, errors.Wrapf(err, "failed to encode %s", pluralForKind(kind))
		}
		filename, err := s.writeCachedFile(fmt.Sprintf("get_%s", objects[0].GetKind()), args.Format, buf.String())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write %s to a file", pluralForKind(kind))
		}

		result := strings.Builder{}
		result.WriteString(fmt.Sprintf(
			"Retrieved %d %s. Output written to: %s\n", len(objects), pluralForKind(kind), filename,
		))

		for _, obj := range objects {
			result.WriteString(obj.GetName())
			result.WriteString("\n")
		}

		return mcp.NewToolResultText(result.String()), nil
	}
}

func (s mcpServer) SLOStatusTool(
	ctx context.Context,
	_ mcp.CallToolRequest,
	args mcpToolArguments,
) (*mcp.CallToolResult, error) {
	if args.Name == "" {
		return nil, fmt.Errorf("'name' argument is required")
	}
	if args.Project == "" {
		return nil, fmt.Errorf("'project' argument is required")
	}

	status, err := s.client.SLOStatusAPI().V2().GetSLO(ctx, args.Project, args.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get SLO status: %w", err)
	}

	b, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SLO status: %w", err)
	}

	return mcp.NewToolResultText(string(b)), nil
}

func (s mcpServer) ApplyTool(
	ctx context.Context,
	_ mcp.CallToolRequest,
	args mcpToolArguments,
) (*mcp.CallToolResult, error) {
	if args.FileName == "" {
		return nil, fmt.Errorf("'file_name' argument is required")
	}

	objects, err := readObjectsDefinitions(
		ctx,
		s.client.Config,
		nil,
		[]string{args.FileName},
		newFilesPrompt(s.client.Config.FilesPromptEnabled, true, s.client.Config.FilesPromptThreshold),
		false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read objects from '%s' file", args.FileName)
	}
	if err = s.client.Objects().V1().Apply(ctx, objects); err != nil {
		return nil, errors.Wrap(err, "failed to apply objects")
	}

	return mcp.NewToolResultText("The objects were successfully applied."), nil
}

func (s mcpServer) ReplayTool(
	ctx context.Context,
	_ mcp.CallToolRequest,
	args mcpToolArguments,
) (*mcp.CallToolResult, error) {
	if args.From == "" {
		return nil, fmt.Errorf("'from' argument is required")
	}
	if args.Name == "" {
		return nil, fmt.Errorf("'name' argument is required")
	}
	if args.Project == "" {
		return nil, fmt.Errorf("'project' argument is required")
	}
	fromValue := new(flags.TimeValue)
	if err := fromValue.Set(args.From); err != nil {
		return nil, err
	}

	out := new(bytes.Buffer)
	replayCmd := &ReplayCmd{
		client: s.client,
		printer: printer.NewPrinter(printer.Config{
			Output:       out,
			OutputFormat: printer.YAMLFormat,
		}),
		from:    *fromValue,
		sloName: args.Name,
		project: args.Project,
	}

	if err := replayCmd.Run(s.cmd); err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(out.String()), nil
}

func (s mcpServer) EBSTool(
	ctx context.Context,
	_ mcp.CallToolRequest,
	args mcpToolArguments,
) (*mcp.CallToolResult, error) {
	if args.Project == "" {
		return nil, fmt.Errorf("'project' argument is required")
	}
	r, err := s.getEBS(ctx, args.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to get EBS: %w", err)
	}
	outputFile, err := s.writeCachedFile("get_ebs", "yaml", r)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("Retrieved Error Budget status. Saved it in file " + outputFile), nil
}

func (s mcpServer) getEBS(ctx context.Context, project string) (string, error) {
	search := `{}`
	if project != "" {
		search = fmt.Sprintf(`{"textSearch":"%s"}`, project)
	}
	body := strings.NewReader(search)
	req, err := s.client.CreateRequest(ctx, http.MethodPost, "/dashboards/servicehealth/error-budget", nil, nil, body)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(b), nil
}

func (s mcpServer) writeCachedFile(prefix, format, content string) (string, error) {
	if s.tempDir == "" {
		// Use .nobl9 directory in current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
		}
		s.tempDir = filepath.Join(cwd, ".nobl9")

		// Create .nobl9 directory if it doesn't exist
		if err := os.MkdirAll(s.tempDir, 0o750); err != nil {
			return "", fmt.Errorf("failed to create .nobl9 directory: %w", err)
		}
	}
	filename := filepath.Join(s.tempDir, fmt.Sprintf("%s_%d.%s", prefix, time.Now().Unix(), format))
	err := os.WriteFile(filename, []byte(content), 0o600)
	if err != nil {
		slog.Error("Failed to write temp file", "error", err, "filename", filename)
		return "", err
	}
	return filename, nil
}
