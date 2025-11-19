package builtin

import (
	"context"
	"io"
	"net/http"
)

type RawBytesAdapter struct{}

func (a *RawBytesAdapter) Adapt(_ context.Context, resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
