package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nobl9/nobl9-go/sdk"
)

func getEBS(ctx context.Context, sessionId, project string) (string, error) {
	client, err := sdk.DefaultClient()
	if err != nil {
		return "", err
	}

	search := `{}`
	if project != "" {
		search = fmt.Sprintf(`{"textSearch":"%s"}`, project)
	}
	body := strings.NewReader(search)
	url := fmt.Sprintf("%s/dashboards/servicehealth/error-budget", client.Config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Config.AccessToken))
	req.Header.Set("organization", client.Config.Organization)
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
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(b), nil
}
