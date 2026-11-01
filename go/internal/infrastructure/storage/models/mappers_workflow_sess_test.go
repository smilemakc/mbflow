package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSess_WorkflowModelToDomain_Version(t *testing.T) {
	wm := &WorkflowModel{ID: uuid.New(), Name: "wf", Version: 7, Status: "draft"}
	assert.Equal(t, 7, WorkflowModelToDomain(wm).Version)
	assert.Equal(t, 7, WorkflowFromStorage(wm).Version)

	// Round trip through the storage mapper keeps the version.
	back := WorkflowToStorage(WorkflowModelToDomain(wm), wm.ID)
	assert.Equal(t, 7, back.Version)
}
