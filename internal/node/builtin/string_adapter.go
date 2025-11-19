package builtin

import (
	"context"
	"io"
	"net/http"
)

type StringAdapter struct{}

func (a *StringAdapter) Adapt(_ context.Context, resp *http.Response) (string, error) {
	if resp.Body == nil {
		return "", nil
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
