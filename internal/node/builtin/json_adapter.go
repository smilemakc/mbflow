package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// JSONResponseAdapter decodes an HTTP response body as JSON into T.
type JSONResponseAdapter[T any] struct{}

func (a *JSONResponseAdapter[T]) Adapt(_ context.Context, resp *http.Response) (T, error) {
	var zero T
	if resp.Body == nil {
		return zero, nil
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	var out T
	if len(b) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return zero, err
	}
	return out, nil
}
