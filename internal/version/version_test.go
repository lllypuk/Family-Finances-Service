package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/version"
)

func TestVersion_String_ReturnsLinkedValue(t *testing.T) {
	assert.Equal(t, version.Version, version.String())
	assert.NotEmpty(t, version.String())
}
