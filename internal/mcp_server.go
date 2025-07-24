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
	cmd    *cobra.Command
	client *sdk.Client
	server *server.MCPServer
}

func (s mcpServer) Start() error {
	// Register tools and resources for each kind.
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
	s.server.AddTool(t, s.SLOStatusTool)

	t = mcp.NewTool("get_ebs",
		mcp.WithDescription("Get Error Budget Status for multiple SLOs"),
		mcp.WithString("project",
			mcp.Description("The Project name"),
		),
	)
	s.server.AddTool(t, s.EBSTool)

	t = mcp.NewTool("apply",
		mcp.WithDescription("Apply changes to nobl9"),
		mcp.WithString("file_name",
			mcp.Description("The file to apply"),
			mcp.Required(),
		),
	)
	s.server.AddTool(t, s.ApplyTool)

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
	s.server.AddTool(t, s.ReplayTool)

	slog.Info("Starting Nobl9 MCP server", "version", "0.1.0")
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
		s.getObjectsToolHandler(kind),
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

		filename := fmt.Sprintf("%s_%d.yaml", objects[0].GetKind(), time.Now().Unix())
		if err := s.writeObjectsToFile(filename, "yaml", objects); err != nil {
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

func (s mcpServer) getObjectsToolHandler(kind manifest.Kind) server.ToolHandlerFunc {
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

		filename := fmt.Sprintf("get_%s_%d.%s", objects[0].GetKind(), time.Now().Unix(), format)
		if err := s.writeObjectsToFile(filename, format, objects); err != nil {
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

func (s mcpServer) writeObjectsToFile(outFilename, formatStr string, objects []manifest.Object) error {
	format, err := manifest.ParseObjectFormat(strings.ToUpper(formatStr))
	if err != nil {
		return err
	}
	outFile, err := os.Create(outFilename) // #nosec: G304
	if err != nil {
		slog.Error("Failed to create output file",
			slog.String("error", err.Error()),
			slog.String("filename", outFilename))
		return err
	}
	defer func() { _ = outFile.Close() }()
	return sdk.EncodeObjects(objects, outFile, format)
}

func (s mcpServer) SLOStatusTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func (s mcpServer) ApplyTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func (s mcpServer) ReplayTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sloName := req.Params.Arguments["slo"].(string)
	projectName := req.Params.Arguments["project"].(string)
	fromStr := req.Params.Arguments["from"].(string)

	fromValue := new(flags.TimeValue)
	if err := fromValue.Set(fromStr); err != nil {
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
		sloName: sloName,
		project: projectName,
	}

	if err := replayCmd.Run(s.cmd); err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(out.String()), nil
}

func (s mcpServer) EBSTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.Params.Arguments["project"].(string)

	r, err := s.getEBS(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to get EBS: %w", err)
	}
	outputFile := fmt.Sprintf("%s_%d.yaml", "get_ebs", time.Now().Unix())
	outFile, err := os.Create(outputFile) // #nosec: G304
	if err != nil {
		slog.Error("Failed to create output file", "error", err)
		return nil, err
	}
	defer func() { _ = outFile.Close() }()
	_, _ = outFile.WriteString(r)

	return mcp.NewToolResultText("Retrieved Error Budget status. Saved it in file " + outputFile), nil
}

func (s mcpServer) getEBS(ctx context.Context, project string) (string, error) {
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

	httpClient := http.Client{
		Timeout: time.Second * 20,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	// TODO call sloctl to refresh token, redo the request
	// if resp.StatusCode == http.StatusUnauthorized {
	// }
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(b), nil
}
