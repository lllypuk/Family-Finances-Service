package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/version"
)

func TestVersion_String_ReturnsLinkedValue(t *testing.T) {
	// go test не передаёт -ldflags, поэтому здесь всегда значение по умолчанию.
	assert.Equal(t, "dev", version.String())
	assert.Equal(t, version.Version, version.String())
}
