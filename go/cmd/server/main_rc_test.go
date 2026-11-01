package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smilemakc/mbflow/go/pkg/server"
)

type rcRunner struct {
	err error
	ran bool
}

func (r *rcRunner) Run() error {
	r.ran = true
	return r.err
}

func TestRCRun(t *testing.T) {
	r := &rcRunner{}
	assert.NoError(t, run(func(...server.Option) (*rcRunner, error) { return r, nil }))
	assert.True(t, r.ran)

	err := run(func(...server.Option) (*rcRunner, error) { return nil, errors.New("bad config") })
	assert.EqualError(t, err, "failed to create server: bad config")

	r = &rcRunner{err: errors.New("port in use")}
	err = run(func(...server.Option) (*rcRunner, error) { return r, nil })
	assert.EqualError(t, err, "server error: port in use")
}

func TestRCMain_ReportsFatalErrors(t *testing.T) {
	oldNew, oldFatal := newServer, fatal
	defer func() { newServer, fatal = oldNew, oldFatal }()

	var fatalArgs []any
	fatal = func(v ...any) { fatalArgs = v }
	newServer = func(...server.Option) (*server.Server, error) { return nil, errors.New("no database") }

	main()
	if assert.Len(t, fatalArgs, 1) {
		assert.EqualError(t, fatalArgs[0].(error), "failed to create server: no database")
	}
}
