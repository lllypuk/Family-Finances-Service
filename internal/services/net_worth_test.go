//nolint:testpackage // нужен доступ к чистым функциям
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/services/dto"
)

func snap(id uuid.UUID, side holding.Side, y int, m time.Month, d int, v money.Minor) holding.SeriesRow {
	return holding.SeriesRow{HoldingID: id, Side: side, Date: date.New(y, m, d), ValueMinor: v}
}

func asset(id uuid.UUID, m time.Month, d int, v money.Minor) holding.SeriesRow {
	return snap(id, holding.SideAsset, 2026, m, d, v)
}

func liability(id uuid.UUID, m time.Month, d int, v money.Minor) holding.SeriesRow {
	return snap(id, holding.SideLiability, 2026, m, d, v)
}

func nw(month string, assets, liabilities money.Minor) dto.NetWorthMonth {
	return dto.NetWorthMonth{
		Month: month, AssetsMinor: assets, LiabilitiesMinor: liabilities, NetMinor: assets - liabilities,
	}
}

func TestFoldNetWorth(t *testing.T) {
	flat, deposit, mortgage := uuid.New(), uuid.New(), uuid.New()
	jan1 := date.New(2026, time.January, 1)
	apr30 := date.New(2026, time.April, 30)

	tests := []struct {
		name     string
		rows     []holding.SeriesRow
		from, to date.Date
		want     []dto.NetWorthMonth
	}{
		{
			name: "carries forward through empty months",
			rows: []holding.SeriesRow{asset(flat, time.January, 10, 100)},
			from: jan1, to: apr30,
			want: []dto.NetWorthMonth{nw("2026-01", 100, 0), nw("2026-02", 100, 0), nw("2026-03", 100, 0),
				nw("2026-04", 100, 0)},
		},
		{
			name: "holding is absent before its first snapshot",
			rows: []holding.SeriesRow{asset(flat, time.January, 10, 100), asset(deposit, time.March, 1, 50)},
			from: jan1, to: apr30,
			want: []dto.NetWorthMonth{nw("2026-01", 100, 0), nw("2026-02", 100, 0), nw("2026-03", 150, 0),
				nw("2026-04", 150, 0)},
		},
		{
			name: "later snapshot of the month wins",
			rows: []holding.SeriesRow{asset(flat, time.February, 1, 100), asset(flat, time.February, 20, 70)},
			from: jan1, to: date.New(2026, time.February, 28),
			want: []dto.NetWorthMonth{nw("2026-01", 0, 0), nw("2026-02", 70, 0)},
		},
		{
			name: "zero snapshot drops the contribution",
			rows: []holding.SeriesRow{
				liability(mortgage, time.January, 5, 300),
				liability(mortgage, time.February, 5, 0),
			},
			from: jan1,
			to:   date.New(2026, time.March, 31),
			want: []dto.NetWorthMonth{nw("2026-01", 0, 300), nw("2026-02", 0, 0), nw("2026-03", 0, 0)},
		},
		{
			name: "initial state before from counts from the first bucket",
			rows: []holding.SeriesRow{
				snap(flat, holding.SideAsset, 2025, time.June, 1, 1000),
				liability(mortgage, time.February, 1, 400),
			},
			from: jan1, to: date.New(2026, time.February, 28),
			want: []dto.NetWorthMonth{nw("2026-01", 1000, 0), nw("2026-02", 1000, 400)},
		},
		{
			name: "from mid-month: the bucket is the state at month end",
			rows: []holding.SeriesRow{asset(flat, time.March, 5, 100), asset(flat, time.March, 20, 200)},
			from: date.New(2026, time.March, 15), to: date.New(2026, time.March, 31),
			want: []dto.NetWorthMonth{nw("2026-03", 200, 0)},
		},
		{
			name: "snapshot exactly on from is a change inside the period",
			rows: []holding.SeriesRow{asset(flat, time.March, 15, 100)},
			from: date.New(2026, time.March, 15), to: date.New(2026, time.March, 15),
			want: []dto.NetWorthMonth{nw("2026-03", 100, 0)},
		},
		{
			name: "snapshot on the last day of the month belongs to that month",
			rows: []holding.SeriesRow{asset(flat, time.January, 31, 100)},
			from: jan1, to: date.New(2026, time.February, 28),
			want: []dto.NetWorthMonth{nw("2026-01", 100, 0), nw("2026-02", 100, 0)},
		},
		{
			name: "from equals to",
			rows: []holding.SeriesRow{asset(flat, time.June, 1, 100)},
			from: date.New(2026, time.June, 1), to: date.New(2026, time.June, 1),
			want: []dto.NetWorthMonth{nw("2026-06", 100, 0)},
		},
		{
			name: "to mid-month cuts a later snapshot",
			rows: []holding.SeriesRow{asset(flat, time.September, 1, 100), asset(flat, time.September, 25, 999)},
			from: date.New(2026, time.September, 1), to: date.New(2026, time.September, 18),
			want: []dto.NetWorthMonth{nw("2026-09", 100, 0)},
		},
		{
			name: "empty history gives zero buckets",
			from: jan1, to: date.New(2026, time.February, 1),
			want: []dto.NetWorthMonth{nw("2026-01", 0, 0), nw("2026-02", 0, 0)},
		},
		{
			name: "liabilities above assets give a negative net",
			rows: []holding.SeriesRow{asset(flat, time.January, 1, 100), liability(mortgage, time.January, 1, 500)},
			from: jan1, to: date.New(2026, time.January, 31),
			want: []dto.NetWorthMonth{nw("2026-01", 100, 500)},
		},
		{
			name: "sum of two holdings exceeds MaxAmount",
			rows: []holding.SeriesRow{
				asset(flat, time.January, 1, money.MaxAmount), asset(deposit, time.January, 1, money.MaxAmount),
			},
			from: jan1, to: date.New(2026, time.January, 31),
			want: []dto.NetWorthMonth{nw("2026-01", 2*money.MaxAmount, 0)},
		},
		{
			name: "from on the 31st keeps February",
			rows: []holding.SeriesRow{asset(flat, time.February, 10, 100)},
			from: date.New(2026, time.January, 31), to: date.New(2026, time.March, 1),
			want: []dto.NetWorthMonth{nw("2026-01", 0, 0), nw("2026-02", 100, 0), nw("2026-03", 100, 0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, foldNetWorth(tt.rows, tt.from, tt.to))
		})
	}
}

// Архивности свёртка не знает: архивную позицию в выборке держит SeriesValues (тест репозитория).
func TestFoldNetWorth_DeletedSnapshotChangesPastBuckets(t *testing.T) {
	flat := uuid.New()
	from, to := date.New(2026, time.January, 1), date.New(2026, time.March, 31)
	rows := []holding.SeriesRow{
		asset(flat, time.January, 1, 100),
		asset(flat, time.February, 1, 300),
		asset(flat, time.March, 1, 500),
	}

	before := foldNetWorth(rows, from, to)
	after := foldNetWorth([]holding.SeriesRow{rows[0], rows[2]}, from, to)

	require.Len(t, after, 3)
	assert.Equal(t, money.Minor(300), before[1].AssetsMinor)
	assert.Equal(t, money.Minor(100), after[1].AssetsMinor, "февраль снова несёт январский снимок")
	assert.Equal(t, before[2], after[2])
}
