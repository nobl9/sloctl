package events

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"strings"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
)

const BudgetAdjustmentAPI = "/v1/budgetadjustments"

func DoRequest(
	client *sdk.Client,
	ctx context.Context,
	method, endpoint string,
	values url.Values,
	body io.Reader,
) ([]byte, error) {
	var err error
	req, err := client.CreateRequest(ctx, method, endpoint, nil, values, body)
	if err != nil {
		return nil, err
	}
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	var respBody []byte
	if resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}
	if resp.StatusCode >= 300 {
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
			return nil, errors.Errorf("unexpected response for API")
		}
		respErr := sdk.APIErrors{}
		if err := json.Unmarshal(respBody, &respErr); err != nil {
			return nil, err
		}
		return respBody, respErr
	}
	return respBody, nil
}
