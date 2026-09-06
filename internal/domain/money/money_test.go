package money_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/money"
)

const floatDelta = 0.0001

func TestMinor_Abs(t *testing.T) {
	assert.Equal(t, money.Minor(0), money.Minor(0).Abs())
	assert.Equal(t, money.Minor(99), money.Minor(99).Abs())
	assert.Equal(t, money.Minor(99), money.Minor(-99).Abs())
}

func TestMinor_Percent(t *testing.T) {
	tests := []struct {
		name     string
		value    money.Minor
		total    money.Minor
		expected float64
	}{
		{name: "Zero total gives zero, not NaN", value: 100, total: 0, expected: 0},
		{name: "Small share keeps fraction", value: 1, total: 3, expected: 33.3333},
		{name: "Full share", value: 500, total: 500, expected: 100},
		{name: "Over total", value: 750, total: 500, expected: 150},
		{name: "Negative value", value: -25, total: 100, expected: -25},
		{name: "One kopeck of a large total", value: 1, total: 1_000_000, expected: 0.0001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.value.Percent(tt.total), floatDelta)
		})
	}
}

func TestMinor_DivRound(t *testing.T) {
	tests := []struct {
		name     string
		value    money.Minor
		divisor  int64
		expected money.Minor
	}{
		{name: "Exact division", value: 100, divisor: 4, expected: 25},
		{name: "Rounds down below half", value: 100, divisor: 3, expected: 33},
		{name: "Rounds up above half", value: 200, divisor: 3, expected: 67},
		{name: "Half rounds away from zero", value: 5, divisor: 2, expected: 3},
		{name: "Negative half rounds away from zero", value: -5, divisor: 2, expected: -3},
		{name: "Negative divisor", value: 5, divisor: -2, expected: -3},
		{name: "Both negative", value: -5, divisor: -2, expected: 3},
		{name: "Negative below half", value: -100, divisor: 3, expected: -33},
		{name: "Odd divisor half boundary", value: 7, divisor: 5, expected: 1},
		{name: "Odd divisor above half", value: 8, divisor: 5, expected: 2},
		{name: "Zero divisor gives zero", value: 100, divisor: 0, expected: 0},
		{name: "Zero value", value: 0, divisor: 7, expected: 0},
		{name: "Divisor one", value: -42, divisor: 1, expected: -42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value.DivRound(tt.divisor))
		})
	}
}

func TestMinor_DivRound_SymmetricAroundZero(t *testing.T) {
	for value := money.Minor(-20); value <= 20; value++ {
		assert.Equal(t, -value.DivRound(3), (-value).DivRound(3), "value %d", value)
	}
}

func TestMinor_MarshalJSON(t *testing.T) {
	payload := struct {
		AmountMinor money.Minor `json:"amount_minor"`
	}{AmountMinor: -12345}

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{"amount_minor":-12345}`, string(encoded))
}

func TestMinor_SumKeepsKopecks(t *testing.T) {
	var total money.Minor
	for range 3 {
		total += 33
	}

	assert.Equal(t, money.Minor(99), total)
}

func TestMinor_ValueAndScan(t *testing.T) {
	value, err := money.Minor(12345).Value()
	require.NoError(t, err)
	assert.Equal(t, int64(12345), value)

	var m money.Minor
	require.NoError(t, m.Scan(int64(-99)))
	assert.Equal(t, money.Minor(-99), m)

	require.NoError(t, m.Scan(nil))
	assert.Equal(t, money.Minor(0), m)

	require.Error(t, m.Scan("12345"))
	// float64 в колонке INTEGER означает старую схему: усечь его молча нельзя.
	require.Error(t, m.Scan(123.45))
}

func TestMinor_Share(t *testing.T) {
	assert.InDelta(t, 0.25, money.Minor(2_500).Share(10_000), 1e-9)
	assert.InDelta(t, -0.5, money.Minor(-5_000).Share(10_000), 1e-9)
	assert.InDelta(t, 0.0, money.Minor(100).Share(0), 1e-9)
}

func TestMinor_String(t *testing.T) {
	assert.Equal(t, "12345", money.Minor(12345).String())
	assert.Equal(t, "-1", money.Minor(-1).String())
	assert.Equal(t, "0", money.Minor(0).String())
}
