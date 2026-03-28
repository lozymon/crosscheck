package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

// Response is a fully-buffered HTTP response.
type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

func newResponse(r *http.Response) (*Response, error) {
	defer func() { _ = r.Body.Close() }()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	headers := make(map[string]string, len(r.Header))

	for k := range r.Header {
		headers[k] = r.Header.Get(k)
	}

	return &Response{
		Status:  r.StatusCode,
		Headers: headers,
		Body:    body,
	}, nil
}

// Get extracts a value from the JSON body using a gjson path.
// Accepts JSONPath-style "$.foo.bar" (leading "$." is stripped) or plain "foo.bar".
func (r *Response) Get(path string) gjson.Result {
	path = strings.TrimPrefix(path, "$.")

	return gjson.GetBytes(r.Body, path)
}

// BodyString returns the response body as a string.
func (r *Response) BodyString() string {
	return string(r.Body)
}
