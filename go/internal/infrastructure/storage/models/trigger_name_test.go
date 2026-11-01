package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

func TestTriggerModel_DisplayNameAndDescription(t *testing.T) {
	stored := &TriggerModel{
		Name:        "Nightly",
		Description: "Runs at night",
		Config:      JSONBMap{"name": "legacy", "description": "legacy desc"},
	}
	assert.Equal(t, "Nightly", stored.DisplayName())
	assert.Equal(t, "Runs at night", stored.DisplayDescription())

	legacy := &TriggerModel{Config: JSONBMap{"name": "legacy", "description": "legacy desc"}}
	assert.Equal(t, "legacy", legacy.DisplayName())
	assert.Equal(t, "legacy desc", legacy.DisplayDescription())

	wrongType := &TriggerModel{Config: JSONBMap{"name": 42, "description": true}}
	assert.Empty(t, wrongType.DisplayName())
	assert.Empty(t, wrongType.DisplayDescription())

	empty := &TriggerModel{}
	assert.Empty(t, empty.DisplayName())
	assert.Empty(t, empty.DisplayDescription())
}

func TestTriggerMappers_NameAndDescription(t *testing.T) {
	tr := &pkgmodels.Trigger{
		ID:          uuid.NewString(),
		WorkflowID:  uuid.NewString(),
		Name:        "Hourly sync",
		Description: "Pulls data",
		Type:        pkgmodels.TriggerTypeManual,
		Config:      map[string]any{},
	}
	m := TriggerDomainToModel(tr)
	assert.Equal(t, "Hourly sync", m.Name)
	assert.Equal(t, "Pulls data", m.Description)

	back := TriggerModelToDomain(m)
	assert.Equal(t, "Hourly sync", back.Name)
	assert.Equal(t, "Pulls data", back.Description)

	legacy := TriggerModelToDomain(&TriggerModel{Config: JSONBMap{"name": "old", "description": "old desc"}})
	assert.Equal(t, "old", legacy.Name)
	assert.Equal(t, "old desc", legacy.Description)
}
