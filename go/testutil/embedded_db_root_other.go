//go:build !unix

package testutil

import "errors"

func prepareRootRun(uint32, string, string) (string, error) {
	return "", errors.New("running embedded PostgreSQL as root is supported on unix only")
}
