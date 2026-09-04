package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/version"
)

func TestVersion_String_Default(t *testing.T) {
	assert.Equal(t, "dev", version.String())
}
