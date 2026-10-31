package models

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

func TestExecutionMetadata_RoundTrip(t *testing.T) {
	exm := ExecutionDomainToModel(&pkgmodels.Execution{Metadata: map[string]any{"retry_of": "abc"}})
	assert.Equal(t, "abc", exm.Metadata["retry_of"])
	assert.Equal(t, "abc", ExecutionModelToDomain(exm).Metadata["retry_of"])

	assert.Nil(t, ExecutionDomainToModel(&pkgmodels.Execution{}).Metadata)
	assert.Nil(t, ExecutionModelToDomain(&ExecutionModel{}).Metadata)
}
