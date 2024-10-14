package request

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
)

const BudgetAdjustmentAPI = "/v1/budgetadjustments"

type HTTPError struct {
	// StatusCode is the HTTP status code of the response.
	// Example: 200, 400, 404, 500.
	StatusCode int `json:"statusCode"`
	// Method is the HTTP method used to make the request.
	// Example: "GET", "POST", "PUT", "DELETE".
	Method string `json:"method"`
	// URL is the URL of the API endpoint that was called.
	URL string `json:"url"`
	// TraceID is an optional, unique identifier that can be used to trace the error in Nobl9 platform.
	// Contact [Nobl9 support] if you need help debugging the issue based on the TraceID.
	//
	// [Nobl9 support]: https://nobl9.com/contact/support
	TraceID string `json:"traceId,omitempty"`
	// Errors is a list of errors returned by the API.
	// At least one error is always guaranteed to be set.
	// At the very minimum it will contain just the [APIError.Title].
	Errors []APIError `json:"errors"`
}

// APIError defines a standardized format for error responses across all Nobl9 public services.
// It ensures that errors are communicated in a consistent and structured manner,
// making it easier for developers to handle and debug issues.
type APIError struct {
	// Title is a human-readable summary of the error. It is required.
	Title string `json:"title"`
	// Code is an application-specific error code. It is optional.
	Code string `json:"code,omitempty"`
	// Source provides additional context for the source of the error. It is optional.
	// Source *APIErrorSource `json:"source,omitempty"`
	PropertyName string `json:"propertyName,omitempty"`
	// PropertyValue is an optional value of the property that caused the error.
	PropertyValue string `json:"propertyValue,omitempty"`
}

func (e *APIError) ToString() string {
	if e.PropertyName != "" && e.PropertyValue != "" {
		return fmt.Sprintf(
			"  -'%s' with value '%s'\n    -%s\n",
			// e.Source.PropertyName,
			// e.Source.PropertyValue,
			e.PropertyName,
			e.PropertyValue,
			e.Title,
		)
	}
	return e.Title
}

type APIErrors struct {
	Errors []APIError `json:"errors"`
}

func (e *APIErrors) ToString() string {
	var buffer bytes.Buffer

	buffer.WriteString("\n")
	for _, error := range e.Errors {
		buffer.WriteString(error.ToString())
	}

	return buffer.String()
}

// APIErrorSource provides additional context for the source of the [APIError].
type APIErrorSource struct {
	// PropertyName is an optional name of the property that caused the error.
	// It can be a JSON path or a simple property name.
	PropertyName string `json:"propertyName,omitempty"`
	// PropertyValue is an optional value of the property that caused the error.
	PropertyValue string `json:"propertyValue,omitempty"`
}

func DoRequest(
	client *sdk.Client,
	ctx context.Context,
	method, endpoint string,
	values url.Values,
) ([]byte, *APIErrors, error) {
	var err error
	req, err := client.CreateRequest(ctx, method, endpoint, nil, values, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, nil, err
	}
	var body []byte
	if resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
	}
	if resp.StatusCode >= 300 {
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
			return nil,
				&APIErrors{
					Errors: []APIError{{Title: string(bytes.TrimSpace(body))}},
				},
				nil
		}
		respErr := APIErrors{}
		if err := json.Unmarshal(body, &respErr); err != nil {
			return nil, nil, errors.Errorf(
				"bad response (status: %d): %s",
				resp.StatusCode,
				string(body),
			)
		}
		return body, &respErr, nil
	}
	return body, nil, nil
}
