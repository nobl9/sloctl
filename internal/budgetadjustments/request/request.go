package request

import (
	"context"
	_ "embed"
	"io"
	"net/url"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
)

const BudgetAdjustmentAPI = "/v1/budgetadjustments"

func DoRequest(
	client *sdk.Client,
	ctx context.Context,
	method, endpoint string,
	values url.Values,
) ([]byte, error) {
	req, err := client.CreateRequest(ctx, method, endpoint, nil, values, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("bad response (status: %d): %s", resp.StatusCode, string(data))
	}
	return io.ReadAll(resp.Body)
}
