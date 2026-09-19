package reconciliation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/reconciliation"
)

func TestValidBalance(t *testing.T) {
	assert.True(t, reconciliation.ValidBalance(0))
	assert.True(t, reconciliation.ValidBalance(money.MaxAmount))
	assert.True(t, reconciliation.ValidBalance(-money.MaxAmount))
	assert.False(t, reconciliation.ValidBalance(money.MaxAmount+1))
	assert.False(t, reconciliation.ValidBalance(-money.MaxAmount-1))
}
