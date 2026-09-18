package account_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/account"
)

func TestNormalizeName(t *testing.T) {
	name, err := account.NormalizeName("  Тинькофф ")
	require.NoError(t, err)
	assert.Equal(t, "Тинькофф", name)

	name, err = account.NormalizeName(strings.Repeat("я", account.MaxNameLength))
	require.NoError(t, err)
	assert.Len(t, []rune(name), account.MaxNameLength)

	_, err = account.NormalizeName(" \t ")
	require.ErrorIs(t, err, account.ErrNameEmpty)

	_, err = account.NormalizeName(strings.Repeat("я", account.MaxNameLength+1))
	require.ErrorIs(t, err, account.ErrNameLong)
}
