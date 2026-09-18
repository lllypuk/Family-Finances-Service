package holding_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/holding"
)

func TestValidKind(t *testing.T) {
	tests := []struct {
		side holding.Side
		kind holding.Kind
		want bool
	}{
		{holding.SideAsset, holding.KindCash, true},
		{holding.SideAsset, holding.KindDeposit, true},
		{holding.SideAsset, holding.KindInvestment, true},
		{holding.SideAsset, holding.KindProperty, true},
		{holding.SideAsset, holding.KindVehicle, true},
		{holding.SideAsset, holding.KindOther, true},
		{holding.SideAsset, holding.KindMortgage, false},
		{holding.SideAsset, holding.KindLoan, false},
		{holding.SideAsset, holding.KindCreditCard, false},
		{holding.SideLiability, holding.KindMortgage, true},
		{holding.SideLiability, holding.KindLoan, true},
		{holding.SideLiability, holding.KindCreditCard, true},
		{holding.SideLiability, holding.KindOther, true},
		{holding.SideLiability, holding.KindProperty, false},
		{holding.SideLiability, holding.KindCash, false},
		{holding.SideAsset, "", false},
		{"equity", holding.KindOther, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, holding.ValidKind(tt.side, tt.kind), "%s/%s", tt.side, tt.kind)
	}
}

func TestValidSide(t *testing.T) {
	assert.True(t, holding.ValidSide(holding.SideAsset))
	assert.True(t, holding.ValidSide(holding.SideLiability))
	assert.False(t, holding.ValidSide("Asset"))
}

func TestNormalizeName(t *testing.T) {
	name, err := holding.NormalizeName("  Квартира ")
	require.NoError(t, err)
	assert.Equal(t, "Квартира", name)

	_, err = holding.NormalizeName(" \t ")
	require.ErrorIs(t, err, holding.ErrNameEmpty)

	_, err = holding.NormalizeName(strings.Repeat("я", holding.MaxNameLength+1))
	require.ErrorIs(t, err, holding.ErrNameLong)
}
