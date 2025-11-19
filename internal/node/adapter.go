package node

import "context"

// DataAdapter transforms a source value R (e.g., *http.Response or []byte)
// into a target value T to be used as node output data.
type DataAdapter[R any, T any] interface {
	Adapt(ctx context.Context, from R) (T, error)
}
