package names_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/names"
)

func TestKey(t *testing.T) {
	assert.Equal(t, names.Key("Карта"), names.Key(" карта "))
	assert.Equal(t, "тинькофф black", names.Key("\tТИНЬКОФФ Black\n"))
	assert.NotEqual(t, names.Key("Сбер"), names.Key("Сбер 2"))
	assert.Empty(t, names.Key("   "))
}
