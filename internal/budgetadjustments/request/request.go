package request

import (
	"bytes"
	"context"
	_ "embed"
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
) ([]byte, error) {
	var err error
	req, err := client.CreateRequest(ctx, method, endpoint, nil, values, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	var body []byte
	if resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}
	if resp.StatusCode >= 300 {
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
			return nil, newUnexpectedError(string(bytes.TrimSpace(body)))
		}
		respErr := sdk.HTTPError{}
		if err := json.Unmarshal(body, &respErr); err != nil {
			return nil, newUnexpectedError(string(bytes.TrimSpace(body)))
		}
		return body, errors.New(respErr.Error())
	}
	return body, nil
}

func newUnexpectedError(title string) error {
	return errors.New(sdk.HTTPError{
		Errors: []sdk.APIError{{Title: title}},
	}.Error())
}
